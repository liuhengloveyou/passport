package passport

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/database"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/sms"

	"github.com/jackc/pgx/v5/pgxpool"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var passportConfigFile = flag.String("passport", "./passport.conf.yaml", "配置文件路径")

func init() {
	if err := InitFromConfig(*passportConfigFile); err != nil {
		log.Println(err)
	}
}

// InitFromConfig 从配置文件加载并初始化 passport 运行环境（日志、数据库、短信等）。
func InitFromConfig(configPath string) error {
	return initEnv(configPath, nil)
}

// EnsureEnv 确保运行环境已初始化；若传入 option 则合并补充配置。
func EnsureEnv(option *protos.OptionStruct) error {
	if common.ServConfig.SessionKey == "" {
		return initEnv(*passportConfigFile, option)
	}
	if option != nil {
		if err := InitWithOption(option); err != nil {
			return err
		}
	}
	return initSmsIfNeeded()
}

func InitWithOption(option *protos.OptionStruct) error {
	if option.LogDir != "" && common.Logger == nil {
		fmt.Println("passport InitLog: ", option.LogDir, option.LogLevel)
		if err := InitLog(option.LogDir, option.LogLevel); err != nil {
			return err
		}
	}

	if common.DB == nil {
		if option.DBDriver != "" && option.DBDSN != "" {
			if err := InitDBWithDriver(option.DBDriver, option.DBDSN); err != nil {
				return err
			}
		}
	}

	if option.RedisAddr != "" && common.RedisClient == nil {
		fmt.Println("passport InitRedis: ", option.RedisAddr)
		common.ServConfig.RedisAddr = option.RedisAddr
		if err := InitRedis(option.RedisAddr); err != nil {
			return err
		}
	}

	if common.ServConfig.AvatarDir == "" {
		common.ServConfig.AvatarDir = "./avatar/"
	}
	if option.AvatarDir != "" {
		common.ServConfig.AvatarDir = option.AvatarDir
	}

	common.ServConfig.SessionStoreType = option.SessionStoreType
	common.ServConfig.ApiConf = option.ApiConf
	common.ServConfig.RootUserID = option.RootUserID
	common.ServConfig.RootTenantID = option.RootTenantID

	return nil
}

func InitLog(logDir, logLevel string) error {
	writer, _ := rotatelogs.New(
		logDir+"/passport.%Y%m%d%H%M",
		rotatelogs.WithLinkName(logDir+"/log.passport"),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)

	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return err
	}

	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

	var core zapcore.Core
	if os.Getenv("PASSPORT_LOG_TO_CONSOLE") == "true" {
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), zapcore.AddSync(writer), level),
			zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), zapcore.AddSync(os.Stdout), level),
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoder),
			zapcore.AddSync(writer),
			level)
	}

	common.Logger = zap.New(core, zap.AddCaller())
	common.Logger.Info("passport initLog OK\n")

	return nil
}

// InitDB 初始化 PostgreSQL 数据库（向后兼容）。
func InitDB(urn string) error {
	pool, err := pgxpool.New(context.Background(), urn)
	if err != nil {
		return err
	}

	if err = pool.Ping(context.Background()); err != nil {
		panic(err)
	}

	common.DBPool = pool

	postgresDB, err := database.NewPostgresDB(urn)
	if err != nil {
		return err
	}
	common.DB = postgresDB

	return nil
}

// InitDBWithDriver 使用数据库抽象层初始化数据库。
func InitDBWithDriver(driver, dsn string) error {
	driverType := database.DriverType(driver)
	db, err := database.NewDB(driverType, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	common.DB = db

	if driverType == database.DriverPostgreSQL {
		common.DBPool, err = pgxpool.New(context.Background(), dsn)
		if err != nil {
			return err
		}
	}

	if err = common.DB.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func InitRedis(addr string) error {
	common.RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	if _, err := common.RedisClient.Ping(context.Background()).Result(); err != nil {
		panic(err)
	}

	fmt.Println("passport redis inited.")
	return nil
}

func initEnv(configPath string, option *protos.OptionStruct) error {
	os.Setenv("PASSPORT_LOG_TO_CONSOLE", "true")

	gob.Register(protos.User{})
	gob.Register(protos.MapStruct{})
	gob.Register(map[string]interface{}{})

	if common.ServConfig.PidFile == "" {
		common.ServConfig.PidFile = "/tmp/passport.pid"
	}

	if err := common.InitValidate(); err != nil {
		return err
	}

	if configPath != "" {
		if err := gocommon.LoadYamlConfig(configPath, &common.ServConfig); err != nil {
			return err
		}
	}

	if len(common.ServConfig.SessionKey) == 0 {
		return fmt.Errorf("sessionKey nil.")
	}

	if err := InitWithOption(&common.ServConfig); err != nil {
		if common.IsDatabaseNotExistErr(err) {
			log.Println("InitWithOption: passport 数据库尚不存在，跳过连接（可用 -init 创建）:", err)
		} else {
			return err
		}
	}

	if option != nil {
		if err := InitWithOption(option); err != nil {
			if common.IsDatabaseNotExistErr(err) {
				log.Println("InitWithOption: passport 数据库尚不存在，跳过连接（可用 -init 创建）:", err)
			} else {
				return err
			}
		}
	}

	return initSmsIfNeeded()
}

func initSmsIfNeeded() error {
	if len(common.ServConfig.SmsDriveer) == 0 {
		return nil
	}
	if err := sms.Init(common.ServConfig.SmsDriveer, common.ServConfig.SmsConf); err != nil {
		return fmt.Errorf("sms.Init: %w", err)
	}
	return nil
}

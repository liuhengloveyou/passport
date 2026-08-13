package common

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/sessions"
	"github.com/liuhengloveyou/passport/v3/sms"

	"github.com/jackc/pgx/v5/pgxpool"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	SYS_PWD         = "When you forgive, You love. And when you love, God's light shines on you. Now, 202601229"
	SessUserInfoKey = "sess-user"
	MAX_UPLOAD_LEN  = (8 * 1024 * 1024) // 最大上传文件大小
)

var (
	passportconfile = flag.String("passport", "./passport.conf.yaml", "配置文件路径")
	ServConfig      protos.OptionStruct

	Logger      *zap.Logger
	DBPool      *pgxpool.Pool // 向后兼容，仅用于PostgreSQL
	DB          database.DB   // 新的数据库抽象接口（支持PostgreSQL和SQLite3）
	RedisClient *redis.Client
)

type NilWriter struct{}

func (p *NilWriter) Write(b []byte) (n int, err error) { return 0, nil }

func init() {
	var e error

	os.Setenv("PASSPORT_LOG_TO_CONSOLE", "true")

	gob.Register(protos.User{})
	gob.Register(protos.MapStruct{})
	gob.Register(map[string]interface{}{})

	// 默认配置参数
	ServConfig.PidFile = "/tmp/passport.pid"

	if e = InitValidate(); e != nil {
		panic(e)
	}

	if e = gocommon.LoadYamlConfig(*passportconfile, &ServConfig); e != nil {
		log.Println(e)
		return
	}

	if len(ServConfig.SessionKey) == 0 {
		panic("sessionKey nil.")
	}

	if e = InitWithOption(&ServConfig); e != nil {
		// 目标库尚未创建时（如首次 -init）允许延迟连接，由 InitSystemEnv/EnsureDatabase 建库后再连
		if isDatabaseNotExistErr(e) {
			log.Println("InitWithOption: passport 数据库尚不存在，跳过连接（可用 -init 创建）:", e)
		} else {
			log.Panic("InitWithOption ", e)
		}
	}

	if len(ServConfig.SmsDriveer) > 0 {
		if e = sms.Init(ServConfig.SmsDriveer, ServConfig.SmsConf); e != nil {
			log.Panic("sms.Init ", sms.ErrSmsDriver)
		}
	}
}

func InitWithOption(option *protos.OptionStruct) (e error) {
	if option.LogDir != "" && Logger == nil {
		fmt.Println("passport InitLog: ", option.LogDir, option.LogLevel)
		if err := InitLog(option.LogDir, option.LogLevel); err != nil {
			return e
		}
	}

	// 数据库初始化：优先使用新的DBDriver配置
	if DB == nil {
		if option.DBDriver != "" && option.DBDSN != "" {
			// 使用新的数据库配置
			if e = InitDBWithDriver(option.DBDriver, option.DBDSN); e != nil {
				return e
			}
		}
	}

	if option.RedisAddr != "" && RedisClient == nil {
		fmt.Println("passport InitRedis: ", option.RedisAddr)
		ServConfig.RedisAddr = option.RedisAddr
		if e = InitRedis(option.RedisAddr); e != nil {
			return e
		}
	}

	if ServConfig.AvatarDir == "" {
		ServConfig.AvatarDir = "./avatar/"
	}
	if option.AvatarDir != "" {
		ServConfig.AvatarDir = option.AvatarDir // 头像上传目录
	}

	ServConfig.SessionStoreType = option.SessionStoreType
	ServConfig.ApiConf = option.ApiConf
	ServConfig.RootUserID = option.RootUserID
	ServConfig.RootTenantID = option.RootTenantID

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
	if e := level.UnmarshalText([]byte(logLevel)); e != nil {
		return e
	}

	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

	var core zapcore.Core
	// 测试环境下同时输出到终端
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

	Logger = zap.New(core, zap.AddCaller())
	Logger.Info("passport initLog OK\n")

	return nil
}

// InitDB 初始化PostgreSQL数据库（向后兼容）
func InitDB(urn string) (err error) {
	DBPool, err = pgxpool.New(context.Background(), urn)
	if err != nil {
		return err
	}

	if err = DBPool.Ping(context.Background()); err != nil {
		panic(err)
	}

	// 同时设置新的DB接口（用于PostgreSQL）
	postgresDB, err := database.NewPostgresDB(urn)
	if err != nil {
		return err
	}
	DB = postgresDB

	return nil
}

// InitDBWithDriver 使用新的数据库抽象层初始化数据库
func InitDBWithDriver(driver, dsn string) (err error) {
	driverType := database.DriverType(driver)
	DB, err = database.NewDB(driverType, dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// 如果是PostgreSQL，同时设置DBPool以保持向后兼容
	if driverType == database.DriverPostgreSQL {
		DBPool, err = pgxpool.New(context.Background(), dsn)
		if err != nil {
			return err
		}
	}

	if err = DB.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Logger.Info("Database initialized", zap.String("driver", driver), zap.String("dsn", maskDSN(dsn)))
	return nil
}

func isDatabaseNotExistErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") && strings.Contains(msg, "database")
}

// maskDSN 隐藏DSN中的敏感信息（用于日志）
func maskDSN(dsn string) string {
	// 简单实现：隐藏密码部分
	// 实际使用时可以更完善
	if len(dsn) > 50 {
		return dsn[:20] + "..."
	}
	return dsn
}

// GetDialect 获取当前数据库的方言
func GetDialect() database.Dialect {
	if DB == nil {
		// 默认返回PostgreSQL方言
		return database.NewDialect(database.DriverPostgreSQL)
	}
	return database.NewDialect(DB.DriverType())
}

// NewSessionStore 创建并返回一个新的session store
// 根据配置决定使用cookie store还是memory store
func NewSessionStore() interface{} {
	// 使用和httpApi.go相同的逻辑
	switch ServConfig.SessionStoreType {
	case "mem":
		// TODO: 实现memory store
		// return sessions.NewMemStore([]byte(SYS_PWD), sessPWD[:])
		fallthrough
	default:
		sessPWD := sha256.Sum256([]byte(SYS_PWD))
		store := sessions.NewCookieStore([]byte(SYS_PWD), sessPWD[:])
		store.MaxAge(ServConfig.SessionExpire)
		return store
	}
}

func InitRedis(addr string) (err error) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if _, e := RedisClient.Ping(context.Background()).Result(); e != nil {
		panic(e)
	}

	fmt.Println("passport redis inited.")

	return nil
}

func EncryPWD(pwd string) string {
	if pwd == "" {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%v%v%v", SYS_PWD, pwd, SYS_PWD))))
}

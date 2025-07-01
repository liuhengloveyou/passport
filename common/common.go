package common

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"time"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sms"

	"github.com/jackc/pgx/v5/pgxpool"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	SYS_PWD         = "When you forgive, You love. And when you love, God's light shines on you. Now, 2021"
	SessUserInfoKey = "sess-user"
	MAX_UPLOAD_LEN  = (8 * 1024 * 1024) // 最大上传文件大小
)

var (
	passportconfile = flag.String("passport", "./passport.conf.yaml", "配置文件路径")
	ServConfig      protos.OptionStruct

	Logger      *zap.Logger
	DBPool      *pgxpool.Pool
	RedisClient *redis.Client
)

type NilWriter struct{}

func (p *NilWriter) Write(b []byte) (n int, err error) { return 0, nil }

func init() {
	var e error

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
		log.Panic("InitWithOption ", e)
	}

	if len(ServConfig.SmsDriveer) > 0 {
		if e = sms.Init(ServConfig.SmsDriveer, ServConfig.SmsConf); e != nil {
			log.Panic("sms.Init ", sms.ErrSmsDriver)
		}
	}
}

func InitWithOption(option *protos.OptionStruct) (e error) {
	if option.LogDir != "" && Logger == nil {
		if err := InitLog(option.LogDir, option.LogLevel); err != nil {
			return e
		}
	}

	if option.PostgreURN != "" && DBPool == nil {
		ServConfig.PostgreURN = option.PostgreURN
		if e = InitDB(option.PostgreURN); e != nil {
			return e
		}
	}

	if option.RedisAddr != "" && RedisClient == nil {
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
	ServConfig.RootTenantID = option.RootTenantID

	return nil
}

func InitLog(logDir, logLevel string) error {
	writer, _ := rotatelogs.New(
		logDir+"/passport.%Y%m%d%H%M",
		rotatelogs.WithLinkName("log.passport"),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)

	level := zapcore.InfoLevel
	if e := level.UnmarshalText([]byte(logLevel)); e != nil {
		return e
	}

	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoder),
		zapcore.AddSync(writer),
		level)

	Logger = zap.New(core, zap.AddCaller())
	Logger.Info("passport initLog OK\n")

	return nil
}

func InitDB(urn string) (err error) {
	DBPool, err = pgxpool.New(context.Background(), urn)
	if err != nil {
		return err
	}

	if err = DBPool.Ping(context.Background()); err != nil {
		panic(err)
	}

	return nil
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

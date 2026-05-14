package common

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
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
		panic(err)
	}

	// Logger.Info("Database initialized", zap.String("driver", driver), zap.String("dsn", maskDSN(dsn)))
	return nil
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

func InitDBTable(db *pgxpool.Pool) error {
	// 初始化数据库表
	ctx := context.Background()

	// 创建用户表
	_, err := db.Exec(ctx, `
		-- 用户表
		CREATE TABLE IF NOT EXISTS users 
		(
		uid BIGSERIAL PRIMARY KEY,
		tenant_id BIGINT NOT NULL DEFAULT 0,
		nickname VARCHAR(64) UNIQUE,
		cellphone VARCHAR(11) UNIQUE,
		email VARCHAR(255) UNIQUE,
		wx_openid VARCHAR(64) UNIQUE,
		password VARCHAR(512) NOT NULL,
		avatar_url VARCHAR(255),
		gender SMALLINT,
		addr VARCHAR(1024),
		province VARCHAR(64),
		city VARCHAR(64),
		ext JSONB,
		create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		login_time TIMESTAMPTZ
		);
		-- ALTER TABLE IF EXISTS public.users OWNER to pcdn;
		-- DROP INDEX IF EXISTS public.tenant_id;
		CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id) WITH (deduplicate_items=True);
		-- 替换原来的 ALTER SEQUENCE users_uid_seq RESTART WITH 10000;
		DO $$
		BEGIN
			IF (SELECT last_value FROM users_uid_seq) < 10000 THEN
				ALTER SEQUENCE users_uid_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建用户表失败: %w", err)
	}

	// 创建租户表
	_, err = db.Exec(ctx, `
		-- 租户表
		CREATE TABLE IF NOT EXISTS tenants (
			id BIGSERIAL PRIMARY KEY,
			uid BIGINT NOT NULL DEFAULT 0,
			tenant_name VARCHAR(255) NOT NULL UNIQUE,
			tenant_type VARCHAR(45) NOT NULL DEFAULT '',
			info JSONB,
			configuration JSONB,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_tenants_tenant_name ON tenants(tenant_name);
		-- 替换原来的 ALTER SEQUENCE tenants_id_seq RESTART WITH 10000;
		DO $$
		BEGIN
			IF (SELECT last_value FROM tenants_id_seq) < 10000 THEN
				ALTER SEQUENCE tenants_id_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建租户表失败: %w", err)
	}

	// 创建权限表
	_, err = db.Exec(ctx, `
		-- 权限表
		CREATE TABLE IF NOT EXISTS permission (
			id BIGSERIAL PRIMARY KEY,
			tenant_id BIGINT NOT NULL,
			domain VARCHAR(128) NOT NULL,
			title VARCHAR(128) NOT NULL,
			value VARCHAR(256) NOT NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (tenant_id, domain, title),
			UNIQUE (value, domain, tenant_id)
		);
		CREATE INDEX IF NOT EXISTS idx_permission_tenant_id ON permission(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_permission_domain ON permission(domain);
		-- 替换原来的 ALTER SEQUENCE permission_id_seq RESTART WITH 10000;
		DO $$
		BEGIN
			IF (SELECT last_value FROM permission_id_seq) < 10000 THEN
				ALTER SEQUENCE permission_id_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建权限表失败: %w", err)
	}

	// 创建部门表
	_, err = db.Exec(ctx, `
		-- 部门表
		CREATE TABLE IF NOT EXISTS departments (
			id BIGSERIAL PRIMARY KEY,
			parent_id BIGINT NOT NULL DEFAULT 0,
			uid BIGINT NOT NULL,
			tenant_id BIGINT NOT NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			name VARCHAR(16) NOT NULL,
			config JSONB,
			UNIQUE (tenant_id, name)
		);
		CREATE INDEX IF NOT EXISTS idx_departments_tenant_id ON departments(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id);
		CREATE INDEX IF NOT EXISTS idx_departments_uid ON departments(uid);
		-- 替换原来的 ALTER SEQUENCE departments_id_seq RESTART WITH 10000;
		DO $$
		BEGIN
			IF (SELECT last_value FROM departments_id_seq) < 10000 THEN
				ALTER SEQUENCE departments_id_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建部门表失败: %w", err)
	}

	// 创建用户闭包表
	_, err = db.Exec(ctx, `
		-- 用户闭包表
		CREATE TABLE IF NOT EXISTS user_closure (
			ancestor_id BIGINT NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
			descendant_id BIGINT NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
			depth INT NOT NULL CHECK (depth >= 0),
			PRIMARY KEY (ancestor_id, descendant_id)
		);
		CREATE INDEX IF NOT EXISTS idx_user_closure_ancestor ON user_closure(ancestor_id);
		CREATE INDEX IF NOT EXISTS idx_user_closure_descendant ON user_closure(descendant_id);
	`)
	if err != nil {
		return fmt.Errorf("创建用户闭包表失败: %w", err)
	}

	// 创建租户闭包表
	_, err = db.Exec(ctx, `
		-- 租户闭包表
		CREATE TABLE IF NOT EXISTS tenant_closure (
			ancestor_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			descendant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			depth INT NOT NULL CHECK (depth >= 0),
			PRIMARY KEY (ancestor_id, descendant_id)
		);
		CREATE INDEX IF NOT EXISTS idx_tenant_closure_tenant_id ON tenant_closure(ancestor_id);
		CREATE INDEX IF NOT EXISTS idx_tenant_closure_ancestor_id ON tenant_closure(descendant_id);
	`)
	if err != nil {
		return fmt.Errorf("创建租户闭包表失败: %w", err)
	}

	return nil
}

func EncryPWD(pwd string) string {
	if pwd == "" {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%v%v%v", SYS_PWD, pwd, SYS_PWD))))
}

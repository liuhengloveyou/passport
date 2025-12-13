package face

import (
	"context"
	"os"
	"testing"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/database"
	"github.com/liuhengloveyou/passport/protos"
	"go.uber.org/zap"
)

// TestDBConfig 测试数据库配置
type TestDBConfig struct {
	Name     string
	Driver   database.DriverType
	DSN      string
	SetupSQL string // 初始化表结构的SQL
}

// GetTestDBConfigs 获取所有测试数据库配置
func GetTestDBConfigs(t *testing.T) []TestDBConfig {
	configs := []TestDBConfig{}

	// SQLite3配置（使用文件数据库）
	sqlite3DSN := "/tmp/passport.db"
	if os.Getenv("SQLITE3_TEST_DB") != "" {
		// 如果设置了环境变量，使用指定的路径
		sqlite3DSN = os.Getenv("SQLITE3_TEST_DB")
	}
	// 清理旧数据库（每次测试使用新数据库）
	os.Remove(sqlite3DSN)

	configs = append(configs, TestDBConfig{
		Name:   "SQLite3",
		Driver: database.DriverSQLite3,
		DSN:    sqlite3DSN,
		SetupSQL: `
			CREATE TABLE IF NOT EXISTS users (
				uid INTEGER PRIMARY KEY AUTOINCREMENT,
				tenant_id INTEGER NOT NULL DEFAULT 0,
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
				ext TEXT,
				create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				login_time TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
		`,
	})

	// PostgreSQL配置（如果提供了连接字符串）
	pgDSN := os.Getenv("POSTGRES_TEST_DSN")
	if pgDSN == "" {
		// 默认测试连接（可以根据实际情况修改）
		pgDSN = "postgres://lh:lhisroot@127.0.0.1:5432/passport?sslmode=disable"
	}

	configs = append(configs, TestDBConfig{
		Name:   "PostgreSQL",
		Driver: database.DriverPostgreSQL,
		DSN:    pgDSN,
		SetupSQL: `
				CREATE TABLE IF NOT EXISTS users (
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
				CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
				-- 重置序列
				DO $$
				BEGIN
					IF EXISTS (SELECT 1 FROM pg_sequences WHERE sequencename = 'users_uid_seq') THEN
						IF (SELECT last_value FROM users_uid_seq) < 10000 THEN
							ALTER SEQUENCE users_uid_seq RESTART WITH 10000;
						END IF;
					END IF;
				END $$;
			`,
	})

	return configs
}

// SetupTestDB 设置测试数据库
func SetupTestDB(t *testing.T, config TestDBConfig) (database.DB, func()) {
	// 初始化日志
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	// 创建数据库连接
	db, err := database.NewDB(config.Driver, config.DSN)
	if err != nil {
		t.Skipf("跳过 %s 测试: 无法连接数据库: %v", config.Name, err)
		return nil, nil
	}

	ctx := context.Background()

	// 测试连接
	if err := db.Ping(ctx); err != nil {
		t.Skipf("跳过 %s 测试: 无法ping数据库: %v", config.Name, err)
		db.Close()
		return nil, nil
	}

	// 初始化表结构
	if config.SetupSQL != "" {
		// 对于PostgreSQL，需要按语句分割执行
		if config.Driver == database.DriverPostgreSQL {
			// PostgreSQL支持多语句执行
			if _, err := db.Exec(ctx, config.SetupSQL); err != nil {
				t.Logf("警告: 初始化 %s 表结构失败: %v (可能表已存在)", config.Name, err)
			}
		} else {
			// SQLite3需要逐句执行
			statements := []string{
				`CREATE TABLE IF NOT EXISTS users (
					uid INTEGER PRIMARY KEY AUTOINCREMENT,
					tenant_id INTEGER NOT NULL DEFAULT 0,
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
					ext TEXT,
					create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					login_time TIMESTAMP
				)`,
				`CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id)`,
			}
			for _, stmt := range statements {
				if _, err := db.Exec(ctx, stmt); err != nil {
					t.Logf("警告: 执行SQL失败: %v", err)
				}
			}
		}
	}

	// 设置全局数据库连接（用于测试）
	common.DB = db
	if config.Driver == database.DriverPostgreSQL {
		// 为了向后兼容，也设置DBPool
		// 这里需要从DSN创建pgxpool，但为了简化，我们跳过
		// 实际使用中，service层应该使用common.DB而不是common.DBPool
	}

	// 初始化common配置
	if common.ServConfig.SessionKey == "" {
		common.ServConfig.SessionKey = "test-session-key"
	}
	if common.Logger == nil {
		common.Logger = logger
	}

	// 清理函数
	cleanup := func() {
		// 清理测试数据（可选）
		// ctx := context.Background()
		// db.Exec(ctx, "DELETE FROM users WHERE uid >= 10000")
		db.Close()
	}

	return db, cleanup
}

// RunWithDBs 使用所有配置的数据库运行测试
func RunWithDBs(t *testing.T, testFunc func(t *testing.T, db database.DB, dbName string)) {
	configs := GetTestDBConfigs(t)

	if len(configs) == 0 {
		t.Skip("没有可用的测试数据库配置")
		return
	}

	for _, config := range configs {
		t.Run(config.Name, func(t *testing.T) {
			db, cleanup := SetupTestDB(t, config)
			if db == nil {
				return
			}
			defer cleanup()

			testFunc(t, db, config.Name)
		})
	}
}

// InitTestOptions 初始化测试选项
func InitTestOptions(db database.DB, dbName string) *protos.OptionStruct {
	options := &protos.OptionStruct{
		LogDir:     "./logs",
		LogLevel:   "debug",
		SessionKey: "test-session-key",
	}

	// 根据数据库类型设置配置
	if db.DriverType() == database.DriverPostgreSQL {
		options.DBDriver = "postgres"
		// 从环境变量获取DSN
		if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
			options.DBDSN = dsn
		} else {
			options.DBDSN = "postgres://postgres:postgres@localhost:5432/passport_test?sslmode=disable"
		}
	} else {
		options.DBDriver = "sqlite3"
		options.DBDSN = ":memory:"
	}

	return options
}

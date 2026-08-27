package common

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/liuhengloveyou/passport/v4/database"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/sessions"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	SYS_PWD         = "When you forgive, You love. And when you love, God's light shines on you. Now, 202601229"
	SessUserInfoKey = "sess-user"
	MAX_UPLOAD_LEN  = (8 * 1024 * 1024) // 最大上传文件大小
)

var (
	ServConfig protos.OptionStruct

	Logger      *zap.Logger
	DBPool      *pgxpool.Pool // 向后兼容，仅用于PostgreSQL
	DB          database.DB   // 新的数据库抽象接口（支持PostgreSQL和SQLite3）
	RedisClient *redis.Client
)

type NilWriter struct{}

func (p *NilWriter) Write(b []byte) (n int, err error) { return 0, nil }

func IsDatabaseNotExistErr(err error) bool {
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

func EncryPWD(pwd string) string {
	if pwd == "" {
		return ""
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%v%v%v", SYS_PWD, pwd, SYS_PWD))))
}

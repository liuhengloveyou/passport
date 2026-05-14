package database

import (
	"context"
	"fmt"
)

// DriverType 数据库驱动类型
type DriverType string

const (
	DriverPostgreSQL DriverType = "postgres"
	DriverSQLite3    DriverType = "sqlite3"
)

// DB 数据库抽象接口
type DB interface {
	// QueryRow 执行查询并返回单行
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
	// Query 执行查询并返回多行
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)
	// Exec 执行非查询SQL
	Exec(ctx context.Context, sql string, args ...interface{}) (Result, error)
	// Begin 开始事务
	Begin(ctx context.Context) (Tx, error)
	// Ping 检查连接
	Ping(ctx context.Context) error
	// Close 关闭连接
	Close() error
	// DriverType 返回驱动类型
	DriverType() DriverType
}

// Tx 事务抽象接口
type Tx interface {
	// QueryRow 在事务中执行查询并返回单行
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
	// Query 在事务中执行查询并返回多行
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)
	// Exec 在事务中执行非查询SQL
	Exec(ctx context.Context, sql string, args ...interface{}) (Result, error)
	// Commit 提交事务
	Commit(ctx context.Context) error
	// Rollback 回滚事务
	Rollback(ctx context.Context) error
}

// Row 单行结果接口
type Row interface {
	Scan(dest ...interface{}) error
}

// Rows 多行结果接口
type Rows interface {
	Scan(dest ...interface{}) error
	Next() bool
	Close() error
	Err() error
}

// Result 执行结果接口
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Dialect SQL方言接口
type Dialect interface {
	// Placeholder 返回占位符格式
	Placeholder() string
	// PlaceholderFormat 返回占位符格式（用于squirrel）
	PlaceholderFormat() string
	// LastInsertID 获取最后插入的ID（SQLite3需要）
	LastInsertID(ctx context.Context, db DB, table string) (int64, error)
	// SupportsReturning 是否支持RETURNING子句
	SupportsReturning() bool
	// JSONType 返回JSON类型名称
	JSONType() string
	// AutoIncrement 返回自增语法
	AutoIncrement() string
}

// NewDB 根据驱动类型创建数据库连接
func NewDB(driverType DriverType, dsn string) (DB, error) {
	switch driverType {
	case DriverPostgreSQL:
		return NewPostgresDB(dsn)
	case DriverSQLite3:
		return NewSQLite3DB(dsn)
	default:
		return nil, fmt.Errorf("unsupported driver type: %s", driverType)
	}
}

// NewDialect 根据驱动类型创建SQL方言
func NewDialect(driverType DriverType) Dialect {
	switch driverType {
	case DriverPostgreSQL:
		return &PostgresDialect{}
	case DriverSQLite3:
		return &SQLite3Dialect{}
	default:
		// 默认使用PostgreSQL方言
		return &PostgresDialect{}
	}
}

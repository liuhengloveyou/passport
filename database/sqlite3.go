package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// SQLite3DB SQLite3数据库实现
type SQLite3DB struct {
	db *sql.DB
}

// NewSQLite3DB 创建SQLite3数据库连接
func NewSQLite3DB(dsn string) (DB, error) {
	// 如果不是内存数据库，确保目录存在
	if dsn != ":memory:" && dsn != "file::memory:?cache=shared" {
		// 提取文件路径（移除可能的查询参数，如 ?cache=shared）
		filePath := dsn
		for i := 0; i < len(filePath); i++ {
			if filePath[i] == '?' {
				filePath = filePath[:i]
				break
			}
		}

		// 获取目录路径
		dir := filepath.Dir(filePath)
		if dir != "." && dir != "" {
			// 创建目录（如果不存在）
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory for sqlite3: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite3: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite3建议单连接
	db.SetMaxIdleConns(1)

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite3: %w", err)
	}

	return &SQLite3DB{db: db}, nil
}

func (db *SQLite3DB) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return db.db.QueryRowContext(ctx, sql, args...)
}

func (db *SQLite3DB) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	rows, err := db.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &SQLite3Rows{rows: rows}, nil
}

func (db *SQLite3DB) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	result, err := db.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &SQLite3Result{result: result}, nil
}

func (db *SQLite3DB) Begin(ctx context.Context) (Tx, error) {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &SQLite3Tx{tx: tx}, nil
}

func (db *SQLite3DB) Ping(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

func (db *SQLite3DB) Close() error {
	return db.db.Close()
}

func (db *SQLite3DB) DriverType() DriverType {
	return DriverSQLite3
}

// SQLite3Tx SQLite3事务实现
type SQLite3Tx struct {
	tx *sql.Tx
}

func (tx *SQLite3Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return tx.tx.QueryRowContext(ctx, sql, args...)
}

func (tx *SQLite3Tx) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	rows, err := tx.tx.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &SQLite3Rows{rows: rows}, nil
}

func (tx *SQLite3Tx) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	result, err := tx.tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &SQLite3Result{result: result}, nil
}

func (tx *SQLite3Tx) Commit(ctx context.Context) error {
	return tx.tx.Commit()
}

func (tx *SQLite3Tx) Rollback(ctx context.Context) error {
	return tx.tx.Rollback()
}

// SQLite3Rows SQLite3多行结果实现
type SQLite3Rows struct {
	rows *sql.Rows
}

func (r *SQLite3Rows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *SQLite3Rows) Next() bool {
	return r.rows.Next()
}

func (r *SQLite3Rows) Close() error {
	return r.rows.Close()
}

func (r *SQLite3Rows) Err() error {
	return r.rows.Err()
}

// SQLite3Result SQLite3执行结果实现
type SQLite3Result struct {
	result sql.Result
}

func (r *SQLite3Result) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *SQLite3Result) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

// SQLite3Dialect SQLite3 SQL方言
type SQLite3Dialect struct{}

func (d *SQLite3Dialect) Placeholder() string {
	return "?"
}

func (d *SQLite3Dialect) PlaceholderFormat() string {
	return "Question" // 用于squirrel
}

func (d *SQLite3Dialect) LastInsertID(ctx context.Context, db DB, table string) (int64, error) {
	// SQLite3使用last_insert_rowid()
	var id int64
	row := db.QueryRow(ctx, "SELECT last_insert_rowid()")
	if err := row.Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}
	return id, nil
}

func (d *SQLite3Dialect) SupportsReturning() bool {
	// SQLite3 3.35.0+ 支持RETURNING，但为了兼容性，我们使用last_insert_rowid()
	return false // 保守起见，不使用RETURNING
}

func (d *SQLite3Dialect) JSONType() string {
	return "TEXT" // SQLite3使用TEXT存储JSON
}

func (d *SQLite3Dialect) AutoIncrement() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

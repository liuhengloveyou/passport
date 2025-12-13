package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB PostgreSQL数据库实现
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB 创建PostgreSQL数据库连接
func NewPostgresDB(dsn string) (DB, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresDB{pool: pool}, nil
}

func (db *PostgresDB) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

func (db *PostgresDB) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresRows{rows: rows}, nil
}

func (db *PostgresDB) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	result, err := db.pool.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresResult{rowsAffected: result.RowsAffected()}, nil
}

func (db *PostgresDB) Begin(ctx context.Context) (Tx, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &PostgresTx{tx: tx}, nil
}

func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *PostgresDB) Close() error {
	db.pool.Close()
	return nil
}

func (db *PostgresDB) DriverType() DriverType {
	return DriverPostgreSQL
}

// PostgresTx PostgreSQL事务实现
type PostgresTx struct {
	tx pgx.Tx
}

func (tx *PostgresTx) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return tx.tx.QueryRow(ctx, sql, args...)
}

func (tx *PostgresTx) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	rows, err := tx.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresRows{rows: rows}, nil
}

func (tx *PostgresTx) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	result, err := tx.tx.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &PostgresResult{rowsAffected: result.RowsAffected()}, nil
}

func (tx *PostgresTx) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

func (tx *PostgresTx) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}

// PostgresRows PostgreSQL多行结果实现
type PostgresRows struct {
	rows pgx.Rows
}

func (r *PostgresRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *PostgresRows) Next() bool {
	return r.rows.Next()
}

func (r *PostgresRows) Close() error {
	r.rows.Close()
	return nil
}

func (r *PostgresRows) Err() error {
	return r.rows.Err()
}

// PostgresResult PostgreSQL执行结果实现
type PostgresResult struct {
	rowsAffected int64
}

func (r *PostgresResult) LastInsertId() (int64, error) {
	// PostgreSQL不支持LastInsertId，需要通过RETURNING获取
	return 0, fmt.Errorf("PostgreSQL does not support LastInsertId, use RETURNING clause instead")
}

func (r *PostgresResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// PostgresDialect PostgreSQL SQL方言
type PostgresDialect struct{}

func (d *PostgresDialect) Placeholder() string {
	return "$"
}

func (d *PostgresDialect) PlaceholderFormat() string {
	return "Dollar" // 用于squirrel
}

func (d *PostgresDialect) LastInsertID(ctx context.Context, db DB, table string) (int64, error) {
	// PostgreSQL不支持，应该使用RETURNING
	return 0, fmt.Errorf("PostgreSQL does not support LastInsertID, use RETURNING clause")
}

func (d *PostgresDialect) SupportsReturning() bool {
	return true
}

func (d *PostgresDialect) JSONType() string {
	return "JSONB"
}

func (d *PostgresDialect) AutoIncrement() string {
	return "BIGSERIAL"
}

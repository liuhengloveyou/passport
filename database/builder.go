package database

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

// InsertWithID 插入数据并返回ID（兼容PostgreSQL和SQLite3）
func InsertWithID(ctx context.Context, db DB, tx Tx, table string, data map[string]interface{}) (int64, error) {
	dialect := NewDialect(db.DriverType())

	// 构建插入SQL
	var builder sq.InsertBuilder
	if dialect.PlaceholderFormat() == "Dollar" {
		builder = sq.Insert(table).SetMap(data).PlaceholderFormat(sq.Dollar)
	} else {
		builder = sq.Insert(table).SetMap(data).PlaceholderFormat(sq.Question)
	}

	// PostgreSQL使用RETURNING，SQLite3使用last_insert_rowid()
	if dialect.SupportsReturning() {
		// PostgreSQL: 使用RETURNING子句
		sql, vals, err := builder.Suffix("RETURNING uid").ToSql()
		if err != nil {
			return 0, fmt.Errorf("failed to build insert sql: %w", err)
		}

		var uid int64
		if tx != nil {
			err = tx.QueryRow(ctx, sql, vals...).Scan(&uid)
		} else {
			err = db.QueryRow(ctx, sql, vals...).Scan(&uid)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to insert and get id: %w", err)
		}
		return uid, nil
	} else {
		// SQLite3: 先插入，再获取last_insert_rowid()
		sql, vals, err := builder.ToSql()
		if err != nil {
			return 0, fmt.Errorf("failed to build insert sql: %w", err)
		}

		if tx != nil {
			_, err = tx.Exec(ctx, sql, vals...)
		} else {
			_, err = db.Exec(ctx, sql, vals...)
		}
		if err != nil {
			return 0, fmt.Errorf("failed to insert: %w", err)
		}

		// 获取最后插入的ID
		uid, err := dialect.LastInsertID(ctx, db, table)
		if err != nil {
			return 0, fmt.Errorf("failed to get last insert id: %w", err)
		}
		return uid, nil
	}
}

// GetPlaceholderFormat 获取squirrel占位符格式
func GetPlaceholderFormat(driverType DriverType) sq.PlaceholderFormat {
	dialect := NewDialect(driverType)
	switch dialect.PlaceholderFormat() {
	case "Dollar":
		return sq.Dollar
	case "Question":
		return sq.Question
	default:
		return sq.Dollar
	}
}

// BuildJSONColumn 构建JSON列定义（兼容PostgreSQL和SQLite3）
func BuildJSONColumn(driverType DriverType) string {
	dialect := NewDialect(driverType)
	return dialect.JSONType()
}

// BuildAutoIncrement 构建自增列定义
func BuildAutoIncrement(driverType DriverType) string {
	dialect := NewDialect(driverType)
	return dialect.AutoIncrement()
}

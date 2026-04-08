package dao

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/database"
)

// TenantClosureInsert 插入租户闭包表记录
func TenantClosureInsert(tx database.Tx, ancestorId, descendantId uint64) error {
	ctx := context.Background()
	driverType := common.DB.DriverType()

	// 根据数据库类型构建占位符
	var ph1, ph2, ph3 string
	if driverType == "postgres" {
		ph1, ph2, ph3 = "$1", "$2", "$3"
	} else {
		ph1, ph2, ph3 = "?", "?", "?"
	}

	insertPrefix := "INSERT INTO tenant_closure"
	insertSuffix := ""
	switch driverType {
	case "postgres":
		insertSuffix = " ON CONFLICT (ancestor_id, descendant_id) DO NOTHING"
	case "sqlite":
		insertPrefix = "INSERT OR IGNORE INTO tenant_closure"
	default:
		// mysql/mariadb 等数据库使用 INSERT IGNORE 保证幂等
		insertPrefix = "INSERT IGNORE INTO tenant_closure"
	}

	// 插入自己到自己的记录（距离为0）
	_, err := tx.Exec(ctx, insertPrefix+" (ancestor_id, descendant_id, depth) VALUES ("+ph1+", "+ph2+", "+ph3+")"+insertSuffix, descendantId, descendantId, 0)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureInsert self record ERR: %v\n", err)
		return err
	}

	// 插入父到子的记录
	_, err = tx.Exec(ctx, insertPrefix+" (ancestor_id, descendant_id, depth) VALUES ("+ph1+", "+ph2+", "+ph3+")"+insertSuffix, ancestorId, descendantId, 1)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureInsert parent to child record ERR: %v\n", err)
		return err
	}

	// 如果有父租户，插入从所有祖先到当前租户的记录
	insertSQL := `
		` + insertPrefix + ` (ancestor_id, descendant_id, depth)
		SELECT ancestor_id, ` + ph1 + `, depth + 1
		FROM tenant_closure
		WHERE descendant_id = ` + ph2 + insertSuffix
	_, err = tx.Exec(ctx, insertSQL, descendantId, ancestorId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureInsert ancestor records ERR: %v\n", err)
		return err
	}

	return nil
}

// TenantClosureIsDescendant 检查descendantID是否为ancestorID的后代（包括自己）
func TenantClosureIsDescendant(ancestorID, descendantID uint64) (int, error) {
	ctx := context.Background()
	driverType := common.DB.DriverType()

	var ph1, ph2 string
	if driverType == "postgres" {
		ph1, ph2 = "$1", "$2"
	} else {
		ph1, ph2 = "?", "?"
	}

	var depth int
	err := common.DB.QueryRow(ctx,
		"SELECT depth FROM tenant_closure WHERE ancestor_id = "+ph1+" AND descendant_id = "+ph2,
		ancestorID, descendantID).Scan(&depth)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return -1, nil
		}
		common.Logger.Sugar().Errorf("TenantClosureIsDescendant ERR: %v %v %v\n", ancestorID, descendantID, err)
		return -1, err
	}
	return depth, nil
}

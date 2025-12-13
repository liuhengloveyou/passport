package dao

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/liuhengloveyou/passport/common"
)

// TenantClosureInsert 插入租户闭包表记录
func TenantClosureInsert(tx pgx.Tx, ancestorId, descendantId uint64) error {
	// 插入自己到自己的记录（距离为0）
	_, err := tx.Exec(context.Background(), "INSERT INTO tenant_closure (ancestor_id, descendant_id, depth) VALUES ($1, $2, $3)", descendantId, descendantId, 0)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureInsert self record ERR: %v\n", err)
		return err
	}

	// 插入父到子的记录
	_, err = tx.Exec(context.Background(), "INSERT INTO tenant_closure (ancestor_id, descendant_id, depth) VALUES ($1, $2, $3)", ancestorId, descendantId, 1)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureInsert parent to child record ERR: %v\n", err)
		return err
	}

	// 如果有父租户，插入从所有祖先到当前租户的记录
	_, err = tx.Exec(context.Background(), `
			INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
			SELECT ancestor_id, $1, depth + 1
			FROM tenant_closure
			WHERE descendant_id = $2
		`, descendantId, ancestorId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureInsert ancestor records ERR: %v\n", err)
		return err
	}

	return nil
}

// TenantClosureIsDescendant 检查descendantID是否为ancestorID的后代（包括自己）
func TenantClosureIsDescendant(ancestorID, descendantID uint64) (int, error) {
	var depth int
	err := common.DBPool.QueryRow(context.Background(),
		"SELECT depth FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2",
		ancestorID, descendantID).Scan(&depth)

	if err != nil {
		if err == pgx.ErrNoRows {
			return -1, nil
		}
		common.Logger.Sugar().Errorf("TenantClosureIsDescendant ERR: %v\n", ancestorID, descendantID, err)

		return -1, err
	}
	return depth, nil
}

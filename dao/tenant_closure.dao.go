package dao

import (
	"context"
	"fmt"
	"strings"

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

// TenantClosureDeleteByDescendant 删除指定租户作为后代的所有闭包记录
func TenantClosureDeleteByDescendant(tx pgx.Tx, tenantID uint64) error {
	_, err := tx.Exec(context.Background(), "DELETE FROM tenant_closure WHERE descendant_id = $1", tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureDeleteByDescendant ERR: %v\n", err)
		return err
	}
	return nil
}

// TenantClosureIsDescendant 检查descendantID是否为ancestorID的后代（包括自己）
func TenantClosureIsDescendant(ancestorID, descendantID uint64) (bool, error) {
	var count int
	err := common.DBPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2",
		ancestorID, descendantID).Scan(&count)

	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureIsDescendant ERR: %v\n", err)
		return false, err
	}
	return count > 0, nil
}

// TenantClosureGetDescendants 获取指定租户的所有子孙节点ID
func TenantClosureGetDescendants(tx pgx.Tx, tenantID uint64) ([]uint64, error) {
	rows, err := tx.Query(context.Background(), "SELECT descendant_id FROM tenant_closure WHERE ancestor_id = $1 AND depth > 0", tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantClosureGetDescendants ERR: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	var descendants []uint64
	for rows.Next() {
		var descendantID uint64
		if err := rows.Scan(&descendantID); err != nil {
			common.Logger.Sugar().Errorf("TenantClosureGetDescendants scan ERR: %v\n", err)
			return nil, err
		}
		descendants = append(descendants, descendantID)
	}

	if err := rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("TenantClosureGetDescendants rows ERR: %v\n", err)
		return nil, err
	}

	return descendants, nil
}

// TenantClosureUpdateSubtree 更新整个子树的层级关系
// 注意：调用此函数时，应确保事务隔离级别至少为 REPEATABLE READ，建议使用 SERIALIZABLE
// 示例：
//
//	opts := pgx.TxOptions{
//	    IsoLevel: pgx.Serializable,
//	}
//
// tx, err := common.DBPool.BeginTx(context.Background(), opts)
func TenantClosureUpdateSubtree(tx pgx.Tx, tenantID, newParentID uint64) error {
	// 参数验证
	if tenantID == 0 {
		return fmt.Errorf("invalid tenant ID: cannot be zero")
	}

	// 检查是否尝试将节点设为自己的子节点（循环引用）
	if newParentID > 0 {
		var isDescendant bool
		err := tx.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2)",
			tenantID, newParentID).Scan(&isDescendant)
		if err != nil {
			common.Logger.Sugar().Errorf("TenantClosureUpdateSubtree check circular reference ERR: %v\n", err)
			return common.ErrTenantSetParent
		}
		if isDescendant {
			common.Logger.Sugar().Errorf("circular reference: cannot set tenant %d as child of its own descendant %d", tenantID, newParentID)
			return common.ErrTenantCircularRef
		}
	}

	// 1. 获取所有子孙节点（包括自己）
	allDescendants, err := TenantClosureGetDescendants(tx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get descendants: %w", err)
	}
	// 将自己也加入到列表中
	allDescendants = append([]uint64{tenantID}, allDescendants...)

	// 2. 批量获取子树内部的深度关系，避免删除后丢失
	subtreeDepths := make(map[uint64]int)

	// 为批量查询构建参数
	queryParams := make([]interface{}, 0, len(allDescendants)*2)
	queryParams = append(queryParams, tenantID)

	// 构建查询占位符
	placeholders := make([]string, 0, len(allDescendants))
	for i, descendantID := range allDescendants {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+2))
		queryParams = append(queryParams, descendantID)
	}

	// 批量查询深度
	rows, err := tx.Query(context.Background(),
		fmt.Sprintf("SELECT descendant_id, depth FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id IN (%s)",
			strings.Join(placeholders, ",")),
		queryParams...)
	if err != nil {
		return fmt.Errorf("failed to query subtree depths: %w", err)
	}
	defer rows.Close()

	// 处理查询结果
	for rows.Next() {
		var descendantID uint64
		var depth int
		if err = rows.Scan(&descendantID, &depth); err != nil {
			return fmt.Errorf("failed to scan depth row: %w", err)
		}
		subtreeDepths[descendantID] = depth
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating depth rows: %w", err)
	}

	// 确保根节点有深度值
	if _, exists := subtreeDepths[tenantID]; !exists {
		subtreeDepths[tenantID] = 0
		common.Logger.Sugar().Infof("Root tenant %d depth not found, setting to 0", tenantID)
	}

	// 3. 批量删除所有子孙节点与外部祖先的旧层级关系（保留子树内部关系）
	// 使用批量操作替代循环
	deleteSQL := `
		DELETE FROM tenant_closure 
		WHERE descendant_id IN (
			SELECT unnest($1::bigint[])
		) 
		AND ancestor_id NOT IN (
			SELECT descendant_id FROM tenant_closure WHERE ancestor_id = $2
			UNION SELECT $2
		)
	`
	_, err = tx.Exec(context.Background(), deleteSQL, allDescendants, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete old relations: %w", err)
	}

	// 4. 为所有子孙节点重新建立与新父级的层级关系
	if newParentID > 0 {
		// 获取新父级的所有祖先
		ancestorRows, err := tx.Query(context.Background(),
			"SELECT ancestor_id, depth FROM tenant_closure WHERE descendant_id = $1",
			newParentID)
		if err != nil {
			return fmt.Errorf("failed to query new parent ancestors: %w", err)
		}
		defer ancestorRows.Close()

		// 收集所有祖先信息
		type AncestorInfo struct {
			ID    uint64
			Depth int
		}
		ancestors := []AncestorInfo{}

		for ancestorRows.Next() {
			var ancestor AncestorInfo
			if err = ancestorRows.Scan(&ancestor.ID, &ancestor.Depth); err != nil {
				return fmt.Errorf("failed to scan ancestor row: %w", err)
			}
			ancestors = append(ancestors, ancestor)
		}

		if err = ancestorRows.Err(); err != nil {
			return fmt.Errorf("error iterating ancestor rows: %w", err)
		}

		// 批量插入新关系
		if len(ancestors) > 0 && len(allDescendants) > 0 {
			// 准备批量插入
			batch := &pgx.Batch{}

			for _, descendantID := range allDescendants {
				subtreeDepth, exists := subtreeDepths[descendantID]
				if !exists {
					common.Logger.Sugar().Warnf("Subtree depth not found for descendant %d, skipping", descendantID)
					continue
				}

				for _, ancestor := range ancestors {
					// 计算新的深度：祖先到新父级的深度 + 子树中节点到子树根的深度 + 1
					newDepth := ancestor.Depth + subtreeDepth + 1

					// 添加到批处理
					batch.Queue("INSERT INTO tenant_closure (ancestor_id, descendant_id, depth) VALUES ($1, $2, $3)",
						ancestor.ID, descendantID, newDepth)
				}
			}

			// 执行批处理
			br := tx.SendBatch(context.Background(), batch)
			defer br.Close()

			// 检查批处理结果
			for i := 0; i < batch.Len(); i++ {
				_, err := br.Exec()
				if err != nil {
					return fmt.Errorf("failed to execute batch insert at index %d: %w", i, err)
				}
			}
		}
	}

	common.Logger.Sugar().With(
		"tenantID", tenantID,
		"newParentID", newParentID,
		"descendantCount", len(allDescendants),
	).Infof("Successfully updated tenant subtree")

	return nil
}

package dao

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/common"
)

// SQL常量定义，提高代码可维护性
const (
	sqlCheckCircularReference = `
		SELECT a.ancestor_id, a.descendant_id
		FROM tenant_closure a
		JOIN tenant_closure b ON a.ancestor_id = b.descendant_id AND a.descendant_id = b.ancestor_id
		WHERE a.ancestor_id != a.descendant_id
	`
	sqlCheckDirectRelation  = `SELECT EXISTS(SELECT 1 FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2 AND depth = 1)`
	sqlInsertDirectRelation = `INSERT INTO tenant_closure (ancestor_id, descendant_id, depth) VALUES ($1, $2, 1) ON CONFLICT (ancestor_id, descendant_id) DO UPDATE SET depth = 1 WHERE tenant_closure.depth > 1`
)

// CircularReferenceError 循环引用错误类型
type CircularReferenceError struct {
	TenantID, DescendantID uint64
}

func (e CircularReferenceError) Error() string {
	return fmt.Sprintf("circular reference: cannot set tenant %d as child of its descendant %d", e.TenantID, e.DescendantID)
}

// DetectCircularReference 检测租户树中的循环引用
// 返回包含循环引用的节点ID列表，如果没有循环引用则返回空列表
// 注意：由于tenants表中已经没有parent_id字段，现在使用tenant_closure表中的depth=1关系来检测循环引用
func DetectCircularReference(tx pgx.Tx) ([]uint64, error) {
	// 使用更简单的方法检测循环引用：如果A是B的祖先，同时B也是A的祖先，则存在循环引用
	rows, err := tx.Query(context.Background(), sqlCheckCircularReference)
	if err != nil {
		return nil, fmt.Errorf("failed to detect circular references: %w", err)
	}
	defer rows.Close()

	var circularNodes []uint64
	var seen = make(map[uint64]bool)

	for rows.Next() {
		var ancestorID, descendantID uint64
		if err := rows.Scan(&ancestorID, &descendantID); err != nil {
			return nil, fmt.Errorf("failed to scan circular reference: %w", err)
		}

		if !seen[ancestorID] {
			circularNodes = append(circularNodes, ancestorID)
			seen[ancestorID] = true
		}

		if !seen[descendantID] {
			circularNodes = append(circularNodes, descendantID)
			seen[descendantID] = true
		}

		common.Logger.Sugar().Warnf("Circular reference detected: %d is both ancestor and descendant of %d",
			ancestorID, descendantID)
	}

	if len(circularNodes) > 0 {
		common.Logger.Sugar().Warnf("Detected circular references involving nodes: %v", circularNodes)
	}

	return circularNodes, nil
}

// TenantClosureUpdateSubtree 更新整个子树的层级关系 - 优化版closure table实现
// 算法步骤：
// 1. 删除移动子树与其当前所有祖先的关系（不包括自身和新父节点及其祖先）
// 2. 为移动子树中每个节点与新父节点的所有祖先建立新关系
// 特殊情况：
// - 当节点已经是目标父节点的子节点时，保留这种关系
func TenantClosureUpdateSubtree(tx pgx.Tx, tenantID, newParentID uint64, debug bool) error {
	// 参数验证
	if tenantID == 0 {
		return fmt.Errorf("invalid tenant ID: cannot be zero")
	}

	// 检查循环引用
	if newParentID > 0 {
		var count int
		err := tx.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2",
			tenantID, newParentID).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check circular reference: %w", err)
		}
		if count > 0 {
			return &CircularReferenceError{TenantID: tenantID, DescendantID: newParentID}
		}
	}

	// 记录操作前的状态（如果开启调试模式）
	if debug && newParentID > 0 {
		var beforeCount, existingRelation int
		tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM tenant_closure WHERE descendant_id = $1", tenantID).Scan(&beforeCount)
		tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2", newParentID, tenantID).Scan(&existingRelation)
		common.Logger.Sugar().Infof("Before move: tenant %d has %d ancestor relations, relation with %d exists: %d",
			tenantID, beforeCount, newParentID, existingRelation)
	}

	// 步骤1: 使用CTE优化删除操作
	deleteSQL := `
		WITH subtree AS (
		  SELECT descendant_id FROM tenant_closure WHERE ancestor_id = $1
		),
		supertree AS (
		  SELECT ancestor_id FROM tenant_closure WHERE descendant_id = $2
		)
		DELETE FROM tenant_closure 
		WHERE descendant_id IN (SELECT * FROM subtree)
		  AND ancestor_id NOT IN (SELECT * FROM subtree)
		  AND ancestor_id != $2
		  AND ancestor_id NOT IN (SELECT * FROM supertree)
		RETURNING ancestor_id, descendant_id
	`

	// 删除旧的直接父子关系（深度为1），但保留新父节点的关系
	deleteOldParentSQL := `
		DELETE FROM tenant_closure 
		WHERE descendant_id = $1 
		AND depth = 1 
		AND ancestor_id != $2
	`

	deleteResult, err := tx.Exec(context.Background(), deleteSQL, tenantID, newParentID)
	if err != nil {
		return fmt.Errorf("failed to delete old relations: %w", err)
	}

	deleteParentResult, err := tx.Exec(context.Background(), deleteOldParentSQL, tenantID, newParentID)
	if err != nil {
		return fmt.Errorf("failed to delete old parent relations: %w", err)
	}

	if debug {
		common.Logger.Sugar().Infof("Deleted %d old relations and %d direct parent relations for tenant %d",
			deleteResult.RowsAffected(), deleteParentResult.RowsAffected(), tenantID)
	}

	// 步骤2: 为移动子树中每个节点与新父节点的所有祖先建立新关系
	if newParentID > 0 {
		// 优化：合并多个计数查询为一个（如果开启调试模式）
		if debug {
			var supertreeCount, subtreeCount, directRelationCount int
			err := tx.QueryRow(context.Background(), `
				SELECT 
					(SELECT COUNT(*) FROM tenant_closure WHERE descendant_id = $1),
					(SELECT COUNT(*) FROM tenant_closure WHERE ancestor_id = $2),
					(SELECT COUNT(*) FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2)
			`, newParentID, tenantID).Scan(&supertreeCount, &subtreeCount, &directRelationCount)

			if err == nil {
				common.Logger.Sugar().Infof("Before INSERT: supertree (parent %d) has %d ancestors, subtree (tenant %d) has %d descendants, direct relation exists: %d",
					newParentID, supertreeCount, tenantID, subtreeCount, directRelationCount)
			}
		}

		insertSQL := `
			INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
			SELECT supertree.ancestor_id, subtree.descendant_id, supertree.depth + subtree.depth + 1
			FROM tenant_closure AS supertree
			CROSS JOIN tenant_closure AS subtree
			WHERE supertree.descendant_id = $2
			AND subtree.ancestor_id = $1
			ON CONFLICT (ancestor_id, descendant_id) DO UPDATE 
			SET depth = EXCLUDED.depth
			WHERE tenant_closure.depth > EXCLUDED.depth
		`
		insertResult, err := tx.Exec(context.Background(), insertSQL, tenantID, newParentID)
		if err != nil {
			return fmt.Errorf("failed to insert new relations: %w", err)
		}

		if debug {
			common.Logger.Sugar().Infof("Inserted %d new relations for subtree %d with parent %d",
				insertResult.RowsAffected(), tenantID, newParentID)
		}

		// 验证直接父子关系是否建立，如果没有则手动添加
		var directRelationExists bool
		err = tx.QueryRow(context.Background(), sqlCheckDirectRelation, newParentID, tenantID).Scan(&directRelationExists)
		if err != nil {
			return fmt.Errorf("failed to verify direct parent relation: %w", err)
		}

		if !directRelationExists {
			if debug {
				common.Logger.Sugar().Warnf("Direct parent relation not established, manually adding it")
			}
			_, err = tx.Exec(context.Background(), sqlInsertDirectRelation, newParentID, tenantID)
			if err != nil {
				return fmt.Errorf("failed to manually add direct parent relation: %w", err)
			}
			if debug {
				common.Logger.Sugar().Infof("Manually added direct parent relation between %d and %d", newParentID, tenantID)
			}
		}
	}

	// 最终验证（如果开启调试模式）
	if debug {
		var finalAncestorCount int
		tx.QueryRow(context.Background(), "SELECT COUNT(*) FROM tenant_closure WHERE descendant_id = $1", tenantID).Scan(&finalAncestorCount)
		common.Logger.Sugar().Infof("Final state: tenant %d has %d ancestor relations", tenantID, finalAncestorCount)

		if newParentID > 0 {
			var finalDirectRelation bool
			tx.QueryRow(context.Background(), sqlCheckDirectRelation, newParentID, tenantID).Scan(&finalDirectRelation)
			common.Logger.Sugar().Infof("Final verification: direct relation between %d and %d exists: %v",
				newParentID, tenantID, finalDirectRelation)
		}

		common.Logger.Sugar().Infof("Successfully moved subtree %d to parent %d using optimized closure table algorithm",
			tenantID, newParentID)
	}

	return nil
}

// TenantClosureUpdateSubtreeV2 原始函数的兼容包装
// 保留此函数以兼容现有代码，内部调用优化后的TenantClosureUpdateSubtree函数
func TenantClosureUpdateSubtreeV2(tx pgx.Tx, tenantID, newParentID uint64) error {
	return TenantClosureUpdateSubtree(tx, tenantID, newParentID, true)
}

// 辅助函数：执行SQL并处理错误
func execSQL(tx pgx.Tx, sql string, args ...interface{}) error {
	_, err := tx.Exec(context.Background(), sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}
	return nil
}

// CheckAndRepairTenantClosureTable 检查并修复整个租户闭包表的数据一致性
// 这是一个系统维护函数，可以用于修复数据库中可能存在的不一致问题
func CheckAndRepairTenantClosureTable(tx pgx.Tx) error {
	common.Logger.Sugar().Info("Starting full tenant closure table consistency check")

	// 1. 确保所有节点都有自引用关系（depth=0）
	err := execSQL(tx, `
		INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
		SELECT id, id, 0 FROM tenants
		ON CONFLICT (ancestor_id, descendant_id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure self-references: %w", err)
	}

	// 2. 确保所有直接父子关系存在（depth=1）
	// 注意：由于tenants表中已经没有parent_id字段，这一步现在依赖于tenant_closure表中已有的关系
	// 我们只需确保depth=1的关系正确，不再从tenants表中读取parent_id
	common.Logger.Sugar().Info("Skipping direct parent-child relation check as parent_id field no longer exists in tenants table")

	// 3. 确保传递关系完整（如果A是B的祖先，B是C的祖先，则A也是C的祖先）
	// 使用批处理优化多个SQL操作
	batch := &pgx.Batch{}

	// 添加传递关系SQL到批处理
	batch.Queue(`
		INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
		SELECT a.ancestor_id, b.descendant_id, a.depth + b.depth
		FROM tenant_closure a
		JOIN tenant_closure b ON a.descendant_id = b.ancestor_id
		WHERE a.ancestor_id != b.descendant_id
		ON CONFLICT (ancestor_id, descendant_id) DO NOTHING
	`)

	// 添加更新depth值SQL到批处理
	batch.Queue(`
		UPDATE tenant_closure c
		SET depth = (
			SELECT MIN(a.depth + b.depth)
			FROM tenant_closure a
			JOIN tenant_closure b ON a.descendant_id = b.ancestor_id
			WHERE a.ancestor_id = c.ancestor_id AND b.descendant_id = c.descendant_id
		)
		WHERE EXISTS (
			SELECT 1
			FROM tenant_closure a
			JOIN tenant_closure b ON a.descendant_id = b.ancestor_id
			WHERE a.ancestor_id = c.ancestor_id AND b.descendant_id = c.descendant_id
			AND a.depth + b.depth < c.depth
		)
	`)

	// 执行批处理
	results := tx.SendBatch(context.Background(), batch)
	defer results.Close()

	// 检查第一个SQL的结果
	_, err = results.Exec()
	if err != nil {
		return fmt.Errorf("failed to ensure transitive relations: %w", err)
	}

	// 检查第二个SQL的结果
	_, err = results.Exec()
	if err != nil {
		return fmt.Errorf("failed to update depth values: %w", err)
	}

	common.Logger.Sugar().Info("Tenant closure table consistency check and repair completed successfully")
	return nil
}

// EnsureTenantClosureConsistency 确保特定租户与其父节点的闭包关系一致性
// 检查并修复可能的数据问题，确保所有必要的关系都存在
func EnsureTenantClosureConsistency(tx pgx.Tx, tenantID, parentID uint64) error {
	if tenantID == 0 || parentID == 0 {
		return nil // 根节点或无效ID，不需要检查
	}

	// 检查直接父子关系
	var directRelationExists bool
	err := tx.QueryRow(context.Background(), sqlCheckDirectRelation, parentID, tenantID).Scan(&directRelationExists)
	if err != nil {
		return fmt.Errorf("failed to check direct relation: %w", err)
	}

	// 如果直接关系不存在，添加它
	if !directRelationExists {
		common.Logger.Sugar().Warnf("Consistency check: Missing direct relation between %d and %d, adding it", parentID, tenantID)
		_, err = tx.Exec(context.Background(), sqlInsertDirectRelation, parentID, tenantID)
		if err != nil {
			return fmt.Errorf("failed to add direct relation: %w", err)
		}
	}

	// 检查父节点的所有祖先与子节点的关系
	_, err = tx.Exec(context.Background(), `
		INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
		SELECT a.ancestor_id, $2, a.depth + 1
		FROM tenant_closure a
		WHERE a.descendant_id = $1
		ON CONFLICT (ancestor_id, descendant_id) DO NOTHING
	`, parentID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to ensure ancestor relations: %w", err)
	}

	common.Logger.Sugar().Infof("Consistency check completed for tenant %d with parent %d", tenantID, parentID)
	return nil
}

// TenantClosureUpdateSubtreeV2Safe 安全版本，增加了额外的检查和日志
func TenantClosureUpdateSubtreeV2Safe(tx pgx.Tx, tenantID, newParentID uint64) error {
	// 执行移动操作（使用优化版本，开启调试模式）
	err := TenantClosureUpdateSubtree(tx, tenantID, newParentID, true)
	if err != nil {
		return err
	}

	// 执行数据一致性检查和修复
	if newParentID > 0 {
		err = EnsureTenantClosureConsistency(tx, tenantID, newParentID)
		if err != nil {
			common.Logger.Sugar().Warnf("Consistency check failed: %v, but continuing with operation", err)
		}

		// 验证新的父子关系
		var directRelationExists bool
		err = tx.QueryRow(context.Background(), sqlCheckDirectRelation, newParentID, tenantID).Scan(&directRelationExists)
		if err != nil {
			return fmt.Errorf("failed to verify new parent relation: %w", err)
		}
		if !directRelationExists {
			return fmt.Errorf("failed to establish direct parent relation between %d and %d", newParentID, tenantID)
		}
		common.Logger.Sugar().Infof("Verified: tenant %d is now direct child of %d", tenantID, newParentID)

		// 检查是否创建了循环引用
		circularNodes, err := DetectCircularReference(tx)
		if err != nil {
			common.Logger.Sugar().Warnf("Failed to check for circular references: %v", err)
		} else if len(circularNodes) > 0 {
			common.Logger.Sugar().Errorf("WARNING: Circular references detected after move operation: %v", circularNodes)
			// 这里不返回错误，因为操作已经完成，但记录警告以便后续处理
		}
	}

	return nil
}

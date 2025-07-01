package dao

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	sq "github.com/Masterminds/squirrel"
)

func DepartmentCreate(model *protos.Department) (lastInsertId int64, err error) {
	// 使用 RETURNING id 子句获取新插入记录的 ID
	err = common.DBPool.QueryRow(context.Background(), "INSERT INTO departments (uid, tenant_id, parent_id, name, create_time) VALUES ($1, $2, $3, $4, $5) RETURNING uid",
		model.UserId, model.TenantID, model.ParentID, model.Name, time.Now()).Scan(&lastInsertId)

	if err != nil {
		common.Logger.Sugar().Errorf("Failed to insert department: %v", err)
		return -1, err
	}

	return
}

func DepartmentDelete(id, tenantID uint64) (rowsAffected int64, err error) {
	// 使用 Exec 执行删除操作
	rst, err := common.DBPool.Exec(context.Background(), "DELETE FROM departments WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to delete department: %v", err)
		return 0, err
	}

	// 获取受影响的行数
	rowsAffected = rst.RowsAffected()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to get rows affected: %v", err)
		return 0, err
	}
	return
}

func DepartmentUpdate(model *protos.Department) (rowsAffected int64, err error) {
	var rst pgconn.CommandTag

	common.Logger.Sugar().Debugf("UPDATE departments SET name=$1 WHERE (id=$2 AND tenant_id=$3)", model.Name, model.Id, model.TenantID)
	rst, err = common.DBPool.Exec(context.Background(), "UPDATE departments SET name=$1 WHERE (id=$2 AND tenant_id=$3)", model.Name, model.Id, model.TenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to update department: %v", err)
		return 0, err
	}

	rowsAffected = rst.RowsAffected()
	return
}

func DepartmentUpdateConfig(model *protos.Department) (rowsAffected int64, err error) {
	var rst pgconn.CommandTag

	common.Logger.Sugar().Debugf("UPDATE departments SET config=$1 WHERE (id=$2)", model.Config, model.Id)
	rst, err = common.DBPool.Exec(context.Background(), "UPDATE departments SET config=$1 WHERE (id=$2)", model.Config, model.Id)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to update department config: %v", err)
		return 0, err
	}

	rowsAffected = rst.RowsAffected()
	return
}

func DepartmentFind(id, tenantId, page, pageSize uint64) (rr []protos.Department, err error) {
	tx := sq.Select("*").From("departments").PlaceholderFormat(sq.Dollar).Where("tenant_id = $1", tenantId).OrderBy("update_time desc")

	if id > 0 {
		tx = tx.Where("id = $2", id)
	}
	if page > 0 && pageSize > 0 {
		tx = tx.Limit(pageSize).Offset((page - 1) * pageSize)
	}

	sql, args, err := tx.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("DepartmentFind: %v %v", sql, args)

	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return nil, err
	}
	defer rows.Close()

	rr = []protos.Department{}
	for rows.Next() {
		var dept protos.Department
		err = rows.Scan(&dept.Id, &dept.UserId, &dept.TenantID, &dept.ParentID, &dept.Name, &dept.Config, &dept.CreateTime, &dept.UpdateTime)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to scan row: %v", err)
			return nil, err
		}
		rr = append(rr, dept)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("Error iterating rows: %v", err)
		return nil, err
	}

	return
}

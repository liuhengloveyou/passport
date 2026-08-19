package dao

import (
	"context"
	"time"

	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/database"
	"github.com/liuhengloveyou/passport/v4/protos"

	sq "github.com/Masterminds/squirrel"
)

func DepartmentCreate(model *protos.Department) (lastInsertId int64, err error) {
	err = common.DB.QueryRow(context.Background(),
		"INSERT INTO departments (uid, tenant_id, org_id, parent_id, name, create_time) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		model.UserId, model.TenantID, model.OrgID, model.ParentID, model.Name, time.Now()).Scan(&lastInsertId)

	if err != nil {
		common.Logger.Sugar().Errorf("Failed to insert department: %v", err)
		return -1, err
	}

	return
}

func DepartmentDelete(id, tenantID uint64) (rowsAffected int64, err error) {
	rst, err := common.DB.Exec(context.Background(), "DELETE FROM departments WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to delete department: %v", err)
		return 0, err
	}

	rowsAffected, _ = rst.RowsAffected()
	return
}

func DepartmentDeleteByOrg(tenantID, orgID uint64) error {
	if tenantID == 0 || orgID == 0 {
		return common.ErrParam
	}
	_, err := common.DB.Exec(context.Background(),
		`DELETE FROM departments WHERE tenant_id = $1 AND org_id = $2`, tenantID, orgID)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentDeleteByOrg ERR: %v", err)
		return err
	}
	return nil
}

func DepartmentUpdate(model *protos.Department) (rowsAffected int64, err error) {
	common.Logger.Sugar().Debugf("UPDATE departments SET name=$1 WHERE (id=$2 AND tenant_id=$3)", model.Name, model.Id, model.TenantID)
	rst, err := common.DB.Exec(context.Background(), "UPDATE departments SET name=$1 WHERE (id=$2 AND tenant_id=$3)", model.Name, model.Id, model.TenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to update department: %v", err)
		return 0, err
	}

	rowsAffected, _ = rst.RowsAffected()
	return
}

func DepartmentUpdateConfig(model *protos.Department) (rowsAffected int64, err error) {
	common.Logger.Sugar().Debugf("UPDATE departments SET config=$1 WHERE (id=$2)", model.Config, model.Id)
	rst, err := common.DB.Exec(context.Background(), "UPDATE departments SET config=$1 WHERE (id=$2)", model.Config, model.Id)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to update department config: %v", err)
		return 0, err
	}

	rowsAffected, _ = rst.RowsAffected()
	return
}

func DepartmentFind(id, tenantId, orgId, page, pageSize uint64) (rr []protos.Department, err error) {
	tx := sq.Select(
		"id",
		"parent_id",
		"uid",
		"tenant_id",
		"org_id",
		"create_time",
		"update_time",
		"name",
		"config",
	).From("departments").PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).
		Where(sq.Eq{"tenant_id": tenantId}).
		OrderBy("update_time desc")

	if id > 0 {
		tx = tx.Where(sq.Eq{"id": id})
	}
	if orgId > 0 {
		tx = tx.Where(sq.Eq{"org_id": orgId})
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

	rows, err := common.DB.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return nil, err
	}
	defer rows.Close()

	rr = []protos.Department{}
	for rows.Next() {
		var dept protos.Department
		err = rows.Scan(
			&dept.Id,
			&dept.ParentID,
			&dept.UserId,
			&dept.TenantID,
			&dept.OrgID,
			&dept.CreateTime,
			&dept.UpdateTime,
			&dept.Name,
			&dept.Config,
		)
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

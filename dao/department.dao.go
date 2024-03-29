package dao

import (
	"database/sql"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func DepartmentCreate(db *sqlx.DB, model *protos.Department) (lastInsertId int64, err error) {
	rst, err := db.Exec("INSERT INTO departments (uid, tenant_id, parent_id, name, add_time) VALUES (?, ?, ?, ?, ?)",
		model.UserId, model.TenantID, model.ParentID, model.Name, time.Now())

	if err != nil {
		return -1, err
	}

	lastInsertId, err = rst.LastInsertId()

	return
}

func DepartmentDelete(db *sqlx.DB, id, tenantID uint64) (rowsAffected int64, err error) {
	rst, err := db.Exec("DELETE FROM departments WHERE id = ? AND tenant_id = ?", id, tenantID)
	if err != nil {
		return 0, err
	}

	return rst.RowsAffected()
}

func DepartmentUpdate(db *sqlx.DB, model *protos.Department) (rowsAffected int64, err error) {
	var rst sql.Result

	common.Logger.Sugar().Debugf("UPDATE departments SET name=? WHERE (id=? AND uid=? AND tenant_id=?)", model.Name, model.Id, model.UserId, model.TenantID)
	rst, err = db.Exec("UPDATE departments SET name=? WHERE (id=? AND tenant_id=?)", model.Name, model.Id, model.TenantID)
	if err != nil {
		return
	}

	return rst.RowsAffected()
}

func DepartmentUpdateConfig(db *sqlx.DB, model *protos.Department) (rowsAffected int64, err error) {
	var rst sql.Result

	common.Logger.Sugar().Debugf("UPDATE departments SET config=? WHERE (id=?)", model.Config, model.Id)
	rst, err = db.Exec("UPDATE departments SET config=? WHERE (id=?)", model.Config, model.Id)
	if err != nil {
		return
	}

	return rst.RowsAffected()
}

func DepartmentFind(db *sqlx.DB, id, tenantId uint64) (rr []protos.Department, err error) {
	tx := sq.Select("*").From("departments").Where("tenant_id = ?", tenantId).OrderBy("update_time desc")
	if id > 0 {
		tx = tx.Where("id = ?", id)
	}

	sql, args, err := tx.ToSql()
	common.Logger.Sugar().Debugf("DepartmentFind: %v %v %v", sql, args, err)

	err = db.Select(&rr, sql, args...)
	return
}

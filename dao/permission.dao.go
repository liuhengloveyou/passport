package dao

import (
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func PermissionCreate(db *sqlx.DB, m *protos.PermissionStruct) (id int64, e error) {
	rst, err := db.Exec("INSERT INTO permission (tenant_id, domain, title, value, add_time) VALUES (?, ?, ?, ?, ?)",
		m.TenantID, m.Domain, m.Title, m.Value, time.Now())

	if err != nil {
		if merr, ok := err.(*mysql.MySQLError); ok {
			if merr.Number == 1062 {
				return -1, common.ErrMysql1062
			}
		}
		return -1, err
	}

	id, e = rst.LastInsertId()

	return
}

func PermissionDelete(db *sqlx.DB, id, tenantID uint64) (int64, error) {
	rst, err := db.Exec("DELETE from permission WHERE id = ? AND tenant_id = ?", id, tenantID)
	if err != nil {
		return 0, err
	}

	return  rst.RowsAffected()
}

func PermissionList(db *sqlx.DB, tenantID uint64, domain string) (rr []protos.PermissionStruct, err error) {
	err = common.DB.Select(&rr, "SELECT * from permission WHERE tenant_id = ? AND domain = ?", tenantID, domain)

	return
}

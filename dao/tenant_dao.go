package dao

import (
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TenantInsert(tx *sqlx.Tx, m *protos.Tenant) (id int64, e error) {
	rst, err := tx.Exec("INSERT INTO tenant (uid, tenant_name, tenant_type, configuration, add_time) VALUES (?, ?, ?, ?, ?)",
		m.UID, m.TenantName, m.TenantType, m.Configuration, time.Now())
	if err != nil {
		if merr, ok := err.(*mysql.MySQLError); ok {
			if merr.Number == 1062 {
				return -1, common.ErrMysql1062
			}
		}
		return -1, err
	}

	if id, e = rst.LastInsertId(); e != nil {
		return
	}

	if rst, e = tx.Exec("UPDATE users SET tenant_id = ? WHERE (uid = ?) AND (tenant_id = 0)", id, m.UID); e != nil {
		return
	}
	row, _ := rst.RowsAffected()
	if row != 1 {
		return -1, common.ErrService
	}

	return
}

func TenantGetByID(tenantId uint64) (m *protos.Tenant, e error) {
	sqlStr := "SELECT * FROM tenant where id=?"

	var rst []protos.Tenant
	e = common.DB.Select(&rst, sqlStr, tenantId)
	if len(rst) == 1 {
		m = &rst[0]
	}

	return
}

func TenantUpdateConfiguration(m *protos.Tenant) error {
	_, err := common.DB.Exec("UPDATE tenant SET configuration = ? WHERE (id = ?) AND (update_time = ?)", m.Configuration, m.ID, m.UpdateTime)

	return err
}
package dao

import (
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	"github.com/go-sql-driver/mysql"
)

func TenantInsert(m *protos.Tenant) (id int64, e error) {
	sql := "INSERT INTO `tenant` (`uid`, `tenant_name`, `tenant_type`, `add_time`) VALUES (?, ?, ?, ?)"

	rst, err := common.DB.Exec(sql, m.UID, m.TenantName, m.TenantType, time.Now())
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

func TenantGetByID(tenant_id uint64) (m *protos.Tenant, e error) {
	sql := "SELECT * FROM tenant where id=?"

	var rst []protos.Tenant
	e = common.DB.Select(&rst, sql, tenant_id)
	if len(rst) == 1 {
		m = &rst[0]
	}

	return
}

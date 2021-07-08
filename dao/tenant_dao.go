package dao

import (
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func TenantInsert(tx *sqlx.Tx, m *protos.Tenant) (tenantID int64, e error) {
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

	if tenantID, e = rst.LastInsertId(); e != nil {
		return
	}

	if rst, e = tx.Exec("UPDATE users SET tenant_id = ? WHERE (uid = ?) AND (tenant_id = 0)", tenantID, m.UID); e != nil {
		return
	}
	row, err := rst.RowsAffected()
	if row != 1 || err != nil {
		return -1, common.ErrTenantLimit
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


func UserSelectByTenant(tenantID, page, pageSize uint64, nickname string, uids []uint64) (rr []protos.User, e error) {
	act := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "avatar_url", "gender", "addr", "ext", "add_time").From("users").Where(sq.Eq{"tenant_id": tenantID})
	if nickname != "" {
		act = act.Where(sq.Like{"nickname": "%"+nickname+"%"})
	} else if len(uids) > 0 {
		ors := make(sq.Or, len(uids))
		for i := 0; i < len(uids); i ++ {
			ors[i] = sq.Eq{"uid": uids[i]}
		}
		act = act.Where(ors)
	}

	sql, args, err := act.Offset((page - 1) * pageSize).Limit(pageSize).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)
	if e = common.DB.Select(&rr, sql, args...); e != nil {
		return
	}

	return
}

func UserCountByTenant(tenantID uint64, nickname string, uids []uint64) (r uint64, e error) {
	act := sq.Select("count(uid) as count").From("users").Where(sq.Eq{"tenant_id": tenantID})
	if nickname != "" {
		act = act.Where(sq.Like{"nickname": "%"+nickname+"%"})
	} else if len(uids) > 0 {
		ors := make(sq.Or, len(uids))
		for i := 0; i < len(uids); i ++ {
			ors[i] = sq.Eq{"uid": uids[i]}
		}
		act = act.Where(ors)
	}

	sql, args, err := act.ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)

	m := make([]int64, 0)
	if e = common.DB.Select(&m, sql, args...); e != nil {
		return
	}

	if len(m) == 1 {
		r = uint64(m[0])
	}

	return
}
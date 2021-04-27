package dao

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/liuhengloveyou/passport/protos"

	"github.com/liuhengloveyou/passport/common"

	builder "xorm.io/builder"
)

func UserInsert(p *protos.UserReq) (id int64, e error) {
	table := common.ServConfig.MysqlTableName
	data := builder.Eq{
		"password": p.Password,
		"add_time": time.Now(),
	}

	if p.Cellphone != "" {
		data["cellphone"] = p.Cellphone
	}
	if p.Email != "" {
		data["email"] = p.Email
	}
	if p.Nickname != "" {
		data["nickname"] = p.Nickname
	}

	sql, vals, err := builder.Insert(data).Into(table).ToSQL()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DB.Exec(sql, vals...)
	if err != nil {
		return -1, err
	}

	id, e = rst.LastInsertId()

	return
}

func UserUpdate(p *protos.UserReq) (rows int64, e error) {
	var rst sql.Result

	table := common.ServConfig.MysqlTableName
	where := builder.Eq{
		"uid": p.UID,
	}

	update := make(builder.Eq)
	if p.Cellphone != "" {
		update["cellphone"] = p.Cellphone
	}
	if p.Email != "" {
		update["email"] = p.Email
	}
	if p.Nickname != "" {
		update["nickname"] = p.Nickname
	}
	if p.AvatarURL != "" {
		update["avatar_url"] = p.AvatarURL
	}
	if p.Addr != "" {
		update["addr"] = p.Addr
	}
	if p.Gender == 1 || p.Gender == 2 {
		update["gender"] = p.Gender
	}

	sql, vals, err := builder.Update(update).From(table).Where(where).ToSQL()
	rst, e = common.DB.Exec(sql, vals...)
	if e != nil {
		return
	}

	rows, err = rst.RowsAffected()
	if err != nil {
		return -1, err
	}

	return rows, nil
}

func UserUpdatePWD(UID uint64, oldPWD, newPWD string) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec(fmt.Sprintf("UPDATE %s SET password=? WHERE (uid=? AND password=?)", common.ServConfig.MysqlTableName), newPWD, UID, oldPWD)
	if e != nil {
		return
	}

	return rst.RowsAffected()
}

func SetUserPWD(UID uint64, PWD string) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec(fmt.Sprintf("UPDATE %s SET password=? WHERE (uid=?)", common.ServConfig.MysqlTableName), PWD, UID)
	if e != nil {
		return
	}

	return rst.RowsAffected()
}

func UserUpdateTenantID(UID, tenantID, currTenantID uint64) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec("UPDATE users SET tenant_id = ? WHERE (uid = ?) AND (tenant_id = ?)", tenantID, UID, currTenantID)
	if e != nil {
		return
	}

	return rst.RowsAffected()
}

func UserDelete(tx *sql.Tx) (int64, error) {
	return -1, nil
}

func UserSelectByID(uid uint64) (r *protos.User, e error) {
	r = &protos.User{}
	e = common.DB.Get(r, "SELECT uid, tenant_id, cellphone, email, nickname, avatar_url, gender, addr, tags FROM users WHERE uid = ?", uid)
	return
}

func UserSelectByTenantID(tenantID uint64) (rr []protos.User, e error) {
	e = common.DB.Select(&rr, "SELECT uid, tenant_id, cellphone, email, nickname, avatar_url, gender, addr, tags FROM users where tenant_id = ?", tenantID)
	return
}

func UserSelect(p *protos.UserReq, pageNo, pageSize int) (rr []protos.User, e error) {
	table := common.ServConfig.MysqlTableName
	where := builder.Eq{}
	if p.UID > 0 {
		where["uid"] = p.UID
	}
	if p.Cellphone != "" {
		where["cellphone"] = p.Cellphone
	}
	if p.Email != "" {
		where["email"] = p.Email
	}
	if p.Nickname != "" {
		where["nickname"] = p.Nickname
	}

	cond, values, err := builder.MySQL().Select("uid, tenant_id, cellphone, email, nickname, password, avatar_url, gender, addr, add_time, update_time, tags").Where(where).From(table).ToSQL()
	common.Logger.Sugar().Debugf("%v %v %v", cond, values, err)
	if e = common.DB.Select(&rr, cond, values...); e != nil {
		return
	}

	return rr, nil
}

func BusinessSelect(p *protos.UserReq, models interface{}, pageNo, pageSize int) (e error) {
	table := common.ServConfig.MysqlTableName
	where := make(builder.Eq)
	if p.UID > 0 {
		where["uid"] = p.UID
	}
	if p.Cellphone != "" {
		where["cellphone"] = p.Cellphone
	}
	if p.Email != "" {
		where["email"] = p.Email
	}
	if p.Nickname != "" {
		where["nickname"] = p.Nickname
	}

	cond, values, _ := builder.Select("*").From(table).Where(where).OrderBy("update_time desc").Limit((pageNo-1)*pageSize, pageSize).ToSQL()
	e = common.DB.Select(models, cond, values...)

	return
}

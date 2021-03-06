package dao

import (
	"database/sql"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	builder "xorm.io/builder"
)

func UserInsert(p *protos.UserReq) (id int64, e error) {
	table := "users"
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

	table := "users"
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


func UserUpdateExt(uid uint64, updateTime *time.Time, ext *protos.MapStruct) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec("UPDATE users SET ext=? WHERE (uid=? AND update_time=?)", ext, uid, updateTime)
	if e != nil {
		return
	}

	return rst.RowsAffected()
}


func UserUpdatePWD(UID uint64, oldPWD, newPWD string) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec("UPDATE users SET password=? WHERE (uid=? AND password=?)", newPWD, UID, oldPWD)
	if e != nil {
		return
	}

	return rst.RowsAffected()
}

func SetUserPWD(UID, tenantId uint64, PWD string) (rows int64, e error) {
	var rst sql.Result

	if tenantId  <= 0 {
		rst, e = common.DB.Exec("UPDATE users SET password=? WHERE (uid=?)", PWD, UID)
	} else {
		rst, e = common.DB.Exec("UPDATE users SET password=? WHERE (uid=?) AND (tenant_id = ?)", PWD, UID, tenantId)
	}

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

func UserUpdateLoginTime(UID uint64, t *time.Time) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec("UPDATE users SET login_time = ? WHERE (uid = ?)", t, UID)
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
	e = common.DB.Get(r, "SELECT uid, tenant_id, cellphone, email, nickname, avatar_url, gender, addr, add_time, update_time, ext FROM users WHERE uid = ?", uid)
	return
}

func UserSelect(p *protos.UserReq, pageNo, pageSize int) (rr []protos.User, e error) {
	table := "users"
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

	cond, values, err := builder.MySQL().Select("uid, tenant_id, cellphone, email, nickname, password, avatar_url, gender, addr, add_time, update_time, ext").Where(where).From(table).ToSQL()
	common.Logger.Sugar().Debugf("%v %v %v", cond, values, err)
	if e = common.DB.Select(&rr, cond, values...); e != nil {
		return
	}

	return rr, nil
}

func BusinessSelect(p *protos.UserReq, models interface{}, pageNo, pageSize int) (e error) {
	table := "users"
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
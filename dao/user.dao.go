package dao

import (
	"database/sql"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	sq "github.com/Masterminds/squirrel"
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
	if len(p.Ext) > 0 {
		data["ext"] = p.Ext
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
	if p.AvatarURL != "" {
		update["avatar_url"] = p.AvatarURL
	}

	sql, vals, err := builder.Update(update).From(table).Where(where).ToSQL()
	if sql == "" {
		return
	}
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

func UserUpdateExt(uid uint64, ext *protos.MapStruct) (rows int64, e error) {
	var rst sql.Result

	rst, e = common.DB.Exec("UPDATE users SET ext=? WHERE (uid=?)", ext, uid)
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

	if tenantId <= 0 {
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
	e = common.DB.Get(r, "SELECT * FROM users WHERE uid = ?", uid)

	return
}

func UserSelectOne(p *protos.UserReq) (r *protos.User, e error) {
	eq := sq.Eq{}

	if p.UID > 0 {
		eq["uid"] = p.UID
	}
	if p.Cellphone != "" {
		eq["cellphone"] = p.Cellphone
	}
	if p.Email != "" {
		eq["email"] = p.Email
	}
	if p.Nickname != "" {
		eq["nickname"] = p.Nickname
	}

	sql, args, err := sq.Select("*").Limit(1).Where(eq).From("users").ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)

	var rr []protos.User
	if e = common.DB.Select(&rr, sql, args...); e != nil {
		return
	}
	if len(rr) == 0 {
		common.Logger.Sugar().Warnf("%v %v %v", sql, args, len(rr))
		return nil, nil
	}

	return &rr[0], nil
}

func UserSelect(p *protos.UserReq, pageNo, pageSize uint64) (rr []protos.User, e error) {
	if pageNo < 1 {
		pageNo = 1
	}

	eq := sq.Eq{}
	if p.UID > 0 {
		eq["uid"] = p.UID
	}
	if p.Cellphone != "" {
		eq["cellphone"] = p.Cellphone
	}
	if p.Email != "" {
		eq["email"] = p.Email
	}
	if p.Nickname != "" {
		eq["nickname"] = p.Nickname
	}

	sql, args, err := sq.Select("*").Offset((pageNo - 1) * pageSize).Limit(pageSize).Where(eq).From("users").ToSql()
	// cond, values, err := builder.MySQL().Select("uid, tenant_id, cellphone, email, nickname, password, avatar_url, gender, addr, add_time, update_time, ext").Where(where).From(table).ToSQL()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)
	if e = common.DB.Select(&rr, sql, args...); e != nil {
		return
	}

	return rr, nil
}

func UserSearchLite(p *protos.UserReq, pageNo, pageSize uint64) (rr []protos.User, e error) {
	if pageNo < 1 {
		pageNo = 1
	}

	or := sq.Or{}
	if p.Cellphone != "" {
		or = append(or, sq.Like{"cellphone": "%" + p.Cellphone + "%"})
	}
	if p.Email != "" {
		or = append(or, sq.Like{"email": "%" + p.Email + "%"})
	}
	if p.Nickname != "" {
		or = append(or, sq.Like{"nickname": "%" + p.Nickname + "%"})
	}
	and := sq.And{sq.Eq{"tenant_id": p.TenantID}, or}

	sql, args, err := sq.Select("uid,tenant_id,nickname,avatar_url").Offset((pageNo - 1) * pageSize).Limit(pageSize).Where(and).From("users").ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)
	if e = common.DB.Select(&rr, sql, args...); e != nil {
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

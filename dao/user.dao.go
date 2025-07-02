package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"xorm.io/builder"

	sq "github.com/Masterminds/squirrel"
)

// UserInsert 插入用户数据，支持可选的事务参数
func UserInsert(p *protos.UserReq, tx *pgx.Tx) (uid int64, e error) {
	table := "users"
	data := make(map[string]interface{})

	// 必需字段
	data["password"] = p.Password
	data["create_time"] = time.Now()
	data["update_time"] = time.Now()

	// 处理租户ID
	if p.TenantID > 0 {
		data["tenant_id"] = p.TenantID
	}
	// 注意：不要设置UID字段，让数据库自动生成
	// 如果p.UID有值，会导致主键冲突错误

	// 可选字段
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
	if len(p.WxOpenId) > 0 {
		data["wx_openid"] = p.WxOpenId
	}

	// 使用 RETURNING 子句获取插入记录的 uid
	sql, vals, err := sq.Insert(table).SetMap(data).Suffix("RETURNING uid").PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	// 根据是否传入事务参数选择执行方式
	if tx != nil {
		// 在事务中执行
		err = (*tx).QueryRow(context.Background(), sql, vals...).Scan(&uid)
		common.Logger.Sugar().Debugf("tx.QueryRow: uid=%v err=%v\n", uid, err)
	} else {
		// 使用连接池执行
		err = common.DBPool.QueryRow(context.Background(), sql, vals...).Scan(&uid)
		common.Logger.Sugar().Debugf("db.QueryRow: uid=%v err=%v\n", uid, err)
	}

	if err != nil {
		return -1, err
	}

	return uid, nil
}

func UserUpdate(p *protos.UserReq) (rows int64, e error) {
	table := "users"
	where := builder.Eq{
		"uid": p.UID,
	}

	update := make(map[string]interface{})
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

	// 如果没有需要更新的字段，直接返回
	if len(update) == 0 {
		return 0, nil
	}

	sql, vals, err := sq.Update(table).SetMap(update).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()

	return rows, nil
}

func UserUpdateExt(uid uint64, ext *protos.MapStruct) (rows int64, e error) {
	table := "users"
	where := builder.Eq{
		"uid": uid,
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).Set("ext", ext).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()

	return rows, nil
}

func UserUpdatePWD(UID uint64, oldPWD, newPWD string) (rows int64, e error) {
	table := "users"
	where := builder.Eq{
		"uid":      UID,
		"password": oldPWD,
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).Set("password", newPWD).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()
	return rows, nil
}

func UserUpdatePWDByCellphone(cellphone, newPWD string) (rows int64, e error) {
	table := "users"
	where := builder.Eq{
		"cellphone": cellphone,
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).Set("password", newPWD).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()
	return rows, nil
}

func UserUpdateWxOpenIdByCellphone(cellphone, wxOpenId string) (rows int64, e error) {
	table := "users"
	where := builder.Eq{
		"cellphone": cellphone,
	}

	update := map[string]interface{}{
		"wx_openid":   wxOpenId,
		"update_time": time.Now(),
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).SetMap(update).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()
	return rows, nil
}

func SetUserPWD(uid uint64, tid uint64, pwd string) (r int64, e error) {
	table := "users"
	update := map[string]interface{}{
		"password": pwd,
	}

	// 构建 WHERE 条件
	where := sq.Eq{
		"uid": uid,
	}

	// 如果有租户ID，添加到条件中
	if tid > 0 {
		where["tenant_id"] = tid
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).SetMap(update).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	r = rst.RowsAffected()
	return r, nil
}

func UserUpdateTenantID(UID, tenantID, currTenantID uint64) (rows int64, e error) {
	table := "users"
	update := map[string]interface{}{
		"tenant_id": tenantID,
	}

	// 构建 WHERE 条件
	where := sq.Eq{
		"uid":       UID,
		"tenant_id": currTenantID,
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).SetMap(update).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()
	return rows, nil
}

func UserUpdateLoginTime(UID uint64, t *time.Time) (rows int64, e error) {
	table := "users"
	update := map[string]interface{}{
		"login_time": t,
	}

	// 构建 WHERE 条件
	where := sq.Eq{
		"uid": UID,
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Update(table).SetMap(update).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	rows = rst.RowsAffected()
	return rows, nil
}

func UserDelete(uid uint64, tid uint64) (r int64, e error) {
	table := "users"

	// 构建 WHERE 条件
	where := sq.Eq{
		"uid":       uid,
		"tenant_id": tid,
	}

	// 使用squirrel构建SQL
	sql, vals, err := sq.Delete(table).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, vals, err)
	if err != nil {
		return -1, err
	}

	rst, err := common.DBPool.Exec(context.Background(), sql, vals...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return -1, err
	}

	r = rst.RowsAffected()
	return r, nil
}

func UserQueryByID(uid uint64) (r *protos.User, e error) {
	table := "users"

	// 构建 WHERE 条件
	where := sq.Eq{
		"uid": uid,
	}

	// 使用squirrel构建SQL，明确指定查询字段
	sql, args, err := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "wx_openid",
		"password", "avatar_url", "gender", "addr", "ext",
		"create_time", "update_time", "login_time").From(table).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	// 检查是否有结果
	if !rows.Next() {
		return nil, nil // 没有找到记录
	}

	// 扫描结果到结构体
	r = &protos.User{}
	err = rows.Scan(
		&r.UID,
		&r.TenantID,
		&r.Cellphone,
		&r.Email,
		&r.Nickname,
		&r.WxOpenId,
		&r.Password,
		&r.AvatarURL,
		&r.Gender,
		&r.Addr,
		&r.Ext,
		&r.CreateTime, // create_time
		&r.UpdateTime, // update_time
		&r.LoginTime,  // login_time
	)
	if err != nil {
		common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
		return nil, err
	}

	// 检查是否有多余的结果（应该只有一条记录）
	if rows.Next() {
		common.Logger.Sugar().Warnf("Multiple records found for uid: %d\n", uid)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return nil, err
	}

	return r, nil
}

func UserQueryOne(p *protos.UserReq) (r *protos.User, e error) {
	table := "users"
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
	if p.WxOpenId != "" {
		eq["wx_openid"] = p.WxOpenId
	}

	// 使用squirrel构建SQL，明确指定查询字段
	sql, args, err := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "wx_openid",
		"password", "avatar_url", "gender", "addr", "ext",
		"create_time", "update_time", "login_time").From(table).Where(eq).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	// 检查是否有结果
	if !rows.Next() {
		return nil, nil // 没有找到记录
	}

	// 扫描结果到结构体
	r = &protos.User{}
	err = rows.Scan(
		&r.UID,
		&r.TenantID,
		&r.Cellphone,
		&r.Email,
		&r.Nickname,
		&r.WxOpenId,
		&r.Password,
		&r.AvatarURL,
		&r.Gender,
		&r.Addr,
		&r.Ext,
		&r.CreateTime, // create_time
		&r.UpdateTime, // update_time
		&r.LoginTime,  // login_time
	)
	if err != nil {
		common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
		return nil, err
	}

	// 检查是否有多余的结果（应该只有一条记录）
	if rows.Next() {
		common.Logger.Sugar().Warnf("Multiple records found for query: %v\n", eq)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return nil, err
	}

	return r, nil
}

func UserQuery(p *protos.UserReq, pageNo, pageSize uint64) (rr []protos.User, e error) {
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
	if len(p.WxOpenId) > 0 {
		eq["wx_openid"] = p.WxOpenId
	}

	sql, args, err := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "wx_openid",
		"password", "avatar_url", "gender", "addr", "ext",
		"create_time", "update_time", "login_time").Offset((pageNo - 1) * pageSize).Limit(pageSize).Where(eq).From("users").PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	// 遍历结果集
	for rows.Next() {
		var user protos.User
		err = rows.Scan(
			&user.UID,
			&user.TenantID,
			&user.Cellphone,
			&user.Email,
			&user.Nickname,
			&user.WxOpenId,
			&user.Password,
			&user.AvatarURL,
			&user.Gender,
			&user.Addr,
			&user.Ext,
			&user.CreateTime, // create_time
			&user.UpdateTime, // update_time
			&user.LoginTime,  // login_time
		)
		if err != nil {
			common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
			return nil, err
		}
		rr = append(rr, user)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return nil, err
	}

	return rr, nil
}

func UserLiteQuery(p *protos.UserReq, pageNo, pageSize uint64) (rr []protos.UserLite, e error) {
	if pageNo < 1 {
		pageNo = 1
	}

	and := sq.And{sq.Eq{"tenant_id": p.TenantID}}
	if p.UID != 0 {
		and = append(and, sq.Eq{"uid": p.UID})
	}

	if len(p.WxOpenId) > 0 {
		and = append(and, sq.Eq{"wx_openid": p.WxOpenId})
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

	if len(or) > 0 {
		and = append(and, or)
	}

	sql, args, err := sq.Select("uid,tenant_id,nickname,avatar_url,wx_openid,ext").Offset((pageNo - 1) * pageSize).Limit(pageSize).Where(and).From("users").PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	// 遍历结果集
	for rows.Next() {
		var user protos.UserLite
		err = rows.Scan(
			&user.UID,
			&user.TenantID,
			&user.Nickname,
			&user.AvatarURL,
			&user.WxOpenId,
			&user.Ext,
		)
		if err != nil {
			common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
			return nil, err
		}
		rr = append(rr, user)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return nil, err
	}

	return rr, nil
}

func BusinessQuery(p *protos.UserReq, models interface{}, pageNo, pageSize uint64) (e error) {
	table := "users"
	where := make(sq.Eq)
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

	// 使用squirrel构建SQL，明确指定查询字段
	sql, args, err := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "wx_openid",
		"password", "avatar_url", "gender", "addr", "ext",
		"create_time", "update_time", "login_time").From(table).Where(where).OrderBy("update_time desc").Offset((pageNo - 1) * pageSize).Limit(pageSize).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return err
	}
	defer rows.Close()

	// 将查询结果映射到传入的models切片中
	users, ok := models.(*[]protos.User)
	if !ok {
		return fmt.Errorf("models is not a pointer to []protos.User")
	}

	*users = []protos.User{}
	for rows.Next() {
		var user protos.User
		// 根据数据库表结构的列顺序进行扫描
		err = rows.Scan(
			&user.UID, &user.TenantID, &user.Cellphone, &user.Email, &user.Nickname,
			&user.WxOpenId, &user.Password, &user.AvatarURL, &user.Gender, &user.Addr,
			&user.Ext, &user.CreateTime, &user.UpdateTime, &user.LoginTime)
		if err != nil {
			common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
			return err
		}
		*users = append(*users, user)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows.Err: %v\n", err)
		return err
	}

	return nil
}

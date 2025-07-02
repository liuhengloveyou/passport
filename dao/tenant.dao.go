package dao

import (
	"context"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func TenantInsert(tx pgx.Tx, m *protos.Tenant) (tenantID uint64, e error) {
	// 使用 RETURNING id 子句获取新插入记录的 ID
	err := tx.QueryRow(context.Background(), "INSERT INTO tenants (uid, parent_id, tenant_name, tenant_type, info, configuration, create_time) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		m.UID, m.ParentID, m.TenantName, m.TenantType, m.Info, m.Configuration, time.Now()).Scan(&tenantID)

	if err != nil {
		// PostgreSQL 错误处理
		common.Logger.Sugar().Errorf("Failed to insert tenants: %v", err)
		return 0, err
	}

	// 更新用户的租户ID
	rst, e := tx.Exec(context.Background(), "UPDATE users SET tenant_id = $1 WHERE (uid = $2) AND (tenant_id = 0)", tenantID, m.UID)
	if e != nil {
		return 0, e
	}

	row := rst.RowsAffected()
	if row != 1 {
		return 0, common.ErrTenantLimit
	}

	return
}

func TenantGetByID(tenantId uint64) (m *protos.Tenant, e error) {
	table := "tenants"

	// 构建 WHERE 条件
	where := sq.Eq{
		"id": tenantId,
	}

	// 使用squirrel构建SQL，明确指定字段顺序
	sql, args, err := sq.Select("id", "uid", "parent_id", "tenant_name", "tenant_type", "info", "configuration", "create_time", "update_time").From(table).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	var rst []protos.Tenant
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	// 手动扫描结果到结构体切片
	for rows.Next() {
		var t protos.Tenant
		err = rows.Scan(
			&t.ID,
			&t.UID,
			&t.ParentID,
			&t.TenantName,
			&t.TenantType,
			&t.Info,
			&t.Configuration,
			&t.CreateTime,
			&t.UpdateTime,
		)
		if err != nil {
			common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
			return nil, err
		}
		rst = append(rst, t)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return nil, err
	}

	// 返回第一个结果
	if len(rst) == 1 {
		m = &rst[0]
	}

	return
}

func TenantCount() (r uint64, e error) {
	// 使用squirrel构建SQL
	sql, args, err := sq.Select("count(id)").From("tenants").PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return 0, err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return 0, err
	}
	defer rows.Close()

	// 检查是否有结果
	if !rows.Next() {
		common.Logger.Sugar().Warnf("No result found for count query\n")
		return 0, nil // 没有找到记录
	}

	// 扫描结果
	var count int64
	err = rows.Scan(&count)
	if err != nil {
		common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
		return 0, err
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return 0, err
	}

	return uint64(count), nil
}

func TenantCountByAncestorID(ancestorID uint64) (r uint64, e error) {
	// 使用squirrel构建SQL
	sql, args, err := sq.Select("count(id)").From("tenant_closure").Where(sq.Eq{"ancestor_id": ancestorID}).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return 0, err
	}

	// 执行查询
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return 0, err
	}
	defer rows.Close()

	// 检查是否有结果
	if !rows.Next() {
		common.Logger.Sugar().Warnf("No result found for count query\n")
		return 0, nil // 没有找到记录
	}

	// 扫描结果
	var count int64
	err = rows.Scan(&count)
	if err != nil {
		common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
		return 0, err
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return 0, err
	}

	return uint64(count), nil
}

func TenantList(parentID, page, pageSize uint64) (rr []protos.Tenant, e error) {
	// 创建查询构建器
	query := sq.Select("id", "uid", "parent_id", "tenant_name", "tenant_type", "info", "configuration", "create_time", "update_time").From("tenants").PlaceholderFormat(sq.Dollar)

	// 只有当parentID不为0时，才添加到where条件中
	if parentID != 0 {
		query = query.Where(sq.Eq{"parent_id": parentID})
	}

	// 添加分页和排序
	query = query.Offset((page - 1) * pageSize).Limit(pageSize).OrderBy("id")

	// 生成SQL
	sql, args, err := query.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return nil, err
	}
	defer rows.Close()

	rr = []protos.Tenant{}
	for rows.Next() {
		var tenant protos.Tenant
		err = rows.Scan(&tenant.ID, &tenant.UID, &tenant.ParentID, &tenant.TenantName, &tenant.TenantType, &tenant.Info, &tenant.Configuration, &tenant.CreateTime, &tenant.UpdateTime)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to scan row: %v", err)
			return nil, err
		}
		rr = append(rr, tenant)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("Error iterating rows: %v", err)
		return nil, err
	}

	return
}

// TenantListByDescendants 查询指定租户的后代租户列表
func TenantListByAncestorID(ancestorID, page, pageSize uint64) (rr []protos.Tenant, e error) {
	// 创建查询构建器，通过tenant_closure表查询后代租户
	query := sq.Select("t.id", "t.uid", "t.parent_id", "t.tenant_name", "t.tenant_type", "t.info", "t.configuration", "t.create_time", "t.update_time", "tc.depth").
		From("tenants t").
		Join("tenant_closure tc ON t.id = tc.descendant_id").
		Where(sq.Eq{"tc.ancestor_id": ancestorID}).
		PlaceholderFormat(sq.Dollar)

	// 添加分页和排序
	query = query.Offset((page - 1) * pageSize).Limit(pageSize).OrderBy("t.id desc")

	// 生成SQL
	sql, args, err := query.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return nil, err
	}
	defer rows.Close()

	rr = []protos.Tenant{}
	for rows.Next() {
		var tenant protos.Tenant
		err = rows.Scan(&tenant.ID, &tenant.UID, &tenant.ParentID, &tenant.TenantName, &tenant.TenantType, &tenant.Info, &tenant.Configuration, &tenant.CreateTime, &tenant.UpdateTime, &tenant.Depth)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to scan row: %v", err)
			return nil, err
		}
		rr = append(rr, tenant)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("Error iterating rows: %v", err)
		return nil, err
	}

	return
}

// TenantQuery 根据条件查询租户列表
func TenantQuery(tenantName, cellphone string, page, pageSize uint64) (rr []protos.Tenant, e error) {
	// 创建查询构建器
	query := sq.Select("t.id", "t.uid", "t.parent_id", "t.tenant_name", "t.tenant_type", "t.info", "t.configuration", "t.create_time", "t.update_time").From("tenants t").PlaceholderFormat(sq.Dollar)

	// 添加查询条件
	if tenantName != "" {
		query = query.Where(sq.Like{"t.tenant_name": "%" + tenantName + "%"})
	}

	// 如果需要按管理员手机号查询，需要关联用户表
	if cellphone != "" {
		// 通过租户信息中的AdminCellphone字段查询
		query = query.Where("t.info->>'adminCellphone' LIKE ?", "%"+cellphone+"%")
	}

	// 添加分页和排序
	query = query.Offset((page - 1) * pageSize).Limit(pageSize).OrderBy("t.id")

	// 生成SQL
	sql, args, err := query.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return nil, err
	}
	defer rows.Close()

	rr = []protos.Tenant{}
	for rows.Next() {
		var tenant protos.Tenant
		err = rows.Scan(&tenant.ID, &tenant.UID, &tenant.ParentID, &tenant.TenantName, &tenant.TenantType, &tenant.Info, &tenant.Configuration, &tenant.CreateTime, &tenant.UpdateTime)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to scan row: %v", err)
			return nil, err
		}
		rr = append(rr, tenant)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("Error iterating rows: %v", err)
		return nil, err
	}

	return
}

func TenantUpdateConfiguration(m *protos.Tenant) error {
	_, err := common.DBPool.Exec(context.Background(), "UPDATE tenants SET configuration = $1 WHERE (id = $2) AND (update_time = $3)", m.Configuration, m.ID, m.UpdateTime)

	return err
}

// TenantUpdateParentID 更新租户的父租户ID，支持可选事务参数
func TenantUpdateParentID(tenantID, parentID uint64, tx *pgx.Tx) error {
	// 使用squirrel构建SQL
	sql, args, err := sq.Update("tenants").Set("parent_id", parentID).Set("update_time", time.Now()).Where(sq.Eq{"id": tenantID}).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return err
	}

	// 根据是否提供事务参数选择执行方式
	if tx != nil {
		// 在事务中执行更新
		_, err = (*tx).Exec(context.Background(), sql, args...)
		common.Logger.Sugar().Debugf("tx.exec: %v\n", err)
	} else {
		// 直接执行更新
		_, err = common.DBPool.Exec(context.Background(), sql, args...)
		common.Logger.Sugar().Debugf("db.exec: %v\n", err)
	}

	return err
}

func UserQueryByTenant(tenantID, page, pageSize uint64, nickname string, uids []uint64) (rr []protos.User, e error) {
	act := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "avatar_url", "gender", "addr", "ext", "create_time").From("users").PlaceholderFormat(sq.Dollar).Where(sq.Eq{"tenant_id": tenantID})
	if nickname != "" {
		act = act.Where(sq.Like{"nickname": "%" + nickname + "%"})
	} else if len(uids) > 0 {
		ors := make(sq.Or, len(uids))
		for i := 0; i < len(uids); i++ {
			ors[i] = sq.Eq{"uid": uids[i]}
		}
		act = act.Where(ors)
	}

	sql, args, err := act.Offset((page - 1) * pageSize).Limit(pageSize).ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return nil, err
	}
	defer rows.Close()

	rr = []protos.User{}
	for rows.Next() {
		var user protos.User
		err := rows.Scan(&user.UID, &user.TenantID, &user.Cellphone, &user.Email, &user.Nickname, &user.AvatarURL, &user.Gender, &user.Addr, &user.Ext, &user.CreateTime)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to scan row: %v", err)
			return nil, err
		}
		rr = append(rr, user)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("Error iterating rows: %v", err)
		return nil, err
	}

	return
}

func UserCountByTenant(tenantID uint64, nickname string, uids []uint64) (r uint64, e error) {
	act := sq.Select("count(uid) as count").From("users").PlaceholderFormat(sq.Dollar).Where(sq.Eq{"tenant_id": tenantID})
	if nickname != "" {
		act = act.Where(sq.Like{"nickname": "%" + nickname + "%"})
	} else if len(uids) > 0 {
		ors := make(sq.Or, len(uids))
		for i := 0; i < len(uids); i++ {
			ors[i] = sq.Eq{"uid": uids[i]}
		}
		act = act.Where(ors)
	}

	sql, args, err := act.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return 0, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to execute query: %v", err)
		return 0, err
	}
	defer rows.Close()

	var count int64
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to scan row: %v", err)
			return 0, err
		}
		r = uint64(count)
	}

	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("Error iterating rows: %v", err)
		return 0, err
	}

	return
}

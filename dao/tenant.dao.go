package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
	"go.uber.org/zap"

	sq "github.com/Masterminds/squirrel"
)

func TenantInsert(tx database.Tx, m *protos.Tenant) (tenantID uint64, e error) {
	ctx := context.Background()
	common.Logger.Info("TenantInsert start",
		zap.Uint64("uid", m.UID),
		zap.String("tenant_name", m.TenantName),
		zap.String("tenant_type", m.TenantType),
		zap.Bool("has_info", len(m.Info) > 0),
		zap.Bool("has_configuration", m.Configuration != nil),
	)

	// 准备插入数据
	data := map[string]interface{}{
		"uid":           m.UID,
		"tenant_name":   m.TenantName,
		"tenant_type":   m.TenantType,
		"info":          m.Info,
		"configuration": m.Configuration,
		"create_time":   time.Now(),
	}

	dialect := database.NewDialect(common.DB.DriverType())
	placeholderFormat := database.GetPlaceholderFormat(common.DB.DriverType())
	insertBuilder := sq.Insert("tenants").SetMap(data).PlaceholderFormat(placeholderFormat)

	if dialect.SupportsReturning() {
		insertSQL, insertVals, err := insertBuilder.Suffix("RETURNING id").ToSql()
		if err != nil {
			common.Logger.Error("TenantInsert build SQL failed",
				zap.Error(err),
				zap.String("tenant_name", m.TenantName),
				zap.String("tenant_type", m.TenantType),
			)
			return 0, err
		}
		common.Logger.Debug("TenantInsert execute insert with returning",
			zap.String("sql", insertSQL),
			zap.Any("args", insertVals),
			zap.String("tenant_name", m.TenantName),
			zap.String("tenant_type", m.TenantType),
			zap.Uint64("uid", m.UID),
		)
		if err = tx.QueryRow(ctx, insertSQL, insertVals...).Scan(&tenantID); err != nil {
			common.Logger.Sugar().Errorf("Failed to insert tenants with returning id: %v", err)
			common.Logger.Error("TenantInsert insert failed",
				zap.Error(err),
				zap.String("tenant_name", m.TenantName),
				zap.String("tenant_type", m.TenantType),
				zap.Uint64("uid", m.UID),
			)
			return 0, err
		}
	} else {
		insertSQL, insertVals, err := insertBuilder.ToSql()
		if err != nil {
			common.Logger.Error("TenantInsert build SQL failed",
				zap.Error(err),
				zap.String("tenant_name", m.TenantName),
				zap.String("tenant_type", m.TenantType),
			)
			return 0, err
		}
		common.Logger.Debug("TenantInsert execute insert",
			zap.String("sql", insertSQL),
			zap.Any("args", insertVals),
			zap.String("tenant_name", m.TenantName),
			zap.String("tenant_type", m.TenantType),
			zap.Uint64("uid", m.UID),
		)
		if _, err = tx.Exec(ctx, insertSQL, insertVals...); err != nil {
			common.Logger.Sugar().Errorf("Failed to insert tenants: %v", err)
			common.Logger.Error("TenantInsert insert failed",
				zap.Error(err),
				zap.String("tenant_name", m.TenantName),
				zap.String("tenant_type", m.TenantType),
				zap.Uint64("uid", m.UID),
			)
			return 0, err
		}
		id, idErr := dialect.LastInsertID(ctx, common.DB, "tenants")
		if idErr != nil {
			common.Logger.Error("TenantInsert last insert id failed",
				zap.Error(idErr),
				zap.String("tenant_name", m.TenantName),
				zap.String("tenant_type", m.TenantType),
				zap.Uint64("uid", m.UID),
			)
			return 0, idErr
		}
		tenantID = uint64(id)
	}
	common.Logger.Info("TenantInsert tenant row created",
		zap.Uint64("tenant_id", tenantID),
		zap.String("tenant_name", m.TenantName),
		zap.String("tenant_type", m.TenantType),
		zap.Uint64("uid", m.UID),
	)

	// 可选管理员场景：uid=0 表示先仅创建租户，不绑定管理员账号。
	if m.UID <= 0 {
		common.Logger.Info("TenantInsert skip bind admin user", zap.Uint64("tenant_id", tenantID))
		return
	}

	// 更新用户的租户ID
	// 构建UPDATE语句，使用正确的占位符
	updateSQL, updateVals, err := sq.Update("users").
		Set("tenant_id", tenantID).
		Where(sq.And{sq.Eq{"uid": m.UID}, sq.Eq{"tenant_id": 0}}).
		PlaceholderFormat(placeholderFormat).
		ToSql()
	if err != nil {
		common.Logger.Error("TenantInsert build user update SQL failed",
			zap.Error(err),
			zap.Uint64("tenant_id", tenantID),
			zap.Uint64("uid", m.UID),
		)
		return 0, err
	}
	common.Logger.Debug("TenantInsert execute user bind update",
		zap.String("sql", updateSQL),
		zap.Any("args", updateVals),
		zap.Uint64("tenant_id", tenantID),
		zap.Uint64("uid", m.UID),
	)

	rst, e := tx.Exec(ctx, updateSQL, updateVals...)
	if e != nil {
		common.Logger.Error("TenantInsert bind user failed",
			zap.Error(e),
			zap.Uint64("tenant_id", tenantID),
			zap.Uint64("uid", m.UID),
		)
		return 0, e
	}

	row, _ := rst.RowsAffected()
	if row != 1 {
		common.Logger.Warn("TenantInsert bind user unexpected rows affected",
			zap.Int64("rows_affected", row),
			zap.Uint64("tenant_id", tenantID),
			zap.Uint64("uid", m.UID),
		)
		return 0, common.ErrTenantLimit
	}
	common.Logger.Info("TenantInsert success",
		zap.Uint64("tenant_id", tenantID),
		zap.Uint64("uid", m.UID),
	)

	return
}

// TenantNameExists 是否已有同名租户（与 tenants.tenant_name 唯一约束一致）。
func TenantNameExists(tenantName string) (exists bool, e error) {
	name := strings.TrimSpace(tenantName)
	if name == "" {
		return false, nil
	}
	ph := database.GetPlaceholderFormat(common.DB.DriverType())
	sql, args, err := sq.Select("1").From("tenants").Where(sq.Eq{"tenant_name": name}).Limit(1).PlaceholderFormat(ph).ToSql()
	if err != nil {
		return false, err
	}
	var one int
	err = common.DB.QueryRow(context.Background(), sql, args...).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func TenantGetByID(tenantId uint64) (m *protos.Tenant, e error) {
	table := "tenants t"

	// 构建 WHERE 条件
	where := sq.Eq{
		"id": tenantId,
	}

	// 使用squirrel构建SQL，明确指定字段顺序
	sql, args, err := sq.Select(
		"t.id",
		"t.uid",
		"COALESCE(tp.ancestor_id, 0) AS parent_id",
		"t.tenant_name",
		"t.tenant_type",
		"t.info",
		"t.configuration",
		"t.create_time",
		"t.update_time",
	).
		From(table).
		LeftJoin("tenant_closure tp ON tp.descendant_id = t.id AND tp.depth = 1").
		Where(where).
		PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	var rst []protos.Tenant
	rows, err := common.DB.Query(context.Background(), sql, args...)
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
	sql, args, err := sq.Select("count(id)").From("tenants").PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return 0, err
	}

	// 执行查询
	rows, err := common.DB.Query(context.Background(), sql, args...)
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
	sql, args, err := sq.Select("count(id)").From("tenant_closure").Where(sq.Eq{"ancestor_id": ancestorID}).PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return 0, err
	}

	// 执行查询
	rows, err := common.DB.Query(context.Background(), sql, args...)
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

func TenantList(page, pageSize uint64) (rr []protos.Tenant, e error) {
	// 创建查询构建器
	query := sq.Select(
		"t.id",
		"t.uid",
		"COALESCE(tp.ancestor_id, 0) AS parent_id",
		"t.tenant_name",
		"t.tenant_type",
		"t.info",
		"t.configuration",
		"t.create_time",
		"t.update_time",
	).
		From("tenants t").
		LeftJoin("tenant_closure tp ON tp.descendant_id = t.id AND tp.depth = 1").
		PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType()))

	// 添加分页和排序
	query = query.Offset((page - 1) * pageSize).Limit(pageSize).OrderBy("id")

	// 生成SQL
	sql, args, err := query.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DB.Query(context.Background(), sql, args...)
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
	query := sq.Select(
		"t.id",
		"t.uid",
		"COALESCE(tp.ancestor_id, 0) AS parent_id",
		"t.tenant_name",
		"t.tenant_type",
		"t.info",
		"t.configuration",
		"t.create_time",
		"t.update_time",
		"tc.depth",
	).
		From("tenants t").
		Join("tenant_closure tc ON t.id = tc.descendant_id").
		LeftJoin("tenant_closure tp ON tp.descendant_id = t.id AND tp.depth = 1").
		Where(sq.Eq{"tc.ancestor_id": ancestorID}).
		PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType()))

	// 添加分页和排序
	query = query.Offset((page - 1) * pageSize).Limit(pageSize).OrderBy("t.id")

	// 生成SQL
	sql, args, err := query.ToSql()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to build SQL: %v", err)
		return nil, err
	}
	common.Logger.Sugar().Debugf("%v %v", sql, args)

	rows, err := common.DB.Query(context.Background(), sql, args...)
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
	query := sq.Select(
		"t.id",
		"t.uid",
		"COALESCE(tp.ancestor_id, 0) AS parent_id",
		"t.tenant_name",
		"t.tenant_type",
		"t.info",
		"t.configuration",
		"t.create_time",
		"t.update_time",
	).
		From("tenants t").
		LeftJoin("tenant_closure tp ON tp.descendant_id = t.id AND tp.depth = 1").
		PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType()))

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

	rows, err := common.DB.Query(context.Background(), sql, args...)
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
	common.Logger.Debug("TenantUpdateConfiguration %v", zap.Any("tenant", m))

	commandTag, err := common.DB.Exec(context.Background(), "UPDATE tenants SET configuration = $1, update_time = NOW() WHERE (id = $2) AND (update_time = $3)", m.Configuration, m.ID, m.UpdateTime)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to update tenant configuration: %v", err)
		return err
	}

	// 检查是否有行被更新
	rowsAffected, err := commandTag.RowsAffected()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to get rows affected: %v", err)
		return err
	}
	if rowsAffected == 0 {
		// 检查租户是否存在
		var exists bool
		err = common.DB.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", m.ID).Scan(&exists)
		if err != nil {
			common.Logger.Sugar().Errorf("Failed to check tenant existence: %v", err)
			return err
		}

		if !exists {
			return common.ErrTenantNotFound
		}

		// 租户存在但更新时间不匹配
		return common.ErrModify
	}

	return nil
}

func TenantUpdateBase(m *protos.Tenant) error {
	common.Logger.Debug("TenantUpdateBase %v", zap.Any("tenant", m))
	commandTag, err := common.DB.Exec(
		context.Background(),
		"UPDATE tenants SET tenant_name = $1, tenant_type = $2, info = $3, update_time = NOW() WHERE id = $4",
		m.TenantName,
		m.TenantType,
		m.Info,
		m.ID,
	)
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to update tenant base: %v", err)
		return err
	}
	rowsAffected, err := commandTag.RowsAffected()
	if err != nil {
		common.Logger.Sugar().Errorf("Failed to get rows affected: %v", err)
		return err
	}
	if rowsAffected == 0 {
		return common.ErrTenantNotFound
	}
	return nil
}

func UserQueryByTenant(tenantID, page, pageSize uint64, nickname string, uids []uint64) (rr []protos.User, e error) {
	act := sq.Select("uid", "tenant_id", "cellphone", "email", "nickname", "avatar_url", "gender", "addr", "ext", "create_time").From("users").PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).Where(sq.Eq{"tenant_id": tenantID})
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

	rows, err := common.DB.Query(context.Background(), sql, args...)
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
	act := sq.Select("count(uid) as count").From("users").PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).Where(sq.Eq{"tenant_id": tenantID})
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

	rows, err := common.DB.Query(context.Background(), sql, args...)
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

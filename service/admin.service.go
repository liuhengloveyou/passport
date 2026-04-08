package service

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/dao"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
)

/*
rootTenant操作，同时添加用户和租户，
*/
func AdminTenantNew(sess *protos.User, m *protos.NewTenantReq) (uid, tenantID uint64, e error) {
	defer func() {
		evictTenantCache(tenantID)
	}()

	if sess == nil {
		return 0, 0, common.ErrService
	}
	// 只有root租户的超级管理员登录，才能通过该接口添加租户和管理员
	if sess.TenantID != common.ServConfig.RootTenantID {
		return 0, 0, common.ErrNoAuth
	}

	if m.TenantName == "" {
		return 0, 0, common.ErrTenantNameNull
	}
	if m.TenantType == "" {
		return 0, 0, common.ErrTenantTypeNull
	}
	if m.Cellphone == "" && m.Nickname == "" {
		return 0, 0, common.ErrUserNmae
	}
	if m.Password == "" {
		return 0, 0, common.ErrTenantAdminPasswordNull
	}

	ancestorID := sess.TenantID
	if m.ParentID > 0 {
		depth, err := dao.TenantClosureIsDescendant(sess.TenantID, m.ParentID)
		if err != nil {
			common.Logger.Sugar().Errorf("AdminTenantNew check parent descendant ERR: %v\n", err)
			return 0, 0, common.ErrService
		}
		if depth < 0 {
			return 0, 0, common.ErrNoAuth
		}
		ancestorID = m.ParentID
	}

	// 开始事务
	ctx := context.Background()
	tx, e := common.DB.Begin(ctx)
	if e != nil {
		common.Logger.Sugar().Errorf("AdminTenantNew Begin transaction ERR: %v\n", e)
		return 0, 0, common.ErrService
	}
	defer func() {
		if e != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	// 创建管理员用户
	adminUser := &protos.UserReq{
		TenantID:  0,
		Cellphone: m.Cellphone,
		Nickname:  m.Nickname,
		Password:  m.Password,
		Roles:     []string{"root"},
	}

	// 在事务中创建管理员用户
	var adminUID uint64
	adminUID, e = addUserServiceWithTx(tx, adminUser)
	if e != nil {
		common.Logger.Sugar().Errorf("AdminTenantNew addUserServiceWithTx ERR: %v\n", e)
		return 0, 0, e
	}

	if adminUID <= 0 {
		common.Logger.Sugar().Errorf("AdminTenantNew addUserServiceWithTx invalid UID: %v\n", adminUID)
		return 0, 0, common.ErrService
	}

	// 创建租户
	tenant := &protos.Tenant{
		UID:           adminUID,
		ParentID:      ancestorID,
		TenantName:    m.TenantName,
		TenantType:    m.TenantType,
		Info:          m.Info,
		Configuration: m.Configuration,
	}
	if tenant.Info == nil {
		tenant.Info = &protos.TenantInfo{
			AdminCellphone: m.Cellphone,
		}
	} else if tenant.Info.AdminCellphone == "" && m.Cellphone != "" {
		tenant.Info.AdminCellphone = m.Cellphone
	}

	if tenant.Configuration == nil {
		tenant.Configuration = &protos.TenantConfiguration{}
	}
	if len(tenant.Configuration.Roles) == 0 {
		tenant.Configuration.Roles = []protos.RoleStruct{{
			RoleTitle: "超级管理员",
			RoleValue: "root",
		}}
	}
	// 在事务中创建租户
	tenantID, e = dao.TenantInsert(tx, tenant)
	if e != nil {
		common.Logger.Sugar().Errorf("AdminTenantNew TenantInsert ERR: %v\n", e)
		return 0, 0, common.ErrService
	}

	if tenantID <= 0 {
		common.Logger.Sugar().Errorf("AdminTenantNew TenantInsert invalid tenantID: %v\n", tenantID)
		return 0, 0, common.ErrService
	}

	// 在事务中插入租户闭包表记录
	if e = dao.TenantClosureInsert(tx, ancestorID, tenantID); e != nil {
		common.Logger.Sugar().Errorf("AdminTenantNew insertTenantClosure ERR: %v\n", e)
		return 0, 0, common.ErrService
	}

	// 设置用户为超级管理员角色
	if e = accessctl.AddRoleForUserInDomain(adminUID, tenantID, "root"); e != nil {
		common.Logger.Sugar().Errorf("AdminTenantNew AddRoleForUserInDomain ERR: %v\n", e)
		return 0, 0, common.ErrService
	}

	return adminUID, tenantID, nil
}

func AdminTenantList(page, pageSize uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	var rr []protos.Tenant
	rr, e = dao.TenantList(page, pageSize)
	if e != nil {
		common.Logger.Sugar().Error("AdminTenantList db ERR: ", e)
		e = common.ErrService
		return
	}
	if len(rr) == 0 {
		e = common.ErrNull
		return
	}
	rst.List = rr

	if hasTotal {
		rst.Total, e = dao.TenantCount()
		if e != nil {
			common.Logger.Sugar().Error("AdminTenantList db ERR: ", e)
			e = common.ErrService
			return
		}
	}

	return
}

// addUserServiceWithTx 在事务中创建用户
func addUserServiceWithTx(tx database.Tx, p *protos.UserReq) (uid uint64, e error) {
	if p.Cellphone == "" && p.Email == "" && p.Nickname == "" {
		return 0, common.ErrUserNmae
	}
	if p.Password == "" {
		return 0, common.ErrPWDNil
	}

	// 预处理用户数据
	if e = userPreTreat(p); e != nil {
		common.Logger.Sugar().Errorf("addUserServiceWithTx userPreTreat ERR: %v\n", e)
		return 0, common.ErrParam
	}

	if err := userCheckDuplicates(p); err != nil {
		return 0, err
	}

	p.Password = common.EncryPWD(p.Password)

	// 调用dao层在事务中插入用户
	userUID, err := dao.UserInsert(p, tx)
	if err != nil {
		common.Logger.Sugar().Errorf("addUserServiceWithTx UserInsert ERR: %v\n", err)
		return 0, common.ErrService
	}

	return uint64(userUID), nil
}

// 只有root租户的超级管理员登录，才能通过该接口设置租户的父租户
func AdminTenantSetParent(sessUser *protos.User, descendantId, ancestorId uint64) error {
	if common.ServConfig.RootTenantID <= 0 || sessUser.TenantID != common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("AdminSetParent param ERR: ", sessUser.TenantID)
		return common.ErrNoAuth
	}

	// 参数校验
	if descendantId <= 0 || ancestorId <= 0 {
		common.Logger.Sugar().Error("TenantSetParent param ERR: ", ancestorId, descendantId)
		return common.ErrParam
	}
	if descendantId == ancestorId {
		common.Logger.Sugar().Error("TenantSetParent param same ERR: ", descendantId, ancestorId)
		return common.ErrTenantSame
	}

	if descendantId == common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("TenantSetParent param tenantID ERR: ", descendantId)
		return common.ErrTenantRoot
	}

	// 删除缓存
	evictTenantCache(descendantId, ancestorId)

	// 检查tenant是否存在
	tenant, err := dao.TenantGetByID(descendantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantSetParent db ERR: %v\n", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	parentTenant, err := dao.TenantGetByID(ancestorId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantSetParent parent db ERR: %v\n", err)
		return common.ErrService
	}
	if nil == parentTenant {
		return common.ErrTenantNotFound
	}

	// 检查当前的父子关系
	if depth, err := dao.TenantClosureIsDescendant(ancestorId, descendantId); err != nil {
		return common.ErrService
	} else if depth == 1 {
		// 已经是直接子节点，无需操作
		return nil
	} else if depth > 1 {
		// 是间接后代，需要移动到直接子节点位置
		common.Logger.Sugar().Infof("Moving tenant %d from depth %d to direct child of %d", descendantId, depth, ancestorId)
	}

	// 开始事务
	tx, err := common.DBPool.BeginTx(context.Background(), pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		common.Logger.Sugar().Errorf("AdminTenantSetParent begin transaction ERR: %v\n", err)
		return common.ErrService
	}
	defer func() {
		if err != nil {
			tx.Rollback(context.Background())
		} else {
			tx.Commit(context.Background())

		}
	}()

	// 检查是否会形成循环引用（新父租户不能是当前租户的子孙节点）
	if ancestorId != common.ServConfig.RootTenantID {
		var count int
		err = tx.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id = $2",
			ancestorId, descendantId).Scan(&count)
		if err != nil {
			common.Logger.Sugar().Errorf("AdminTenantSetParent check cycle ERR: %v\n", err)
			return common.ErrService
		}
		if count > 0 {
			common.Logger.Sugar().Errorf("AdminTenantSetParent cycle detected: ancestor %d is descendant of %d", ancestorId, descendantId)
			return common.ErrTenantCircularRef
		}
	}

	// 更新整个子树的层级关系 - 使用v2标准算法
	if err = dao.TenantClosureUpdateSubtreeV2Safe(tx, descendantId, ancestorId); err != nil {
		common.Logger.Sugar().Errorf("AdminTenantSetParent update subtree v2 ERR: %v", err)
		return err
	}

	// 插入操作日志
	common.Logger.Sugar().Warnf("AdminTenantSetParent: user %d set tenant %d parent to %d", sessUser.UID, descendantId, ancestorId)

	return nil
}

// AdminTenantUpdateConfig 更新租户配置
func AdminTenantUpdateConfig(sessUser *protos.User, req *protos.UpdateTenantConfigReq) error {
	defer evictTenantCache(req.TenantID)

	// 权限检查：只有root租户的超级管理员或者租户自己的管理员才能更新配置
	if sessUser.TenantID != common.ServConfig.RootTenantID && sessUser.TenantID != req.TenantID {
		common.Logger.Sugar().Error("AdminTenantUpdateConfig auth ERR: ", sessUser.TenantID, req.TenantID)
		return common.ErrNoAuth
	}

	// 参数校验
	if req.TenantID <= 0 || req.Configuration == nil || req.LastUpdateTime == "" {
		common.Logger.Sugar().Error("AdminTenantUpdateConfig param ERR: ", req.TenantID)
		return common.ErrParam
	}

	updateTime, err := time.Parse(time.RFC3339, req.LastUpdateTime)
	if err != nil {
		common.Logger.Sugar().Errorf("解析更新时间失败: %v", err)
		return common.ErrParam
	}

	// 检查tenant是否存在
	tenant, err := dao.TenantGetByID(req.TenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("AdminTenantUpdateConfig db ERR: %v\n", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	// 更新租户配置
	tenant.Configuration = req.Configuration
	tenant.UpdateTime = &updateTime
	if err := dao.TenantUpdateConfiguration(tenant); err != nil {
		common.Logger.Sugar().Errorf("AdminTenantUpdateConfig update ERR: %v\n", err)
		// 根据错误类型返回不同的错误信息
		switch err {
		case common.ErrTenantNotFound:
			return common.ErrTenantNotFound
		case common.ErrModify:
			// 更新时间不匹配
			return common.ErrModify
		default:
			return common.ErrService
		}
	}

	// 记录操作日志
	common.Logger.Sugar().Infof("AdminTenantUpdateConfig: user %d updated tenant %d configuration", sessUser.UID, req.TenantID)

	return nil
}

// AdminTenantDelete 删除指定租户本身，并将其直接子租户挂到当前会话租户下。
func AdminTenantDelete(sessUser *protos.User, tenantID uint64) error {
	if sessUser == nil || sessUser.UID <= 0 || sessUser.TenantID <= 0 || tenantID <= 0 {
		common.Logger.Sugar().Error("AdminTenantDelete param ERR: ", sessUser.UID, sessUser.TenantID, tenantID)
		return common.ErrParam
	}
	if common.ServConfig.RootTenantID <= 0 || sessUser.TenantID != common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("AdminTenantDelete auth ERR: ", sessUser.UID, sessUser.TenantID)
		return common.ErrNoAuth
	}
	if tenantID == common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("AdminTenantDelete root tenant ERR: ", tenantID)
		return common.ErrNoAuth
	}

	// 先检查租户是否存在（通过 service 层接口）
	if _, err := TenantGetByIDService(tenantID); err != nil {
		common.Logger.Sugar().Errorf("AdminTenantDelete TenantGetByIDService ERR: %v\n", err)
		return err
	}

	// 查询该租户的后代节点，用于识别直接子租户（depth=1）。
	subtree, err := collectTenantSubtree(tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("AdminTenantDelete collectTenantSubtree ERR: %v\n", err)
		return common.ErrService
	}

	// 把目标租户的直接子租户重新挂到当前会话租户下。
	for _, child := range subtree {
		if child.Depth != 1 || child.ID == tenantID {
			continue
		}
		if err := AdminTenantSetParent(sessUser, child.ID, sessUser.TenantID); err != nil {
			common.Logger.Sugar().Errorf("AdminTenantDelete reparent child ERR: child=%d parent=%d err=%v\n", child.ID, sessUser.TenantID, err)
			return common.ErrService
		}
	}

	// 仅删除目标租户自身数据。
	if e := deleteUsersByTenant(tenantID); e != nil {
		common.Logger.Sugar().Errorf("AdminTenantDelete deleteUsersByTenant ERR: %v\n", e)
		return common.ErrService
	}
	if e := deleteTenantRelatedData(tenantID); e != nil {
		common.Logger.Sugar().Errorf("AdminTenantDelete deleteTenantRelatedData ERR: %v\n", e)
		return common.ErrService
	}
	if e := deleteTenantRecord(tenantID); e != nil {
		common.Logger.Sugar().Errorf("AdminTenantDelete deleteTenantRecord ERR: %v\n", e)
		return common.ErrService
	}
	evictTenantCache(tenantID)

	return nil
}

func collectTenantSubtree(ancestorID uint64) ([]protos.Tenant, error) {
	out := make([]protos.Tenant, 0, 16)
	page := uint64(1)
	pageSize := uint64(500)

	for {
		list, err := dao.TenantListByAncestorID(ancestorID, page, pageSize)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			break
		}
		out = append(out, list...)
		if uint64(len(list)) < pageSize {
			break
		}
		page++
	}

	return out, nil
}

func deleteUsersByTenant(tenantID uint64) error {
	page := uint64(1)
	pageSize := uint64(500)

	for {
		users, err := dao.UserQueryByTenant(tenantID, page, pageSize, "", nil)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			break
		}
		for _, user := range users {
			if _, err = TenantUserDel(user.UID, tenantID); err != nil {
				return err
			}
		}
		if uint64(len(users)) < pageSize {
			break
		}
	}

	return nil
}

func deleteTenantRelatedData(tenantID uint64) error {
	placeholder := database.GetPlaceholderFormat(common.DB.DriverType())
	ctx := context.Background()

	delDepSQL, delDepArgs, err := sq.Delete("departments").
		Where(sq.Eq{"tenant_id": tenantID}).
		PlaceholderFormat(placeholder).
		ToSql()
	if err != nil {
		return err
	}
	if _, err = common.DB.Exec(ctx, delDepSQL, delDepArgs...); err != nil {
		return err
	}

	delPermissionSQL, delPermissionArgs, err := sq.Delete("permission").
		Where(sq.Eq{"tenant_id": tenantID}).
		PlaceholderFormat(placeholder).
		ToSql()
	if err != nil {
		return err
	}
	if _, err = common.DB.Exec(ctx, delPermissionSQL, delPermissionArgs...); err != nil {
		return err
	}

	return nil
}

func deleteTenantRecord(tenantID uint64) error {
	placeholder := database.GetPlaceholderFormat(common.DB.DriverType())
	delSQL, delArgs, err := sq.Delete("tenants").
		Where(sq.Eq{"id": tenantID}).
		PlaceholderFormat(placeholder).
		ToSql()
	if err != nil {
		return err
	}
	_, err = common.DB.Exec(context.Background(), delSQL, delArgs...)
	return err
}

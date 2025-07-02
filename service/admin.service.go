package service

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/cache"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

/*
rootTenant操作，同时添加用户和租户，
*/
func AdminTenantNew(sess *protos.User, m *protos.NewTenantReq) (uid, tenantID uint64, e error) {
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
	if m.Cellphone == "" {
		return 0, 0, common.ErrTenantAdminCellphoneNull
	}
	if m.Password == "" {
		return 0, 0, common.ErrTenantAdminPasswordNull
	}

	// 开始事务
	ctx := context.Background()
	tx, e := common.DBPool.Begin(ctx)
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
		UID:        adminUID,
		ParentID:   sess.TenantID,
		TenantName: m.TenantName,
		TenantType: m.TenantType,
		Info: &protos.TenantInfo{
			AdminCellphone: m.Cellphone,
		},
		Configuration: &protos.TenantConfiguration{
			Roles: []protos.RoleStruct{{
				RoleTitle: "超级管理员",
				RoleValue: "root",
			}},
		},
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
	if e = dao.TenantClosureInsert(tx, sess.TenantID, tenantID); e != nil {
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

// addUserServiceWithTx 在事务中创建用户
func addUserServiceWithTx(tx pgx.Tx, p *protos.UserReq) (uid uint64, e error) {
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
	userUID, err := dao.UserInsert(p, &tx)
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

	if descendantId == common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("TenantSetParent param tenantID ERR: ", descendantId)
		return common.ErrTenantRoot
	}

	// 删除缓存
	cache.DelTenantCache(descendantId)
	cache.DelTenantCache(ancestorId)

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
			return common.ErrParam
		}
	}

	// 更新整个子树的层级关系
	if err = dao.TenantClosureUpdateSubtree(tx, descendantId, ancestorId); err != nil {
		common.Logger.Sugar().Errorf("AdminTenantSetParent update subtree ERR: %v\n", err)
		return err
	}

	// 更新tenant的parentID
	if err = dao.TenantUpdateParentID(descendantId, ancestorId, &tx); err != nil {
		common.Logger.Sugar().Errorf("AdminTenantSetParent update ERR: %v\n", err)
		return common.ErrService
	}

	// 插入操作日志
	common.Logger.Sugar().Warnf("AdminTenantSetParent: user %d set tenant %d parent to %d", sessUser.UID, descendantId, ancestorId)

	return nil
}

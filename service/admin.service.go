package service

import (
	"github.com/liuhengloveyou/passport/cache"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

/*
rootTenant操作，同时添加用户和租户，
*/
func TenantNew(sess *protos.User, m *protos.NewTenantReq) (uid, tenantID uint64, e error) {
	if sess == nil {
		return 0, 0, common.ErrService
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

	// 创建管理员用户
	adminUser := &protos.UserReq{
		TenantID:  0,
		Cellphone: m.Cellphone,
		Password:  m.Password,
		Roles:     []string{"root"},
	}

	// 创建管理员用户
	adminUID, e := AddUserService(adminUser)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantAdd AddUserService ERR: %v %v\n", tenantID, e)
		return 0, 0, e
	}

	if adminUID <= 0 {
		common.Logger.Sugar().Errorf("TenantAdd AddUserService ERR: %v %v\n", tenantID, e)
		return 0, 0, e
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

	// 创建租户
	tenantID, e = TenantAdd(tenant)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantAdd TenantAdd ERR: %v %v\n", tenantID, e)
		return 0, 0, e
	}

	return adminUID, tenantID, nil
}

func AdminTenantList(parentID, page, pageSize uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	var rr []protos.Tenant
	rr, e = dao.TenantList(parentID, page, pageSize)
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

// AdminTenantQuery 根据条件查询租户列表
func AdminTenantQuery(tenantName, cellphone string, page, pageSize uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	var rr []protos.Tenant
	rr, e = dao.TenantQuery(tenantName, cellphone, page, pageSize)
	if e != nil {
		common.Logger.Sugar().Error("AdminTenantQuery db ERR: ", e)
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
			common.Logger.Sugar().Error("AdminTenantQuery db ERR: ", e)
			e = common.ErrService
			return
		}
	}

	return
}

// AdminTenantQuery 根据条件查询租户列表
func AdminTenantTake(tenantId uint64) (rst *protos.Tenant, e error) {
	// 先查缓存
	rst = cache.GetTenantCache(tenantId)
	if rst != nil {
		return
	}

	rst, e = dao.TenantGetByID(tenantId)
	if e != nil {
		common.Logger.Sugar().Error("AdminTenantTake ERR: ", e)
		e = common.ErrService
		return
	}

	// 缓存
	if rst != nil {
		cache.SetTenantCache(rst)
	}

	return
}

func AdminTenantSetParent(tenantID, parentID uint64) error {
	cache.DelTenantCache(tenantID)

	// 参数校验
	if tenantID <= 0 {
		common.Logger.Sugar().Error("TenantSetParent param tenantID ERR: ", tenantID)
		return common.ErrParam
	}

	if tenantID == common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("TenantSetParent param tenantID ERR: ", tenantID)
		return common.ErrParam
	}

	// 检查tenant是否存在
	tenant, err := dao.TenantGetByID(tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantSetParent db ERR: %v\n", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	// 如果parentID不为0，检查parent tenant是否存在
	if parentID > 0 {
		parentTenant, err := dao.TenantGetByID(parentID)
		if err != nil {
			common.Logger.Sugar().Errorf("TenantSetParent parent db ERR: %v\n", err)
			return common.ErrService
		}
		if nil == parentTenant {
			return common.ErrTenantNotFound
		}
	}

	// 更新tenant的parentID
	if err := dao.TenantUpdateParentID(tenantID, parentID); err != nil {
		common.Logger.Sugar().Errorf("TenantSetParent update ERR: %v\n", err)
		return common.ErrService
	}

	return nil
}

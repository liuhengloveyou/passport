package service

import (
	"github.com/liuhengloveyou/passport/cache"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func TenantTreeList(sessionUser *protos.User, ancestorID, page, pageSize uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	// 只能查询自己的子层级
	if sessionUser.TenantID != common.ServConfig.RootTenantID {

		isDescendant, err := dao.TenantClosureIsDescendant(sessionUser.TenantID, ancestorID)
		if err != nil {
			common.Logger.Sugar().Error("TenantTreeList check descendant ERR: ", err)
			e = common.ErrService
			return
		}
		if !isDescendant {
			common.Logger.Sugar().Warnf("TenantTreeList access denied: tenant %d cannot access parent %d", sessionUser.TenantID, ancestorID)
			e = common.ErrNoAuth
			return
		}
	}

	var rr []protos.Tenant
	rr, e = dao.TenantListByAncestorID(ancestorID, page, pageSize)
	if e != nil {
		common.Logger.Sugar().Error("TenantTreeList descendants db ERR: ", e)
		e = common.ErrService
		return
	}

	if len(rr) == 0 {
		e = common.ErrNull
		return
	}
	rst.List = rr

	if hasTotal {
		if sessionUser.TenantID == common.ServConfig.RootTenantID {
			rst.Total, e = dao.TenantCount()
		} else {
			rst.Total, e = dao.TenantCountByAncestorID(ancestorID)
		}
		if e != nil {
			common.Logger.Sugar().Error("TenantTreeList count db ERR: ", e)
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

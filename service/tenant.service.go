package service

import (
	"context"

	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/cache"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/dao"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
)

// invalidateTenantCache 统一失效指定租户缓存（忽略无效ID）。
func evictTenantCache(tenantIDs ...uint64) {
	for _, id := range tenantIDs {
		if id > 0 {
			cache.DelTenantCache(id)
		}
	}
}

// getTenantByIDCached 先查内存缓存，未命中再查数据库并回填缓存。
func getTenantByIDCached(tenantID uint64) (*protos.Tenant, error) {
	if tenantID <= 0 {
		return nil, common.ErrParam
	}
	if tenant := cache.GetTenantCache(tenantID); tenant != nil {
		return tenant, nil
	}
	tenant, err := dao.TenantGetByID(tenantID)
	if err != nil {
		return nil, err
	}
	if tenant != nil {
		cache.SetTenantCache(tenant)
	}
	return tenant, nil
}

// TenantGetByIDService 按租户ID获取租户信息。
func TenantGetByIDService(tenantID uint64) (*protos.Tenant, error) {
	if tenantID <= 0 {
		return nil, common.ErrParam
	}
	tenant, err := getTenantByIDCached(tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantGetByIDService db ERR: %v\n", err)
		return nil, common.ErrService
	}
	if tenant == nil {
		return nil, common.ErrTenantNotFound
	}
	return tenant, nil
}

// TenantAdd 创建租户并将创建者设为该租户超级管理员。
func TenantAdd(m *protos.Tenant) (tenantID uint64, e error) {
	defer func() {
		evictTenantCache(tenantID)
	}()

	if m.UID <= 0 {
		return 0, common.ErrTenantAddERR
	}

	if m.TenantName == "" {
		return 0, common.ErrTenantNameNull
	}
	if m.TenantType == "" {
		return 0, common.ErrTenantTypeNull
	}

	if m.Configuration == nil {
		m.Configuration = &protos.TenantConfiguration{}
	}
	if len(m.Configuration.More) > 100 {
		return 0, common.ErrService
	}

	// 默认添加超级管理员角色
	m.Configuration.Roles = []protos.RoleStruct{{
		RoleTitle: "超级管理员",
		RoleValue: "root",
	}}

	ctx := context.Background()
	var tx database.Tx
	tx, e = common.DB.Begin(ctx)
	defer func() {
		if e != nil {
			tx.Rollback(ctx)
		}
	}()

	tenantID, e = dao.TenantInsert(tx, m)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantAdd TenantInsert ERR: %v %v %v\n", m.UID, tenantID, e)
		e = common.ErrService
		return
	}
	if tenantID <= 0 {
		common.Logger.Sugar().Errorf("TenantAdd AddRoleForUserInDomain ERR: %v %v %v\n", m.UID, tenantID, e)
		return 0, common.ErrTenantAddERR
	}

	// 当前用户设置为超级管理员角色
	if e = accessctl.AddRoleForUserInDomain(m.UID, uint64(tenantID), "root"); e != nil {
		common.Logger.Sugar().Errorf("TenantAdd AddRoleForUserInDomain ERR: %v %v %v\n", m.UID, tenantID, e)
		return 0, common.ErrService
	}

	e = tx.Commit(ctx)

	return
}

// TenantAddRole 为指定租户新增角色定义。
func TenantAddRole(tenantId uint64, role protos.RoleStruct) error {
	defer evictTenantCache(tenantId)

	tenant, err := getTenantByIDCached(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantAddRole db ERR: %v\n", err)
		return common.ErrService
	}
	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	if len(tenant.Configuration.Roles) > 100 {
		common.Logger.Sugar().Errorf("TenantAddRole Configuration.Roles to long: %v\n", len(tenant.Configuration.Roles))
		return common.ErrService
	}
	for i := 0; i < len(tenant.Configuration.Roles); i++ {
		if tenant.Configuration.Roles[i].RoleTitle == role.RoleTitle || tenant.Configuration.Roles[i].RoleValue == role.RoleValue {
			common.Logger.Sugar().Errorf("TenantAddRole dup: %v %v\n", role, tenant.Configuration.Roles[i])
			return common.ErrPgDupKey
		}
	}

	tenant.Configuration.Roles = append(tenant.Configuration.Roles, role)

	return dao.TenantUpdateConfiguration(tenant)
}

// TenantDelRole 删除指定租户的角色定义（不允许删除 root 角色）。
func TenantDelRole(tenantId uint64, role protos.RoleStruct) error {
	defer evictTenantCache(tenantId)

	common.Logger.Sugar().Debugf("TenantDelRole: %v\n", role)
	if role.RoleValue == "root" {
		common.Logger.Sugar().Errorf("TenantDelRole root\n")
		return common.ErrService
	}
	tenant, err := getTenantByIDCached(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantDelRole db ERR: %v\n", err)
		return common.ErrService
	}
	common.Logger.Sugar().Infof("tenant: %v\n", tenant)
	if nil == tenant {
		return common.ErrTenantNotFound
	}
	common.Logger.Sugar().Infof("tenant: %v %v\n", role, tenant.Configuration.Roles)

	i := 0
	for ; i < len(tenant.Configuration.Roles); i++ {
		if tenant.Configuration.Roles[i].RoleValue == role.RoleValue {
			break
		}
	}

	if i < len(tenant.Configuration.Roles) {
		tenant.Configuration.Roles = append(tenant.Configuration.Roles[:i], tenant.Configuration.Roles[i+1:]...)
		return dao.TenantUpdateConfiguration(tenant)
	}

	return nil
}

// TenantGetRole 获取指定租户的角色列表。
func TenantGetRole(tenantId uint64) (roles []protos.RoleStruct) {
	tenant, err := getTenantByIDCached(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantAddRole db ERR: %v\n", err)
		return
	}
	if nil == tenant {
		return
	}

	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)

	return tenant.Configuration.Roles
}

// TenantLoadConfiguration 读取租户配置，key 为空时返回全部配置。
func TenantLoadConfiguration(tenantId uint64, key string) (interface{}, error) {
	tenant, err := getTenantByIDCached(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantLoadConfiguration db ERR: %v\n", err)
		return nil, common.ErrService
	}
	if nil == tenant {
		common.Logger.Sugar().Errorf("TenantLoadConfiguration nil: %v\n", tenantId)
		return nil, common.ErrTenantNotFound
	}

	if key != "" && tenant.Configuration.More != nil {
		return tenant.Configuration.More[key], nil
	}

	return tenant.Configuration.More, nil
}

// TenantUpdateConfiguration 增量更新租户配置键值。
func TenantUpdateConfiguration(tenantId uint64, data map[string]interface{}) error {
	defer evictTenantCache(tenantId)

	if len(data) <= 0 || len(data) > 100 {
		common.Logger.Sugar().Error("UpdateTenantConfiguration param len ERR: ", len(data))
		return common.ErrParam
	}
	for k, _ := range data {
		if len(k) > 64 {
			common.Logger.Sugar().Error("UpdateTenantConfiguration param k len")
			return common.ErrParam
		}
	}

	tenant, err := getTenantByIDCached(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantUpdateConfiguration db ERR: %v\n", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	if tenant.Configuration == nil {
		tenant.Configuration = &protos.TenantConfiguration{}
	}
	if tenant.Configuration.More == nil {
		tenant.Configuration.More = make(protos.MapStruct, 1)
	}

	for k, v := range data {
		if v != nil {
			if len(tenant.Configuration.More) > 100 {
				common.Logger.Sugar().Errorf("tenant.Configuration.More too len: %d\n", len(tenant.Configuration.More))
				return common.ErrService
			}
			tenant.Configuration.More[k] = v
		} else {
			delete(tenant.Configuration.More, k)
		}
	}

	return dao.TenantUpdateConfiguration(tenant)
}

// getTenantUserRoles 获取租户内用户角色并补全角色标题信息。
func getTenantUserRoles(uid, tenantId uint64) (roles []protos.RoleStruct, err error) {
	tenant, err := getTenantByIDCached(tenantId) // 取tenant里的角色字典
	if err != nil {
		common.Logger.Sugar().Errorf("getTenantUserRole db ERR: %v\n", err)
		return nil, common.ErrService
	}
	if nil == tenant {
		common.Logger.Sugar().Errorf("getTenantUserRole db nil\n")
		return nil, common.ErrTenantNotFound
	}

	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)

	roleVals := accessctl.GetRoleForUserInDomain(uid, tenantId)
	if len(roleVals) == 0 {
		common.Logger.Sugar().Errorf("getTenantUserRole roles nil\n")
		return nil, nil
	}

	roles = make([]protos.RoleStruct, 0)
	for i := 0; i < len(roleVals); i++ {
		var roleOne *protos.RoleStruct = nil
		for j := 0; j < len(tenant.Configuration.Roles); j++ {
			if tenant.Configuration.Roles[j].RoleValue == roleVals[i] {
				roleOne = &protos.RoleStruct{
					RoleTitle: tenant.Configuration.Roles[j].RoleTitle,
					RoleValue: roleVals[i],
				}
			}
		}

		if roleOne != nil {
			roles = append(roles, *roleOne)
		} else {
			// 如果角色字典已经删除
			common.Logger.Sugar().Warnf("getTenantUserRoles DeleteRoleForUserInDomain: %v %v %v\n", uid, tenantId, roleVals[i])
			accessctl.DeleteRoleForUserInDomain(uid, tenantId, roleVals[i])
		}
	}

	return roles, nil
}

// getTenantUserDepartment 根据部门ID列表组装用户部门信息。
func getTenantUserDepartment(uid, tenantId uint64, depIds []uint64) (deps []protos.Department, e error) {
	departments, err := DepartmentFind(0, tenantId, 0, 0)
	if err != nil {
		common.Logger.Sugar().Error("getTenantUserDepartment DepartmentFind ERR: %v", e)
		e = common.ErrService
		return
	}
	if nil == departments {
		common.Logger.Sugar().Errorf("DepartmentFind db nil\n")
		return nil, common.ErrNull
	}

	for _, dep := range depIds {
		for j := 0; j < len(departments); j++ {
			if dep == departments[j].Id {
				deps = append(deps, departments[j])
				break
			}
		}
	}

	return
}

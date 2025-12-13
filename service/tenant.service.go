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

func TenantAdd(m *protos.Tenant) (tenantID uint64, e error) {
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
	var tx pgx.Tx
	tx, e = common.DBPool.Begin(ctx)
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

	cache.DelTenantCache(tenantID)

	return
}

func TenantAddRole(tenantId uint64, role protos.RoleStruct) error {
	cache.DelTenantCache(tenantId)

	tenant, err := dao.TenantGetByID(tenantId)
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

func TenantDelRole(tenantId uint64, role protos.RoleStruct) error {
	cache.DelTenantCache(tenantId)

	common.Logger.Sugar().Debugf("TenantDelRole: %v\n", role)
	if role.RoleValue == "root" {
		common.Logger.Sugar().Errorf("TenantDelRole root\n")
		return common.ErrService
	}
	tenant, err := dao.TenantGetByID(tenantId)
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

func TenantGetRole(tenantId uint64) (roles []protos.RoleStruct) {
	tenant, err := dao.TenantGetByID(tenantId)
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

func TenantLoadConfiguration(tenantId uint64, key string) (interface{}, error) {
	tenant, err := dao.TenantGetByID(tenantId)
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

func TenantUpdateConfiguration(tenantId uint64, data map[string]interface{}) error {
	cache.DelTenantCache(tenantId)

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

	tenant, err := dao.TenantGetByID(tenantId)
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

func getTenantUserRoles(uid, tenantId uint64) (roles []protos.RoleStruct, err error) {
	tenant, err := dao.TenantGetByID(tenantId) // 取tenant里的角色字典
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
	if roleVals == nil || len(roleVals) == 0 {
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

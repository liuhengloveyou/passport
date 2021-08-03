package service

import (
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func TenantAdd(m *protos.Tenant) (tenantID int64, e error) {
	if m.TenantName == "" {
		return -1, common.ErrTenantNameNull
	}
	if m.TenantType == "" {
		return -1, common.ErrTenantTypeNull
	}

	if m.Configuration == nil {
		m.Configuration = &protos.TenantConfiguration{}
	}
	if len(m.Configuration.More) > 100 {
		return -1, common.ErrService
	}

	// 默认添加超级管理员角色
	m.Configuration.Roles = []protos.RoleStruct{{
		RoleTitle: "超级管理员",
		RoleValue: "root",
	}}

	tx := common.DB.MustBegin()
	defer func() {
		if e != nil {
			tx.Rollback()
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
		return -1, common.ErrTenantAddERR
	}

	// 当前用户设置为超级管理员角色
	if e = accessctl.AddRoleForUserInDomain(m.UID, uint64(tenantID), "root"); e != nil {
		common.Logger.Sugar().Errorf("TenantAdd AddRoleForUserInDomain ERR: %v %v %v\n", m.UID, tenantID, e)
		return -1, common.ErrService
	}

	e = tx.Commit()

	return
}

func TenantList(page, pageSize uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	var rr []protos.Tenant
	rr, e = dao.TenantList(page, pageSize)
	if e != nil {
		common.Logger.Sugar().Error("TenantList db ERR: ", e)
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
			common.Logger.Sugar().Error("TenantList db ERR: ", e)
			e = common.ErrService
			return
		}
	}

	return
}

func TenantUserAdd(uid, currTenantID uint64, roles []string, disable int8) (e error) {
	row, e := dao.UserUpdateTenantID(uid, currTenantID, 0)
	if e != nil {
		common.Logger.Sugar().Error("TenantUserAdd db ERR: ", e)
		return common.ErrService
	}
	if row != 1 {
		common.Logger.Sugar().Error("TenantUserAdd UserUpdateTenantID ERR: ", row, e)
		return common.ErrService
	}

	for _, role := range roles {
		if e = accessctl.AddRoleForUserInDomain(uid, currTenantID, role); e != nil {
			common.Logger.Sugar().Errorf("TenantUserAdd AddRoleForUserInDomain ERR: %v\n", e)
			return common.ErrService
		}
	}

	if e = TenantUserDisabledService(uid, currTenantID, disable); e != nil {
		common.Logger.Sugar().Warnf("TenantUserAdd TenantUserDisabledService ERR: %v\n", e)
		e = nil
	}

	return
}

func TenantUserDel(uid, currTenantID uint64) (r int64, e error) {
	// 删除所有角色
	if e = accessctl.DeleteRolesForUserInDomain(uid, currTenantID); e != nil {
		common.Logger.Sugar().Errorf("TenantUserDel ERR: %v\n", e)
		return 0, common.ErrService
	}

	if r, e = dao.UserUpdateTenantID(uid, 0, currTenantID); e != nil {
		common.Logger.Sugar().Errorf("TenantUserDel ERR: %v\n", e)
		return 0, common.ErrService
	}

	return
}

func TenantUserDisabledService(uid, currTenantID uint64, disabled int8) (e error) {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUserDisabledService ERR: %d %v %v\n", uid, currTenantID, disabled)
		return common.ErrParam
	}

	return TenantUpdateUserExt(uid, currTenantID, "disabled", disabled)
}

func TenantUpdateUserExt(uid, currTenantID uint64, k string, v interface{}) error {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %d %v %v\n", uid, k, v)
		return common.ErrParam
	}
	if k == "" {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %d %v %v\n", uid, k, v)
		return common.ErrParam
	}

	userInfo, e := dao.UserSelectByID(uid)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantUserDisabledService db ERR: %v\n", e)
		return common.ErrService
	}

	if userInfo.TenantID != currTenantID {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt tenant ERR: %v %v %v\n", uid, currTenantID, userInfo)
		return common.ErrNoAuth
	}

	userInfo.Ext[k] = v
	if v == nil {
		common.Logger.Sugar().Warnf("TenantUpdateUserExt delete: %v", k)
		delete(userInfo.Ext, k)
	}

	rows, e := dao.UserUpdateExt(uid, userInfo.UpdateTime, &userInfo.Ext)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %v\n", e)
		return common.ErrService
	}
	if rows < 1 {
		common.Logger.Sugar().Warnf("TenantUpdateUserExt RowsAffected 0")
	}

	return nil
}

func TenantUserGet(tenantID, page, pageSize uint64, nickname string, uids []uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	var rr []protos.User
	rr, e = dao.UserSelectByTenant(tenantID, page, pageSize, nickname, uids)
	if e != nil {
		common.Logger.Sugar().Error("TenantUserGet db ERR: ", e)
		e = common.ErrService
		return
	}
	if len(rr) == 0 {
		e = common.ErrNull
		return
	}
	rst.List = rr

	for i := 0; i < len(rr); i++ {
		if rr[i].Roles, e = getTenantUserRoles(rr[i].UID, rr[i].TenantID); e != nil {
			common.Logger.Sugar().Warnf("TenantUserGet getTenantUserRole ERR: %v\n", e)
		}
	}

	if hasTotal {
		rst.Total, e = dao.UserCountByTenant(tenantID, nickname, uids)
		if e != nil {
			common.Logger.Sugar().Error("TenantUserGet db ERR: ", e)
			e = common.ErrService
			return
		}
	}

	return
}

func TenantAddRole(tenantId uint64, role protos.RoleStruct) error {
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
			return common.ErrMysql1062
		}
	}

	tenant.Configuration.Roles = append(tenant.Configuration.Roles, role)

	return dao.TenantUpdateConfiguration(tenant)
}

func TenantDelRole(tenantId uint64, role protos.RoleStruct) error {
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
	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	i := 0
	for ; i < len(tenant.Configuration.Roles); i++ {
		if tenant.Configuration.Roles[i].RoleValue == role.RoleValue {
			break
		}
	}
	tenant.Configuration.Roles = append(tenant.Configuration.Roles[:i], tenant.Configuration.Roles[i+1:]...)

	return dao.TenantUpdateConfiguration(tenant)
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
	tenant, err := dao.TenantGetByID(tenantId)
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

	roles = make([]protos.RoleStruct, len(roleVals))
	for i := 0; i < len(roleVals); i++ {
		roles[i].RoleValue = roleVals[i]
		for j := 0; j < len(tenant.Configuration.Roles); j ++ {
			if tenant.Configuration.Roles[j].RoleValue == roleVals[i] {
				roles[i].RoleTitle = tenant.Configuration.Roles[j].RoleTitle
			}
		}
	}

	return roles, nil
}
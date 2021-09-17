package service

import (
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func TenantUserAdd(uid, currTenantID uint64, depIds []uint64, roles []string, disable int8) (e error) {
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
			common.Logger.Sugar().Errorf("TenantUserAdd AddRoleForUserInDomain ERR: %v", e)
			return common.ErrService
		}
	}

	if e = TenantUserSetDepartment(uid, currTenantID, depIds); e != nil {
		common.Logger.Sugar().Warnf("TenantUserAdd TenantUserSetDepartment ERR: %v", e)
		e = nil
	}

	if e = TenantUserDisabledService(uid, currTenantID, disable); e != nil {
		common.Logger.Sugar().Warnf("TenantUserAdd TenantUserDisabledService ERR: %v", e)
		e = nil
	}

	return
}

func TenantUserDel(uid, currTenantID uint64) (r int64, e error) {
	// 删除所有角色
	if e = accessctl.DeleteRolesForUserInDomain(uid, currTenantID); e != nil {
		common.Logger.Sugar().Errorf("TenantUserDel ERR: %v", e)
		return 0, common.ErrService
	}

	if r, e = dao.UserUpdateTenantID(uid, 0, currTenantID); e != nil {
		common.Logger.Sugar().Errorf("TenantUserDel ERR: %v", e)
		return 0, common.ErrService
	}

	return
}

func TenantUserGet(tenantID, page, pageSize uint64, nickname string, uids []uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	var rr []protos.User
	rr, e = dao.UserSelectByTenant(tenantID, page, pageSize, nickname, uids)
	if e != nil {
		common.Logger.Sugar().Error("TenantUserGet db ERR: %v", e)
		e = common.ErrService
		return
	}
	if len(rr) == 0 {
		e = common.ErrNull
		return
	}
	rst.List = rr

	// 部门字典
	departments, err := DepartmentFind(0, tenantID)
	if err != nil {
		common.Logger.Sugar().Error("TenantUserGet DepartmentFind ERR: %v", e)
		e = common.ErrService
		return
	}

	// role
	for i := 0; i < len(rr); i++ {
		if rr[i].Roles, e = getTenantUserRoles(rr[i].UID, rr[i].TenantID); e != nil {
			common.Logger.Sugar().Warnf("TenantUserGet getTenantUserRole ERR: %v", e)
		}

		if deps, ok := rr[i].Ext["deps"].([]interface{}); ok {
			for _, dep := range deps {
				for j := 0; j < len(departments); j++ {
					if uint64(dep.(float64)) == departments[j].Id {
						rr[i].Departments = append(rr[i].Departments, departments[j])
						break
					}
				}
			}
		}
	}

	if hasTotal {
		rst.Total, e = dao.UserCountByTenant(tenantID, nickname, uids)
		if e != nil {
			common.Logger.Sugar().Error("TenantUserGet db ERR: %v", e)
			e = common.ErrService
			return
		}
	}

	return
}

func TenantUserDisabledService(uid, currTenantID uint64, disabled int8) (e error) {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUserDisabledService ERR: %d %v %v", uid, currTenantID, disabled)
		return common.ErrParam
	}

	return TenantUpdateUserExt(uid, currTenantID, "disabled", disabled)
}

func TenantUserSetDepartment(uid, tenantId uint64, departmentIds []uint64) error {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUserSetDepartment ERR: %d %v %v", uid, tenantId, departmentIds)
		return common.ErrParam
	}

	if len(departmentIds) == 0 {
		common.Logger.Sugar().Infof("TenantUserSetDepartment nil: %d %v", uid, tenantId)
		return TenantUpdateUserExt(uid, tenantId, "deps", nil)
	}

	return TenantUpdateUserExt(uid, tenantId, "deps", departmentIds)
}

func TenantUpdateUserExt(uid, currTenantID uint64, k string, v interface{}) error {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %d %v %v", uid, k, v)
		return common.ErrParam
	}
	if k == "" {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %d %v %v", uid, k, v)
		return common.ErrParam
	}

	userInfo, e := dao.UserSelectByID(uid)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantUserDisabledService db ERR: %v", e)
		return common.ErrNull
	}

	if userInfo.TenantID != currTenantID {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt tenant ERR: %v %v %v", uid, currTenantID, userInfo)
		return common.ErrNoAuth
	}

	common.Logger.Sugar().Infof("TenantUpdateUserExt: %v %v %v %v", uid, currTenantID, k, v)
	userInfo.Ext[k] = v
	if v == nil {
		common.Logger.Sugar().Warnf("TenantUpdateUserExt delete: %v", k)
		delete(userInfo.Ext, k)
	}

	rows, e := dao.UserUpdateExt(uid, &userInfo.Ext)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt ERR: %v", e)
		return common.ErrService
	}
	if rows < 1 {
		common.Logger.Sugar().Warnf("TenantUpdateUserExt RowsAffected 0")
	}

	return nil
}

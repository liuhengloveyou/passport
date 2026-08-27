package service

import (
	"github.com/liuhengloveyou/passport/v4/accessctl"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/protos"
	"go.uber.org/zap"
)

func TenantBindUser(uid, currTenantID uint64) error {
	if uid == 0 || currTenantID == 0 {
		return common.ErrParam
	}
	row, e := dao.UserUpdateTenantID(uid, currTenantID, 0)
	if e != nil {
		return common.ErrService
	}
	if row == 1 {
		return nil
	}
	userInfo, qErr := dao.UserQueryByID(uid)
	if qErr != nil || userInfo == nil {
		return common.ErrService
	}
	if userInfo.TenantID != currTenantID {
		return common.ErrService
	}
	return nil
}

func TenantUserAdd(uid, currTenantID, orgID uint64, depIds []uint64, roles []string, disable protos.UserDisableStatus) (e error) {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	if _, e = RequireOrg(currTenantID, orgID); e != nil {
		return e
	}

	row, e := dao.UserUpdateTenantID(uid, currTenantID, 0)
	if e != nil {
		common.Logger.Sugar().Error("TenantUserAdd db ERR: ", e)
		return common.ErrService
	}
	if row != 1 {
		userInfo, qErr := dao.UserQueryByID(uid)
		if qErr != nil || userInfo == nil {
			common.Logger.Sugar().Error("TenantUserAdd UserUpdateTenantID query ERR: ", row, qErr)
			return common.ErrService
		}
		if userInfo.TenantID != currTenantID {
			common.Logger.Sugar().Error("TenantUserAdd UserUpdateTenantID tenant mismatch: ", row, userInfo.TenantID, currTenantID)
			return common.ErrService
		}
		common.Logger.Sugar().Warnf("TenantUserAdd UserUpdateTenantID skipped: uid=%d already in tenant=%d", uid, currTenantID)
	}

	if e = OrgAddMember(orgID, uid, currTenantID); e != nil {
		return e
	}

	for _, role := range roles {
		if e = accessctl.AddRoleForUserInDomain(uid, currTenantID, orgID, role); e != nil {
			common.Logger.Sugar().Errorf("TenantUserAdd AddRoleForUserInDomain ERR: %v", e)
			return common.ErrService
		}
	}

	if e = TenantUserSetDepartment(uid, currTenantID, orgID, depIds); e != nil {
		common.Logger.Sugar().Errorf("TenantUserAdd TenantUserSetDepartment ERR: %v", e)
		return e
	}

	if e = TenantUserDisabledService(uid, currTenantID, disable); e != nil {
		common.Logger.Sugar().Warnf("TenantUserAdd TenantUserDisabledService ERR: %v", e)
		e = nil
	}

	return
}

func TenantUserDel(uid, currTenantID uint64) (r int64, e error) {
	orgs, err := dao.OrgListByUser(uid, currTenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantUserDel list org ERR: %v", err)
		return 0, common.ErrService
	}
	for i := range orgs {
		if e = OrgRemoveMember(orgs[i].ID, uid, currTenantID); e != nil {
			common.Logger.Sugar().Errorf("TenantUserDel ERR: %v", e)
			return 0, common.ErrService
		}
	}

	if r, e = dao.UserDelete(uid, currTenantID); e != nil {
		common.Logger.Sugar().Errorf("TenantUserDel ERR: %v", e)
		return 0, common.ErrService
	}

	common.Logger.Warn("TenantUserDel: ", zap.Uint64("uid", uid), zap.Uint64("tid", currTenantID), zap.Int64("r", r), zap.Any("e", e))

	return
}

func TenantUserLeaveOrg(uid, currTenantID, orgID uint64) (r int64, e error) {
	if orgID == 0 {
		return 0, common.ErrOrgRequired
	}
	if e = OrgRemoveMember(orgID, uid, currTenantID); e != nil {
		return 0, e
	}
	_ = TenantUserSetDepartment(uid, currTenantID, orgID, nil)

	n, err := dao.OrgMemberCountByUser(uid, currTenantID)
	if err != nil {
		return 0, common.ErrService
	}
	if n > 0 {
		return 1, nil
	}
	return TenantUserDel(uid, currTenantID)
}

func TenantUserGet(tenantID, orgID, page, pageSize uint64, nickname string, uids []uint64, hasTotal bool) (rst protos.PageResponse, e error) {
	if orgID == 0 {
		e = common.ErrOrgRequired
		return
	}
	if _, e = RequireOrg(tenantID, orgID); e != nil {
		return
	}

	var rr []protos.User
	rr, e = dao.UserQueryByOrg(tenantID, orgID, page, pageSize, nickname, uids)
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

	departments, err := DepartmentFind(0, tenantID, orgID, 0, 0)
	if err != nil {
		common.Logger.Sugar().Error("TenantUserGet DepartmentFind ERR: %v", err)
		e = common.ErrService
		return
	}
	depSet := make(map[uint64]protos.Department, len(departments))
	for j := range departments {
		depSet[departments[j].Id] = departments[j]
	}

	for i := 0; i < len(rr); i++ {
		if rr[i].Roles, e = getTenantUserRoles(rr[i].UID, rr[i].TenantID, orgID); e != nil {
			common.Logger.Sugar().Warnf("TenantUserGet getTenantUserRole ERR: %v", e)
		}

		for _, depID := range parseUint64Slice(rr[i].Ext[protos.DepartmentExtKey]) {
			if d, ok := depSet[depID]; ok {
				rr[i].Departments = append(rr[i].Departments, d)
			}
		}
	}

	if hasTotal {
		rst.Total, e = dao.UserCountByOrg(tenantID, orgID, nickname, uids)
		if e != nil {
			common.Logger.Sugar().Error("TenantUserGet db ERR: %v", e)
			e = common.ErrService
			return
		}
	}

	return
}

func TenantUserDisabledService(uid, currTenantID uint64, disabled protos.UserDisableStatus) (e error) {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUserDisabledService ERR: %d %v %v", uid, currTenantID, disabled)
		return common.ErrParam
	}
	if !disabled.IsValid() {
		common.Logger.Sugar().Errorf("TenantUserDisabledService invalid disable status: %d %v %v", uid, currTenantID, disabled)
		return common.ErrParam
	}

	return TenantUpdateUserExt(uid, currTenantID, "disabled", int8(disabled))
}

func TenantUserSetDepartment(uid, tenantId, orgID uint64, departmentIds []uint64) error {
	if uid <= 0 {
		common.Logger.Sugar().Errorf("TenantUserSetDepartment ERR: %d %v %v", uid, tenantId, departmentIds)
		return common.ErrParam
	}
	if orgID == 0 {
		return common.ErrOrgRequired
	}

	orgDeps, err := DepartmentFind(0, tenantId, orgID, 0, 0)
	if err != nil {
		return err
	}
	orgDepSet := make(map[uint64]struct{}, len(orgDeps))
	for i := range orgDeps {
		orgDepSet[orgDeps[i].Id] = struct{}{}
	}
	for _, id := range departmentIds {
		if id == 0 {
			continue
		}
		if _, ok := orgDepSet[id]; !ok {
			return common.ErrParam
		}
	}

	userInfo, e := dao.UserQueryByID(uid)
	if e != nil || userInfo == nil {
		return common.ErrNull
	}
	if userInfo.TenantID != tenantId {
		return common.ErrNoAuth
	}

	kept := make([]uint64, 0)
	for _, id := range parseUint64Slice(userInfo.Ext[protos.DepartmentExtKey]) {
		if _, inOrg := orgDepSet[id]; inOrg {
			continue
		}
		kept = append(kept, id)
	}
	for _, id := range departmentIds {
		if id > 0 {
			kept = append(kept, id)
		}
	}

	// 同步 Casbin 部门关联：移除本 org 旧部门、加入新部门
	if err := accessctl.SyncUserDepsInOrg(uid, tenantId, orgID, departmentIds); err != nil {
		common.Logger.Sugar().Errorf("TenantUserSetDepartment SyncUserDepsInOrg ERR: uid=%d deps=%v err=%v", uid, departmentIds, err)
		return err
	}

	if len(kept) == 0 {
		return TenantUpdateUserExt(uid, tenantId, "deps", nil)
	}
	return TenantUpdateUserExt(uid, tenantId, "deps", kept)
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

	userInfo, e := dao.UserQueryByID(uid)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt db ERR: %v", e)
		return common.ErrNull
	}

	if userInfo.TenantID != currTenantID {
		common.Logger.Sugar().Errorf("TenantUpdateUserExt tenant ERR: %v %v %v", uid, currTenantID, userInfo)
		return common.ErrNoAuth
	}

	common.Logger.Sugar().Infof("TenantUpdateUserExt: %v %v %v %v", uid, currTenantID, k, v)
	if userInfo.Ext == nil {
		userInfo.Ext = protos.MapStruct{}
	}
	if v == nil {
		common.Logger.Sugar().Warnf("TenantUpdateUserExt delete: %v", k)
		delete(userInfo.Ext, k)
	} else {
		userInfo.Ext[k] = v
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

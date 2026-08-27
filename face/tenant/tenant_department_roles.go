// tenant_department_roles.go 部门角色继承管理接口。
//
// 基于 Casbin g 策略链式继承：
//
//	g, dep-3, waiter,  domain      部门3 拥有 waiter 角色
//	g, uid-100, dep-3,  domain     用户100 属于部门3
//	=> Casbin 自动推导: uid-100 → dep-3 → waiter → policy(放行)
//
// 部门角色变更后全体成员自动生效，无需逐人更新。
package tenant

import (
	"net/http"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/accessctl"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/face/core"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/service"
)

type depRolesReq struct {
	DepID uint64   `json:"depId" validate:"required"`
	Roles []string `json:"roles"`
}

type depJoinReq struct {
	DepID uint64   `json:"depId" validate:"required"`
	UIDs  []uint64 `json:"uids" validate:"required"`
}

type depLeaveReq struct {
	DepID uint64 `json:"depId" validate:"required"`
	UID   uint64 `json:"uid" validate:"required"`
}

type depRolesCtx struct {
	User  protos.User
	OrgID uint64
}

func depRolesSession(w http.ResponseWriter, r *http.Request) (depRolesCtx, bool) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return depRolesCtx{}, false
	}
	orgID, err := core.SessionOrgID(r, sessionUser.TenantID)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return depRolesCtx{}, false
	}
	return depRolesCtx{User: sessionUser, OrgID: orgID}, true
}

func requireOwnedDepartment(w http.ResponseWriter, tenantID, orgID, depID uint64) bool {
	owned, err := service.DepartmentFind(depID, tenantID, orgID, 0, 0)
	if err != nil || len(owned) != 1 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNull)
		return false
	}
	return true
}

func tenantRoleSet(tenantID uint64) map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, roleConf := range service.TenantGetRole(tenantID) {
		allowed[roleConf.RoleValue] = struct{}{}
	}
	return allowed
}

func sanitizeDepRoles(roles []string, allowed map[string]struct{}) ([]string, bool) {
	clean := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, ok := allowed[role]; !ok {
			return nil, false
		}
		clean = append(clean, role)
	}
	return clean, true
}

func validateTenantOrgUser(w http.ResponseWriter, uid, tenantID, orgID uint64) bool {
	if uid <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return false
	}
	userInfo, err := dao.UserQueryByID(uid)
	if err != nil || userInfo == nil || userInfo.TenantID != tenantID {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return false
	}
	if err := service.UserInOrg(uid, tenantID, orgID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return false
	}
	return true
}

func validateTenantOrgUserBatch(uid, tenantID, orgID uint64) bool {
	if uid <= 0 {
		return false
	}
	userInfo, err := dao.UserQueryByID(uid)
	if err != nil || userInfo == nil || userInfo.TenantID != tenantID {
		return false
	}
	return service.UserInOrg(uid, tenantID, orgID) == nil
}

// DepartmentGetRoles 获取部门角色列表
func DepartmentGetRoles(w http.ResponseWriter, r *http.Request) {
	ctx, ok := depRolesSession(w, r)
	if !ok {
		return
	}

	var req depRolesReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 4096); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if !requireOwnedDepartment(w, ctx.User.TenantID, ctx.OrgID, req.DepID) {
		return
	}

	roles := accessctl.GetDepRoles(req.DepID, ctx.User.TenantID, ctx.OrgID)
	gocommon.HttpErr(w, http.StatusOK, 0, map[string]any{
		"roles": roles,
	})
}

// DepartmentSetRoles 设置部门角色（全量覆盖）
func DepartmentSetRoles(w http.ResponseWriter, r *http.Request) {
	ctx, ok := depRolesSession(w, r)
	if !ok {
		return
	}

	var req depRolesReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 4096); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if !requireOwnedDepartment(w, ctx.User.TenantID, ctx.OrgID, req.DepID) {
		return
	}

	cleanRoles, valid := sanitizeDepRoles(req.Roles, tenantRoleSet(ctx.User.TenantID))
	if !valid {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.SetDepRoles(req.DepID, ctx.User.TenantID, ctx.OrgID, cleanRoles); err != nil {
		common.Logger.Sugar().Errorf("DepartmentSetRoles ERR: dep=%d err=%v", req.DepID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	synced, syncFailed, err := service.DepartmentSyncMemberLinks(req.DepID, ctx.User.TenantID, ctx.OrgID)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentSetRoles sync ERR: dep=%d err=%v", req.DepID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	common.Logger.Sugar().Infof("DepartmentSetRoles: dep=%d roles=%v synced=%d failed=%d",
		req.DepID, cleanRoles, synced, syncFailed)
	gocommon.HttpErr(w, http.StatusOK, 0, map[string]any{
		"roles":  cleanRoles,
		"synced": synced,
	})
}

// DepartmentJoinUsers 批量让用户加入部门（继承部门角色）
func DepartmentJoinUsers(w http.ResponseWriter, r *http.Request) {
	ctx, ok := depRolesSession(w, r)
	if !ok {
		return
	}

	var req depJoinReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 4096); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if !requireOwnedDepartment(w, ctx.User.TenantID, ctx.OrgID, req.DepID) {
		return
	}

	success, failed := 0, 0
	for _, uid := range req.UIDs {
		if !validateTenantOrgUserBatch(uid, ctx.User.TenantID, ctx.OrgID) {
			common.Logger.Sugar().Warnf("DepartmentJoinUsers skip invalid uid=%d dep=%d", uid, req.DepID)
			failed++
			continue
		}
		if err := accessctl.JoinDepForUser(uid, req.DepID, ctx.User.TenantID, ctx.OrgID); err != nil {
			common.Logger.Sugar().Errorf("DepartmentJoinUsers ERR: uid=%d dep=%d err=%v", uid, req.DepID, err)
			failed++
			continue
		}
		success++
	}

	common.Logger.Sugar().Infof("DepartmentJoinUsers: dep=%d users=%d success=%d failed=%d",
		req.DepID, len(req.UIDs), success, failed)

	gocommon.HttpErr(w, http.StatusOK, 0, map[string]int{
		"success": success,
		"failed":  failed,
	})
}

// DepartmentLeaveUser 用户离开部门（移除角色继承）
func DepartmentLeaveUser(w http.ResponseWriter, r *http.Request) {
	ctx, ok := depRolesSession(w, r)
	if !ok {
		return
	}

	var req depLeaveReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 4096); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if !requireOwnedDepartment(w, ctx.User.TenantID, ctx.OrgID, req.DepID) {
		return
	}
	if !validateTenantOrgUser(w, req.UID, ctx.User.TenantID, ctx.OrgID) {
		return
	}

	if err := accessctl.LeaveDepForUser(req.UID, req.DepID, ctx.User.TenantID, ctx.OrgID); err != nil {
		common.Logger.Sugar().Errorf("DepartmentLeaveUser ERR: uid=%d dep=%d err=%v", req.UID, req.DepID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	common.Logger.Sugar().Infof("DepartmentLeaveUser: uid=%d dep=%d", req.UID, req.DepID)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

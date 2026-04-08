// access_role.go 提供角色分配相关接口：用户角色增删改查。
package access

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// AddRoleForUser 为指定用户添加角色。
func AddRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	req := &protos.RoleStruct{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil || strings.TrimSpace(req.RoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, strings.TrimSpace(req.RoleValue)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// UpdateRoleForUser 更新指定用户角色（先删旧角色再加新角色）。
func UpdateRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	req := &protos.RoleReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil || strings.TrimSpace(req.RoleValue) == "" || strings.TrimSpace(req.NewRoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, strings.TrimSpace(req.RoleValue)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, strings.TrimSpace(req.NewRoleValue)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// RemoveRoleForUser 移除指定用户的角色。
func RemoveRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	req := &protos.RoleStruct{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil || strings.TrimSpace(req.RoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, req.RoleValue); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// GetRolesForMe 查询当前登录用户的角色列表。
func GetRolesForMe(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID)
	rst := make([]protos.RoleStruct, len(roles))
	rolesConfs := service.TenantGetRole(sessionUser.TenantID)
	for i, role := range roles {
		rst[i].RoleValue = role
		for _, roleConf := range rolesConfs {
			if role == roleConf.RoleValue {
				rst[i].RoleTitle = roleConf.RoleTitle
			}
		}
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rst)
}

// GetRolesForUser 查询指定用户的角色列表。
func GetRolesForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	iuid, _ := strconv.ParseUint(r.FormValue("uid"), 10, 64)
	if iuid <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	roles := accessctl.GetRoleForUserInDomain(iuid, sessionUser.TenantID)
	rst := make([]protos.RoleStruct, len(roles))
	rolesConfs := service.TenantGetRole(sessionUser.TenantID)
	for i, role := range roles {
		rst[i].RoleValue = role
		for _, roleConf := range rolesConfs {
			if role == roleConf.RoleValue {
				rst[i].RoleTitle = roleConf.RoleTitle
			}
		}
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rst)
}

// GetUsersForRole 查询指定角色下的用户列表。
func GetUsersForRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	roleName := r.FormValue("role")
	if roleName == "" || len(roleName) > 100 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	roles := accessctl.GetUsersForRoleInDomain(roleName, sessionUser.TenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, roles)
}

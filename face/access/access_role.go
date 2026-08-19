// access_role.go 提供角色分配相关接口：用户角色增删改查。
package access

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/accessctl"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/face/core"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/service"
)

func sessionOrg(w http.ResponseWriter, r *http.Request) (protos.User, uint64, bool) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return sessionUser, 0, false
	}
	orgID, err := core.SessionOrgID(r, sessionUser.TenantID)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return sessionUser, 0, false
	}
	return sessionUser, orgID, true
}

func AddRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser, orgID, ok := sessionOrg(w, r)
	if !ok {
		return
	}
	req := &protos.RoleStruct{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil || strings.TrimSpace(req.RoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, orgID, strings.TrimSpace(req.RoleValue)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func UpdateRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser, orgID, ok := sessionOrg(w, r)
	if !ok {
		return
	}
	req := &protos.RoleReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil || strings.TrimSpace(req.RoleValue) == "" || strings.TrimSpace(req.NewRoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, orgID, strings.TrimSpace(req.RoleValue)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, orgID, strings.TrimSpace(req.NewRoleValue)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func RemoveRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser, orgID, ok := sessionOrg(w, r)
	if !ok {
		return
	}
	req := &protos.RoleStruct{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil || strings.TrimSpace(req.RoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, orgID, req.RoleValue); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func GetRolesForMe(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	orgID := core.ParseOrgID(r)
	if orgID == 0 || service.UserInOrg(sessionUser.UID, sessionUser.TenantID, orgID) != nil {
		gocommon.HttpErr(w, http.StatusOK, 0, []protos.RoleStruct{})
		return
	}
	roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID, orgID)
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

func GetRolesForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser, orgID, ok := sessionOrg(w, r)
	if !ok {
		return
	}
	iuid, _ := strconv.ParseUint(r.FormValue("uid"), 10, 64)
	if iuid <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	roles := accessctl.GetRoleForUserInDomain(iuid, sessionUser.TenantID, orgID)
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

func GetUsersForRole(w http.ResponseWriter, r *http.Request) {
	sessionUser, orgID, ok := sessionOrg(w, r)
	if !ok {
		return
	}
	roleName := r.FormValue("role")
	if roleName == "" || len(roleName) > 100 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	roles := accessctl.GetUsersForRoleInDomain(roleName, sessionUser.TenantID, orgID)
	gocommon.HttpErr(w, http.StatusOK, 0, roles)
}

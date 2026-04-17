// admin_user.go 提供平台管理员用户管理接口。
package admin

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/face/tenant"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

type adminUserEditReq struct {
	UID         uint64   `json:"uid"`
	Disable     *int8    `json:"disable,omitempty"`
	Pwd         string   `json:"pwd,omitempty"`
	DepIds      []uint64 `json:"depIds,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Description *string  `json:"description,omitempty"`
	MacAddr     *string  `json:"macAddr,omitempty"`
}

// UserList 按租户分页查询用户列表。
func UserList(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	tenantID, _ := strconv.ParseUint(r.FormValue("tenantID"), 10, 64)
	nickname := strings.TrimSpace(r.FormValue("nickname"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	if tenantID == 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := authorizeAdminTenant(sessionUser, "admin.user.list", tenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}
	rr, e := service.AdminUserList(tenantID, page, pageSize, hasTotal == 1, nickname)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

// UserDel 由平台管理员删除用户（按 uid 删除其所在租户下的用户记录）。
func UserDel(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)

	req := &protos.Tenant{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.UID == 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	userInfo, err := service.GetUserInfo(req.UID)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	if userInfo.UID == 0 || userInfo.TenantID == 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err = authorizeAdminTenant(sessionUser, "admin.user.del", userInfo.TenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}

	if _, err = service.TenantUserDel(req.UID, userInfo.TenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// UserEdit 由平台管理员统一编辑用户信息（启用状态、角色、部门、描述、MAC、密码）。
func UserEdit(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)

	req := &adminUserEditReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 4096); err != nil || req.UID == 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	userInfo, err := service.GetUserInfo(req.UID)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	if userInfo.UID == 0 || userInfo.TenantID == 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	tenantID := userInfo.TenantID

	if err = authorizeAdminTenant(sessionUser, "admin.user.edit", tenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}

	if req.Disable != nil {
		if *req.Disable != 0 && *req.Disable != 1 {
			gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
			return
		}
		if err = service.TenantUserDisabledService(req.UID, tenantID, *req.Disable); err != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}

	if req.Description != nil {
		if err = service.TenantUpdateUserExt(req.UID, tenantID, "description", strings.TrimSpace(*req.Description)); err != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}
	if req.MacAddr != nil {
		if err = service.TenantUpdateUserExt(req.UID, tenantID, "macAddr", strings.TrimSpace(*req.MacAddr)); err != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}

	if req.DepIds != nil {
		if err = service.TenantUserSetDepartment(req.UID, tenantID, req.DepIds); err != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}

	if req.Roles != nil {
		oldRoles := accessctl.GetRoleForUserInDomain(req.UID, tenantID)
		oldSet := make(map[string]struct{}, len(oldRoles))
		for _, role := range oldRoles {
			role = strings.TrimSpace(role)
			if role != "" {
				oldSet[role] = struct{}{}
			}
		}

		newSet := make(map[string]struct{}, len(req.Roles))
		for _, role := range req.Roles {
			role = strings.TrimSpace(role)
			if role != "" {
				newSet[role] = struct{}{}
			}
		}

		for role := range newSet {
			if _, ok := oldSet[role]; ok {
				delete(newSet, role)
				delete(oldSet, role)
			}
		}

		for role := range newSet {
			if err = accessctl.AddRoleForUserInDomain(req.UID, tenantID, role); err != nil {
				gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
				return
			}
		}
		for role := range oldSet {
			if err = accessctl.DeleteRoleForUserInDomain(req.UID, tenantID, role); err != nil {
				gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
				return
			}
		}
	}

	pwd := strings.TrimSpace(req.Pwd)
	if pwd != "" {
		if len(pwd) < 4 || len(pwd) > 16 {
			gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
			return
		}
		if _, err = service.SetUserPWD(req.UID, tenantID, pwd); err != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// ModifyUserPassword 由平台管理员修改用户密码（不限制租户上下文）。
func ModifyUserPassword(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if err := authorizeAdminTenant(sessionUser, "admin.modifyUserPassword", sessionUser.TenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}

	// 细粒度权限由路由 NeedAccess + AccessFilter 统一校验。
	tenant.ModifyPWDByUIDWithTenant(w, r, false)
}

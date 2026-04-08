// admin_user.go 提供平台管理员用户管理接口。
package admin

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/face/tenant"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserList 按租户分页查询用户列表。
func UserList(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoAuth)
		return
	}
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
	rr, e := service.AdminUserList(tenantID, page, pageSize, hasTotal == 1, nickname)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

// ModifyUserPassword 由平台管理员修改用户密码（不限制租户上下文）。
func ModifyUserPassword(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoAuth)
		return
	}

	// 细粒度权限由路由 NeedAccess + AccessFilter 统一校验。
	tenant.ModifyPWDByUIDWithTenant(w, r, false)
}

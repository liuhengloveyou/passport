// tenant_tenant.go 提供租户本身相关接口：创建、配置读写、租户树查询。
package tenant

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// Add 为当前登录用户创建租户（仅允许未归属租户的用户发起）。
func Add(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		return
	}
	if sessionUser.TenantID > 0 {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrTenantLimit)
		return
	}
	req := &protos.Tenant{UID: sessionUser.UID}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantType = strings.TrimSpace(req.TenantType)
	uid, err := service.TenantAdd(req)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, uid)
}

// UpdateConfiguration 更新当前租户配置（请求体为 JSON 对象）。
func UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req map[string]interface{}
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := service.TenantUpdateConfiguration(sessionUser.TenantID, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// LoadConfiguration 按 key 读取当前租户配置。
func LoadConfiguration(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	k := strings.TrimSpace(r.FormValue("k"))
	confMap, err := service.TenantLoadConfiguration(sessionUser.TenantID, k)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, confMap)
}

// TreeList 分页查询租户树节点，支持按父节点筛选。
func TreeList(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	parentID, _ := strconv.ParseUint(r.FormValue("pid"), 10, 64)
	hasTotal, _ := strconv.ParseInt(r.FormValue("hasTotal"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	if parentID == 0 {
		parentID = sessionUser.TenantID
	}
	rr, e := service.TenantTreeList(&sessionUser, parentID, page, pageSize, hasTotal == 1)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

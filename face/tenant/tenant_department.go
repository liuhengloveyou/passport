// tenant_department.go 提供租户部门管理接口：新增、删除、更新、配置更新与列表查询。
package tenant

import (
	"net/http"
	"strconv"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	"go.uber.org/zap"
)

// DepartmentAdd 在当前租户下创建部门。
func DepartmentAdd(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.Department
	if err := core.ReadJSONBodyFromRequest(r, &req, 2048); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	req.TenantID = sessionUser.TenantID
	req.UserId = sessionUser.UID
	lastInsertId, err := service.DepartmentCreate(&req)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, lastInsertId)
}

// DepartmentDelete 删除当前租户下指定部门。
func DepartmentDelete(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 || id <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := service.DepartmentDelete(id, sessionUser.TenantID); err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// DepartmentUpdate 更新当前租户下部门基础信息。
func DepartmentUpdate(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.Department
	if err := core.ReadJSONBodyFromRequest(r, &req, 2048); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	req.TenantID = sessionUser.TenantID
	if err := service.DepartmentUpdate(&req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// DepartmentUpdateConfig 更新部门配置项。
func DepartmentUpdateConfig(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.KvReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 2048); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := service.DepartmentUpdateConfig(req.ID, sessionUser.UID, sessionUser.TenantID, req.K, req.V); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// DepartmentList 查询当前租户部门列表，支持分页。
func DepartmentList(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("page_size"), 10, 64)
	list, err := service.DepartmentFind(id, sessionUser.TenantID, page, pageSize)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err)
		return
	}
	core.Logger().Debug("DepartmentList", zap.Int("len", len(list)))
	gocommon.HttpErr(w, http.StatusOK, 0, list)
}

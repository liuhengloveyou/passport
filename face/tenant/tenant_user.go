// tenant_user.go 提供租户成员管理接口：成员增删改查、禁用、分配部门与密码重置。
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

// UserAdd 为当前租户新增成员，支持直接绑定已有 UID 或先创建用户再入租户。
func UserAdd(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	req := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if len(req.Roles) > 10 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.UID == 0 {
		nuid, e := service.AddUserService(req)
		if e != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, e)
			return
		}
		req.UID = nuid
	}
	if err := service.TenantUserAdd(req.UID, sessionUser.TenantID, req.DepIds, req.Roles, req.Disable); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, req.UID)
}

// UserDel 将指定用户从当前租户移除。
func UserDel(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	req := &protos.Tenant{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, e := service.TenantUserDel(req.UID, sessionUser.TenantID); e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// UserGet 分页查询当前租户成员，支持昵称和 UID 列表筛选。
func UserGet(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	r.ParseForm()
	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	nickname := strings.TrimSpace(r.FormValue("nickname"))
	uidStr := r.FormValue("uids")
	var uids []uint64
	if uidStr != "" {
		uidss := strings.Split(uidStr, ",")
		uids = make([]uint64, len(uidss))
		for i, ouids := range uidss {
			u, err := strconv.ParseUint(ouids, 10, 64)
			if err != nil {
				gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
				return
			}
			uids[i] = u
		}
	}
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	rr, e := service.TenantUserGet(sessionUser.TenantID, page, pageSize, nickname, uids, hasTotal == 1)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

// UserDisableByUID 启用或禁用当前租户内的指定用户。
func UserDisableByUID(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.DisableUserReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil || req.UID <= 0 || (req.Disable != 0 && req.Disable != 1) {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := service.TenantUserDisabledService(req.UID, sessionUser.TenantID, int8(req.Disable)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// UserModifyExtInfo 更新租户成员的扩展字段。
func UserModifyExtInfo(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.KvReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := service.TenantUpdateUserExt(req.ID, sessionUser.TenantID, req.K, req.V); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// ModifyUserPassword 修改当前租户下指定用户密码（租户范围校验）。
func ModifyUserPassword(w http.ResponseWriter, r *http.Request) { ModifyPWDByUIDWithTenant(w, r, true) }

// ModifyPWDByUIDWithTenant 按 UID 修改密码，可选择是否校验租户上下文。
func ModifyPWDByUIDWithTenant(w http.ResponseWriter, r *http.Request, withTenant bool) {
	sessionUser := core.GetSessionUser(r)
	if withTenant && (sessionUser.UID <= 0 || sessionUser.TenantID <= 0) {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req map[string]interface{}
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	uid, ok := req["uid"].(float64)
	pwd, ok2 := req["pwd"].(string)
	pwd = strings.TrimSpace(pwd)
	if !ok || !ok2 || uint64(uid) <= 0 || len(pwd) < 4 || len(pwd) > 16 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	tid := uint64(0)
	if withTenant {
		tid = sessionUser.TenantID
	}
	if _, err := service.SetUserPWD(uint64(uid), tid, pwd); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

// UserSetDepartment 为当前租户成员设置部门列表。
func UserSetDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.SetDepartmentReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil || req.UID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := service.TenantUserSetDepartment(req.UID, sessionUser.TenantID, req.DepIds); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

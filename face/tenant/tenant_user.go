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
		common.Logger.Sugar().Errorf("tenant.UserAdd no auth: method=%s uri=%s uid=%d tenant=%d", r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	req := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserAdd bad request body: method=%s uri=%s tenant=%d err=%v", r.Method, r.RequestURI, sessionUser.TenantID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if len(req.Roles) > 10 {
		common.Logger.Sugar().Errorf("tenant.UserAdd roles too many: tenant=%d req_uid=%d roles=%d", sessionUser.TenantID, req.UID, len(req.Roles))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserAdd start: operator_uid=%d tenant=%d req_uid=%d roles=%v depIds=%v disable=%d",
		sessionUser.UID, sessionUser.TenantID, req.UID, req.Roles, req.DepIds, req.Disable)
	if req.UID == 0 {
		nuid, e := service.AddUserService(req)
		if e != nil {
			common.Logger.Sugar().Errorf("tenant.UserAdd AddUserService failed: tenant=%d nickname=%s err=%v", sessionUser.TenantID, strings.TrimSpace(req.Nickname), e)
			gocommon.HttpJsonErr(w, http.StatusOK, e)
			return
		}
		req.UID = nuid
		common.Logger.Sugar().Infof("tenant.UserAdd created base user: tenant=%d new_uid=%d", sessionUser.TenantID, req.UID)
	}
	if err := service.TenantUserAdd(
		req.UID,
		sessionUser.TenantID,
		req.DepIds,
		req.Roles,
		protos.UserDisableStatus(req.Disable),
	); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserAdd TenantUserAdd failed: tenant=%d uid=%d err=%v", sessionUser.TenantID, req.UID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserAdd success: tenant=%d uid=%d", sessionUser.TenantID, req.UID)
	gocommon.HttpErr(w, http.StatusOK, 0, req.UID)
}

// UserDel 将指定用户从当前租户移除。
func UserDel(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		common.Logger.Sugar().Errorf("tenant.UserDel no auth: method=%s uri=%s uid=%d tenant=%d", r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	req := &protos.Tenant{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserDel bad request body: method=%s uri=%s tenant=%d err=%v", r.Method, r.RequestURI, sessionUser.TenantID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserDel start: operator_uid=%d tenant=%d target_uid=%d",
		sessionUser.UID, sessionUser.TenantID, req.UID)
	if _, e := service.TenantUserDel(req.UID, sessionUser.TenantID); e != nil {
		common.Logger.Sugar().Errorf("tenant.UserDel failed: tenant=%d target_uid=%d err=%v", sessionUser.TenantID, req.UID, e)
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserDel success: tenant=%d target_uid=%d", sessionUser.TenantID, req.UID)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// UserGet 分页查询当前租户成员，支持昵称和 UID 列表筛选。
func UserGet(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		common.Logger.Sugar().Errorf("tenant.UserGet no auth: method=%s uri=%s uid=%d tenant=%d", r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID)
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
				common.Logger.Sugar().Errorf("tenant.UserGet bad uid list: tenant=%d raw_uids=%s err=%v", sessionUser.TenantID, uidStr, err)
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
	common.Logger.Sugar().Infof("tenant.UserGet start: operator_uid=%d tenant=%d page=%d pageSize=%d hasTotal=%t nickname=%q uids=%v",
		sessionUser.UID, sessionUser.TenantID, page, pageSize, hasTotal == 1, nickname, uids)
	rr, e := service.TenantUserGet(sessionUser.TenantID, page, pageSize, nickname, uids, hasTotal == 1)
	if e != nil {
		common.Logger.Sugar().Errorf("tenant.UserGet failed: tenant=%d page=%d pageSize=%d err=%v", sessionUser.TenantID, page, pageSize, e)
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserGet success: tenant=%d page=%d pageSize=%d", sessionUser.TenantID, page, pageSize)
	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

// UserDisableByUID 启用或禁用当前租户内的指定用户。
func UserDisableByUID(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		common.Logger.Sugar().Errorf("tenant.UserDisableByUID no auth: method=%s uri=%s uid=%d tenant=%d", r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.DisableUserReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil || req.UID <= 0 || !protos.UserDisableStatus(req.Disable).IsValid() {
		common.Logger.Sugar().Errorf("tenant.UserDisableByUID bad request: tenant=%d uid=%d disable=%d err=%v", sessionUser.TenantID, req.UID, req.Disable, err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserDisableByUID start: operator_uid=%d tenant=%d target_uid=%d disable=%d",
		sessionUser.UID, sessionUser.TenantID, req.UID, req.Disable)
	if err := service.TenantUserDisabledService(
		req.UID,
		sessionUser.TenantID,
		protos.UserDisableStatus(req.Disable),
	); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserDisableByUID failed: tenant=%d target_uid=%d disable=%d err=%v",
			sessionUser.TenantID, req.UID, req.Disable, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserDisableByUID success: tenant=%d target_uid=%d disable=%d",
		sessionUser.TenantID, req.UID, req.Disable)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// UserModifyExtInfo 更新租户成员的扩展字段。
func UserModifyExtInfo(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		common.Logger.Sugar().Errorf("tenant.UserModifyExtInfo no auth: method=%s uri=%s uid=%d tenant=%d", r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.KvReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserModifyExtInfo bad request body: method=%s uri=%s tenant=%d err=%v", r.Method, r.RequestURI, sessionUser.TenantID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserModifyExtInfo start: operator_uid=%d tenant=%d target_uid=%d key=%s",
		sessionUser.UID, sessionUser.TenantID, req.ID, req.K)
	if err := service.TenantUpdateUserExt(req.ID, sessionUser.TenantID, req.K, req.V); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserModifyExtInfo failed: tenant=%d target_uid=%d key=%s err=%v",
			sessionUser.TenantID, req.ID, req.K, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserModifyExtInfo success: tenant=%d target_uid=%d key=%s", sessionUser.TenantID, req.ID, req.K)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// ModifyUserPassword 修改当前租户下指定用户密码（租户范围校验）。
func ModifyUserPassword(w http.ResponseWriter, r *http.Request) { ModifyPWDByUIDWithTenant(w, r, true) }

// ModifyPWDByUIDWithTenant 按 UID 修改密码，可选择是否校验租户上下文。
func ModifyPWDByUIDWithTenant(w http.ResponseWriter, r *http.Request, withTenant bool) {
	sessionUser := core.GetSessionUser(r)
	if withTenant && (sessionUser.UID <= 0 || sessionUser.TenantID <= 0) {
		common.Logger.Sugar().Errorf("tenant.ModifyPWDByUIDWithTenant no auth: method=%s uri=%s uid=%d tenant=%d withTenant=%t",
			r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID, withTenant)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req map[string]interface{}
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		common.Logger.Sugar().Errorf("tenant.ModifyPWDByUIDWithTenant bad request body: method=%s uri=%s tenant=%d err=%v",
			r.Method, r.RequestURI, sessionUser.TenantID, err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	uid, ok := req["uid"].(float64)
	pwd, ok2 := req["pwd"].(string)
	pwd = strings.TrimSpace(pwd)
	if !ok || !ok2 || uint64(uid) <= 0 || len(pwd) < 4 || len(pwd) > 16 {
		common.Logger.Sugar().Errorf("tenant.ModifyPWDByUIDWithTenant bad params: tenant=%d uid_ok=%t pwd_ok=%t uid=%v pwd_len=%d",
			sessionUser.TenantID, ok, ok2, req["uid"], len(pwd))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	tid := uint64(0)
	if withTenant {
		tid = sessionUser.TenantID
	}
	common.Logger.Sugar().Infof("tenant.ModifyPWDByUIDWithTenant start: operator_uid=%d tenant=%d target_uid=%d withTenant=%t",
		sessionUser.UID, sessionUser.TenantID, uint64(uid), withTenant)
	if _, err := service.SetUserPWD(uint64(uid), tid, pwd); err != nil {
		common.Logger.Sugar().Errorf("tenant.ModifyPWDByUIDWithTenant failed: tenant=%d target_uid=%d withTenant=%t err=%v",
			sessionUser.TenantID, uint64(uid), withTenant, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	common.Logger.Sugar().Infof("tenant.ModifyPWDByUIDWithTenant success: tenant=%d target_uid=%d withTenant=%t",
		sessionUser.TenantID, uint64(uid), withTenant)
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

// UserSetDepartment 为当前租户成员设置部门列表。
func UserSetDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		common.Logger.Sugar().Errorf("tenant.UserSetDepartment no auth: method=%s uri=%s uid=%d tenant=%d", r.Method, r.RequestURI, sessionUser.UID, sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}
	var req protos.SetDepartmentReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil || req.UID <= 0 {
		common.Logger.Sugar().Errorf("tenant.UserSetDepartment bad request: tenant=%d uid=%d depIds=%v err=%v", sessionUser.TenantID, req.UID, req.DepIds, err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserSetDepartment start: operator_uid=%d tenant=%d target_uid=%d depIds=%v",
		sessionUser.UID, sessionUser.TenantID, req.UID, req.DepIds)
	if err := service.TenantUserSetDepartment(req.UID, sessionUser.TenantID, req.DepIds); err != nil {
		common.Logger.Sugar().Errorf("tenant.UserSetDepartment failed: tenant=%d target_uid=%d depIds=%v err=%v",
			sessionUser.TenantID, req.UID, req.DepIds, err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	common.Logger.Sugar().Infof("tenant.UserSetDepartment success: tenant=%d target_uid=%d depIds=%v",
		sessionUser.TenantID, req.UID, req.DepIds)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

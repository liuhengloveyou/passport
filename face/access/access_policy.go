// access_policy.go 提供策略管理接口：策略增删与查询。
package access

import (
	"net/http"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"go.uber.org/zap"
)

// AddPolicyToRole 为角色添加访问策略。
func AddPolicyToRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	req := &protos.PolicyReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.AddPolicyToRole(sessionUser.TenantID, req.Role, req.Obj, req.Act); err != nil {
		core.Logger().Error("AddPolicyToRole ERR: ", zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// RemovePolicyFromRole 移除角色访问策略。
func RemovePolicyFromRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	req := &protos.PolicyReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := accessctl.RemovePolicyFromRole(sessionUser.TenantID, req.Role, req.Obj, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// GetPolicy 按角色集合查询策略列表。
func GetPolicy(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	req := strings.Split(r.FormValue("roles"), ",")
	if len(req) > 10 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	polices := accessctl.GetFilteredPolicy(sessionUser.TenantID, req)
	var policesNoDomain []protos.Policy
	if len(polices) > 0 {
		policesNoDomain = make([]protos.Policy, len(polices))
		for i := range polices {
			policesNoDomain[i] = protos.Policy{Role: polices[i][0], Obj: polices[i][2], Act: polices[i][3]}
		}
	}
	gocommon.HttpErr(w, http.StatusOK, 0, policesNoDomain)
}

// GetPolicyForUser 查询当前用户生效的策略列表。
func GetPolicyForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		return
	}
	roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID)
	if len(roles) == 0 {
		gocommon.HttpErr(w, http.StatusOK, 0, nil)
		return
	}
	policys := accessctl.GetFilteredPolicy(sessionUser.TenantID, roles)
	gocommon.HttpErr(w, http.StatusOK, 0, policys)
}

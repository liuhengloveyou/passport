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

// policyRuleToDTO 将 Casbin `p` 规则字段序列转为 API 使用的 protos.Policy（不暴露域，域已在存储层按租户隔离）。
// 支持：
// - Passport 默认 RBAC with domains：sub, dom, obj, act
// - 若底层返回将 ptype 拼入切片：p, sub, dom, obj, act（或 p, sub, obj, act）
func policyRuleToDTO(rule []string) (protos.Policy, bool) {
	if len(rule) < 4 {
		return protos.Policy{}, false
	}
	a := make([]string, len(rule))
	for i, s := range rule {
		a[i] = strings.TrimSpace(s)
	}
	head := strings.ToLower(a[0])
	if head == "p" && len(a) >= 5 {
		if a[1] == "" || a[3] == "" || a[4] == "" {
			return protos.Policy{}, false
		}
		return protos.Policy{Role: a[1], Obj: a[3], Act: a[4]}, true
	}
	if head == "p" && len(a) >= 4 {
		if a[1] == "" || a[2] == "" || a[3] == "" {
			return protos.Policy{}, false
		}
		return protos.Policy{Role: a[1], Obj: a[2], Act: a[3]}, true
	}
	if a[0] == "" || a[2] == "" || a[3] == "" {
		return protos.Policy{}, false
	}
	return protos.Policy{Role: a[0], Obj: a[2], Act: a[3]}, true
}

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
	policesNoDomain := make([]protos.Policy, 0, len(polices))
	for _, row := range polices {
		if p, ok := policyRuleToDTO(row); ok {
			policesNoDomain = append(policesNoDomain, p)
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
	out := make([]protos.Policy, 0, len(policys))
	for _, row := range policys {
		if p, ok := policyRuleToDTO(row); ok {
			out = append(out, p)
		}
	}
	gocommon.HttpErr(w, http.StatusOK, 0, out)
}

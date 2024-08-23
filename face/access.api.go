package face

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"
)

func AddRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("AddRoleForUser TenantID ERR")
		return
	}

	req := &protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.UID < 0 {
		logger.Sugar().Errorf("AddRoleForUser param ERR: %v\n", req)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if strings.TrimSpace(req.RoleValue) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, strings.TrimSpace(req.RoleValue)); err != nil {
		logger.Sugar().Errorf("AddRoleForUser AddRoleForUserInDomain ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Infof("AddRoleForUser OK: %#v\n", req)

	return
}

func RemoveRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("RemoveRoleForUser TenantID ERR")
		return
	}

	req := &protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		logger.Sugar().Errorf("RemoveRoleForUser param err.")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.UID < 0 {
		logger.Sugar().Errorf("RemoveRoleForUser param ERR: %v\n", req)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if strings.TrimSpace(req.RoleValue) == "" {
		logger.Sugar().Errorf("RemoveRoleForUser param err.")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, req.RoleValue); err != nil {
		logger.Sugar().Errorf("DeleteRoleForUserInDomain ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Infof("DeleteRoleForUser OK: %#v\n", req)

	return
}

func updateRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("updateRoleForUser TenantID ERR")
		return
	}

	req := &protos.RoleReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		logger.Sugar().Errorf("updateRoleForUser param err.")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.UID < 0 {
		logger.Sugar().Errorf("updateRoleForUser param ERR: %v\n", req)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if strings.TrimSpace(req.RoleValue) == "" || strings.TrimSpace(req.NewRoleValue) == "" {
		logger.Sugar().Errorf("updateRoleForUser param err.")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, strings.TrimSpace(req.RoleValue)); err != nil {
		logger.Sugar().Errorf("updateRoleForUser DeleteRoleForUserInDomain ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, strings.TrimSpace(req.NewRoleValue)); err != nil {
		logger.Sugar().Errorf("updateRoleForUser AddRoleForUserInDomain ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Infof("DeleteRoleForUser OK: %#v\n", req)

	return
}

func GetRolesForMe(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("GetUsersForRole session ERR")
		return
	}

	logger.Sugar().Infof("GetRolesForMe: %d\n", sessionUser.UID)

	roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID)
	if len(roles) <= 0 {
		gocommon.HttpErr(w, http.StatusOK, 0, roles)
		logger.Sugar().Info("GetRolesForMe nil\n")
		return
	}
	logger.Sugar().Infof("GetRolesForMe roles: %d %v\n", sessionUser.UID, roles)

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
	logger.Sugar().Infof("GetUsersForRole OK: %#v\n", rst)

	return
}

func GetRolesForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("GetUsersForRole session ERR")
		return
	}

	iuid, _ := strconv.ParseUint(r.FormValue("uid"), 10, 64)
	if iuid <= 0 {
		logger.Sugar().Error("GetRolesForUser param ERR: ", r.FormValue("uid"))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	logger.Sugar().Infof("GetRolesForUser: %d\n", iuid)

	roles := accessctl.GetRoleForUserInDomain(iuid, sessionUser.TenantID)
	if len(roles) <= 0 {
		gocommon.HttpErr(w, http.StatusOK, 0, roles)
		logger.Sugar().Info("GetUsersForRole nil\n")
		return
	}
	logger.Sugar().Infof("GetRolesForUser roles: %d %v\n", iuid, roles)

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
	logger.Sugar().Infof("GetUsersForRole OK: %#v\n", rst)

	return
}

func GetUsersForRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("GetUsersForRole session ERR")
		return
	}

	roleName := r.FormValue("role")
	if roleName == "" || len(roleName) > 100 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	roles := accessctl.GetUsersForRoleInDomain(roleName, sessionUser.TenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, roles)
	logger.Sugar().Infof("AddPolicy OK: %#v\n", roles)

	return
}

func AddPolicyToRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("RemoveRoleForUser TenantID ERR")
		return
	}

	req := &protos.PolicyReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	logger.Sugar().Infoln("AddPolicyToRole ", sessionUser.TenantID, req.Role, req.Obj, req.Act)
	if err := accessctl.AddPolicyToRole(sessionUser.TenantID, req.Role, req.Obj, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Infof("AddPolicy OK: %#v\n", req)

	return
}

func RemovePolicyFromRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("RemoveRoleForUser TenantID ERR")
		return
	}

	req := &protos.PolicyReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.RemovePolicyFromRole(sessionUser.TenantID, req.Role, req.Obj, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Infof("RemovePolicy OK: %#v\n", req)

	return
}

func GetPolicy(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("GetPolicy TenantID ERR")
		return
	}

	r.ParseForm()

	req := strings.Split(r.FormValue("roles"), ",")
	logger.Sugar().Info("GetPolicy param: ", req)
	if len(req) > 10 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("GetPolicy param ERR: ", req)
		return
	}

	polices := accessctl.GetFilteredPolicy(sessionUser.TenantID, req)
	var policesNoDomain []protos.Policy
	if len(polices) > 0 {
		policesNoDomain = make([]protos.Policy, len(polices))
		for i := 0; i < len(polices); i++ {
			policesNoDomain[i] = protos.Policy{
				Role: polices[i][0],
				Obj:  polices[i][2],
				Act:  polices[i][3],
			}
		}
	}

	gocommon.HttpErr(w, http.StatusOK, 0, policesNoDomain)
	logger.Sugar().Infof("GetPolicy OK: %#v\n", polices)
}

func GetPolicyForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("GetPolicyForUser TenantID ERR")
		return
	}

	roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID)
	if len(roles) == 0 {
		gocommon.HttpErr(w, http.StatusOK, 0, nil)
		logger.Sugar().Error("GetPolicyForUser TenantID ERR")
		return

	}

	policys := accessctl.GetFilteredPolicy(sessionUser.TenantID, roles)

	gocommon.HttpErr(w, http.StatusOK, 0, policys)
	logger.Sugar().Infof("GetPolicyForUser OK: %#v\n", policys)
}

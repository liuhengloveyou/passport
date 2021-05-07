package face

import (
	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sessions"
	"net/http"
	"strconv"
	"strings"
)

func AddRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("AddRoleForUser TenantID ERR")
		return
	}

	req := &protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.AddRoleForUserInDomain(req.UID, sessionUser.TenantID, req.RoleValue); err != nil {
		logger.Errorf("AddRoleForUser AddRoleForUserInDomain ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("AddRoleForUser OK: %#v\n", req)

	return
}

func RemoveRoleForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("RemoveRoleForUser TenantID ERR")
		return
	}

	req := &protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.DeleteRoleForUserInDomain(req.UID, sessionUser.TenantID, req.RoleValue); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("DeleteRoleForUser OK: %#v\n", req)

	return
}

func GetRolesForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("GetUsersForRole session ERR")
		return
	}

	uid := r.FormValue("uid")
	if uid == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	iuid,_ := strconv.ParseUint(uid, 10, 64)

	roles := accessctl.GetRoleForUserInDomain(iuid, sessionUser.TenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, roles)
	logger.Infof("AddPolicy OK: %#v\n", roles)

	return
}

func GetUsersForRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("GetUsersForRole session ERR")
		return
	}

	roleName := r.FormValue("role")
	if roleName == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	roles := accessctl.GetUsersForRoleInDomain(roleName, sessionUser.TenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, roles)
	logger.Infof("AddPolicy OK: %#v\n", roles)

	return
}

func AddPolicyToRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("RemoveRoleForUser TenantID ERR")
		return
	}

	req := &protos.PolicyReq{}
	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.AddPolicyToRole(sessionUser.TenantID, req.Role, req.Obj, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("AddPolicy OK: %#v\n", req)

	return
}

func RemovePolicyFromRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("RemoveRoleForUser TenantID ERR")
		return
	}

	req := &protos.PolicyReq{}
	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.RemovePolicyFromRole(sessionUser.TenantID, req.Role, req.Obj, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("RemovePolicy OK: %#v\n", req)

	return
}

func GetPolicy(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("GetPolicy TenantID ERR")
		return
	}

	r.ParseForm()

	req := strings.Split(r.FormValue("roles"), ",")
	logger.Info("GetPolicy param: ", req)
	if len(req) > 10 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("GetPolicy param ERR: ", req)
		return
	}

	policys := accessctl.GetFilteredPolicy(sessionUser.TenantID, req)
	gocommon.HttpErr(w, http.StatusOK, 0, policys)
	logger.Infof("GetPolicy OK: %#v\n", policys)

	return
}


func GetPolicyForUser(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("GetPolicyForUser TenantID ERR")
		return
	}

	roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID)

	policys := accessctl.GetFilteredPolicy(sessionUser.TenantID, roles)
	gocommon.HttpErr(w, http.StatusOK, 0, policys)
	logger.Infof("GetPolicyForUser OK: %#v\n", policys)

	return
}

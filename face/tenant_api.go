package face

import (
	"net/http"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func TenantAdd(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID > 0 {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrTenantLimit)
		return
	}
	req := &protos.Tenant{
		UID: sessionUser.UID,
	}

	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("tenantAdd param ERR: ", err)
		return
	}
	logger.Infof("tenantAdd body: %#v\n", req)

	if req.TenantName == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNameNull)
		logger.Error("tenantAdd param ERR: ", req)
		return
	}
	if req.TenantType == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantTypeNull)
		logger.Error("tenantAdd param ERR: ", req)
		return
	}

	uid, err := service.TenantAdd(req)
	if err != nil {
		logger.Error("tenantAdd service ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Info("user add ok:", uid)
	gocommon.HttpErr(w, http.StatusOK, 0, uid)

	return
}

func GetRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("GetRole TenantID ERR")
		return
	}

	roles := service.TenantGetRole(sessionUser.TenantID)

	gocommon.HttpErr(w, http.StatusOK, 0, roles)

	return
}

func RoleAdd(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	req := make(map[string]string, 1)
	if err := readJsonBodyFromRequest(r, &req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("RoleAdd param ERR: ", err)
		return
	}
	if _, ok := req["role"]; !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("RoleAdd param ERR: ", req)
		return
	}
	logger.Infof("RoleAdd body: %#v\n", req)

	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("RoleAdd session ERR: ", sessionUser)
		return
	}

	if err := service.TenantAddRole(sessionUser.TenantID, req["role"]); err != nil {
		logger.Error("TenantAddRole service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	return
}
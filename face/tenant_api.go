package face

import (
	"github.com/liuhengloveyou/passport/dao"
	"net/http"
	"strings"

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

func TenantUserAdd(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("TenantUserAdd TenantID ERR")
		return
	}

	req := &protos.Tenant{} // 只用一个UID字段
	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserAdd param ERR: ", err)
		return
	}
	logger.Infof("tenantAdd body: %#v\n", req)

	row, e := dao.UserUpdateTenantID(req.UID, sessionUser.TenantID, 0)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		logger.Error("TenantUserAdd db ERR: ", e)
		return
	}
	if row != 1 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantLimit)
		logger.Error("TenantUserAdd ERR: ", row, e)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)

	return
}

func TenantUserDel(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("TenantUserDel session ERR")
		return
	}

	req := &protos.Tenant{} // 只用一个UID字段
	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserDel param ERR: ", err)
		return
	}
	logger.Infof("TenantUserDel body: %#v\n", req)

	_, e := service.TenantUserDel(req.UID, sessionUser.TenantID)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)

	return
}
func TenantUserGet(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("TenantUserAdd TenantID ERR")
		return
	}

	rr, e := dao.UserSelectByTenantID(sessionUser.TenantID)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		logger.Error("TenantUserAdd db ERR: ", e)
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rr)

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

	req := protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, &req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("RoleAdd param ERR: ", err)
		return
	}
	if "" == req.RoleTitle || "" == req.RoleValue {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("RoleAdd param ERR: ", req)
		return
	}
	logger.Infof("RoleAdd body: %#v\n", req)

	if err := service.TenantAddRole(sessionUser.TenantID, req); err != nil {
		logger.Error("TenantAddRole service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	return
}

func UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateConfiguration param ERR: ", err)
		return
	}
	k, ok := req["k"].(string)
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateConfiguration param k nil")
		return
	}
	if len(strings.TrimSpace(k)) == 0 || len(strings.TrimSpace(k)) > 64 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateConfiguration param k nil")
		return
	}

	logger.Infof("UpdateConfiguration : %v\n", req)

	if err := service.TenantUpdateConfiguration(sessionUser.TenantID, k, req["v"]); err != nil {
		logger.Error("UpdateConfiguration service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	return
}

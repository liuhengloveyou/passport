package face

import (
	"net/http"
	"strconv"
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

	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("tenantAdd param ERR: ", err)
		return
	}
	logger.Infof("tenantAdd body: %#v\n", req)

	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantType = strings.TrimSpace(req.TenantType)
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

	req := &protos.UserReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserAdd param ERR: ", err)
		return
	}
	logger.Infof("TenantUserAdd body: %#v\n", req)
	if len(req.Roles) > 10 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserAdd param roles ERR")
		return

	}

	if req.UID == 0 {
		nuid, e := service.AddUserService(req)
		if e != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, e)
			logger.Error("TenantUserAdd AddUserService ERR: ", e)
			return
		}

		req.UID = nuid
	}
	if err := service.TenantUserAdd(req.UID, sessionUser.TenantID, req.Roles, req.Disable); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		logger.Error("TenantUserAdd service ERR: ", err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, req.UID)

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
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserDel param ERR: ", err)
		return
	}
	logger.Infof("TenantUserDel body: %#v\n", req)

	_, e := service.TenantUserDel(req.UID, sessionUser.TenantID)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		logger.Error("TenantUserDel service ERR: ", e)
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)

	return
}

func TenantUserDisableByUID(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("TenantID ERR")
		return
	}

	var req protos.DisableUserReq
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserDisableByUID param ERR: ", err)
		return
	}

	if req.UID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserDisableByUID param ERR: ", req)
		return
	}

	if req.Disable != 0 && req.Disable != 1 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantUserDisableByUID param ERR: ", req)
		return
	}

	logger.Infof("TenantUserDisableByUID %v\n", req)

	if err := service.TenantUserDisabledService(req.UID, sessionUser.TenantID, int8(req.Disable)); err != nil {
		logger.Errorf("TenantUserDisableByUID %v %s\n", req, err.Error())
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	return
}

func TenantUserGet(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("TenantUserGet TenantID ERR")
		return
	}

	var err error

	r.ParseForm()
	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	nickname := strings.TrimSpace(r.FormValue("nickname"))
	uidStr := r.FormValue("uids")

	var uids []uint64
	if uidStr != "" {
		uidss := strings.Split(uidStr, ",")
		if len(uidss) <= 0 {
			gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
			logger.Error("TenantUserGet uids ERR: ", uidStr)
			return
		}
		uids = make([]uint64, len(uidss))

		for i, ouids := range uidss {
			if uids[i], err = strconv.ParseUint(ouids, 10, 64); err != nil {
				gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
				logger.Error("TenantUserGet uids ERR: ", uidStr, uidStr, uidss)
				return
			}
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
		logger.Error("TenantUserGet db ERR: ", e)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rr)

	return
}

func TenantGetRole(w http.ResponseWriter, r *http.Request) {
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

func TenantRoleAdd(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	req := protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("RoleAdd param ERR: ", err)
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

func TenantRoleDel(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	req := protos.RoleStruct{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("TenantRoleDel param ERR: ", err)
		return
	}
	logger.Infof("TenantRoleDel body: %#v\n", req)

	if err := service.TenantDelRole(sessionUser.TenantID, req); err != nil {
		logger.Error("TenantRoleDel service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	return
}

func LoadConfiguration(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	k := strings.TrimSpace(r.FormValue("k"))

	confMap, err := service.TenantLoadConfiguration(sessionUser.TenantID, k)
	if err != nil {
		logger.Error("LoadConfiguration service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Debug("LoadConfiguration OK:", sessionUser.UID, sessionUser.TenantID, confMap)
	gocommon.HttpErr(w, http.StatusOK, 0, confMap)
}

func UpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateConfiguration param ERR: ", err)
		return
	}

	logger.Infof("UpdateConfiguration: %v\n", req)

	if err := service.TenantUpdateConfiguration(sessionUser.TenantID, req); err != nil {
		logger.Error("UpdateConfiguration service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func tenantModifyPWDByUID(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("modifyPWDByID param ERR: ", err)
		return
	}

	uid, ok := req["uid"].(float64)
	if !ok || uint64(uid) <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("modifyPWDByUID param ERR: ", req)
		return
	}

	pwd, ok := req["pwd"].(string)
	pwd = strings.TrimSpace(pwd)
	if len(pwd) < 4 || len(pwd) > 16 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("modifyPWDByUID param ERR: ", req)
		return
	}

	logger.Infof("modifyPWDByUID %v %s\n", uid, pwd)

	if _, err := service.SetUserPWD(uint64(uid), sessionUser.TenantID, pwd); err != nil {
		logger.Errorf("modifyPWDByUID %v %s %s\n", uid, pwd, err.Error())
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
	return
}

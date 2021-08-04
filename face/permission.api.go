package face

import (
	"github.com/liuhengloveyou/passport/sessions"
	"net/http"
	"strconv"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"

	gocommon "github.com/liuhengloveyou/go-common"
)

func PermissionCreate(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("PermissionCreate TenantID ERR")
		return
	}

	var req protos.PermissionStruct
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		common.Logger.Sugar().Errorf("PermissionCreate param ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	common.Logger.Sugar().Infof("PermCreate: %#v\n", req)
	req.TenantID =sessionUser.TenantID

	id, err := service.PermissionCreate(&req)
	if err != nil {
		common.Logger.Sugar().Errorf("PermService service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, id)
}

func PermissionDelete(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("PermissionDelete TenantID ERR")
		return
	}

	r.ParseForm()
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	common.Logger.Sugar().Infof("deletePermOne id: %v %v\n", id, sessionUser.TenantID)

	err := service.PermissionDelete(id, sessionUser.TenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("deletePermOne service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func PermissionList(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Error("PermissionList TenantID ERR")
		return
	}

	r.ParseForm()
	domain := r.FormValue("domain")
	common.Logger.Sugar().Infof("PermissionList domain: %v %v\n", domain, sessionUser.TenantID )

	list, err := service.PermissionList(sessionUser.TenantID, domain)
	if err != nil {
		common.Logger.Sugar().Errorf("PermissionList service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, list)
}

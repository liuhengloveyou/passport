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

func getMyInfo(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Error("getMyInfo session ERR")
		return
	}
	logger.Infof("getMyInfo: %d\n", sessionUser.UID)

	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Error("GetMyInfo ERR uid nil.")
		return
	}


	rst, err := service.GetUserInfoService(sessionUser.UID, sessionUser.TenantID)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Error("GetMyInfo ERR: " + err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rst)
	logger.Infof("GetMyInfo OK: %#v %#v\n", sessionUser.UID, rst)

	return
}


func getInfoByUID(w http.ResponseWriter, r *http.Request) {
	sessioinUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessioinUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Error("getInfoByUID ERR uid nil.")
		return
	}

	uid := strings.TrimSpace(r.FormValue("uid"))
	if uid == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("getInfoByUID ERR uid nil.")
		return
	}

	iuid, _ := strconv.ParseUint(uid, 10, 64)
	userInfo, err := service.GetUserInfoService(iuid, 0)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Error("getInfoByUID ERR: " + err.Error())
		return
	}

	if sessioinUser.TenantID > 0 || userInfo.TenantID > 0 {
		if sessioinUser.TenantID != userInfo.TenantID {
			userInfo.Tenant = nil
			userInfo.TenantID = 0
		}
	}

	gocommon.HttpErr(w, http.StatusOK, 0, userInfo)
	logger.Infof("getInfoByUID OK: %#v %#v\n", uid, userInfo)

	return
}

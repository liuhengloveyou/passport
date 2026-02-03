package face

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/service"

	gocommon "github.com/liuhengloveyou/go-common"
)

func getMyInfo(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value("session") == nil {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Sugar().Error("getMyInfo ERR session nil.")
		return
	}

	sessionUser := GetSessionUser(r)
	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Sugar().Error("GetSessionUser failed")
		return
	}

	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
		logger.Sugar().Error("getMyInfo session ERR")
		return
	}
	logger.Sugar().Infof("getMyInfo: %d\n", sessionUser.UID)

	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Sugar().Error("GetMyInfo ERR uid nil.")
		return
	}

	rst, err := service.GetUserInfoService(sessionUser.UID, sessionUser.TenantID)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Sugar().Error("GetMyInfo ERR: " + err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rst)
	logger.Sugar().Infof("GetMyInfo OK: %#v %#v\n", sessionUser.UID, rst)
}

func getInfoByUID(w http.ResponseWriter, r *http.Request) {
	sessionUser := GetSessionUser(r)
	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Sugar().Error("GetSessionUser failed")
		return
	}

	uid := strings.TrimSpace(r.FormValue("uid"))
	if uid == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("getInfoByUID ERR uid nil.")
		return
	}

	iuid, _ := strconv.ParseUint(uid, 10, 64)
	userInfo, err := service.GetUserInfoService(iuid, 0)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Sugar().Error("getInfoByUID ERR: " + err.Error())
		return
	}

	if sessionUser.TenantID > 0 || userInfo.TenantID > 0 {
		if sessionUser.TenantID != userInfo.TenantID {
			userInfo.Tenant = nil
			userInfo.TenantID = 0
		}
	}

	gocommon.HttpErr(w, http.StatusOK, 0, userInfo)
	logger.Sugar().Infof("getInfoByUID OK: %#v %#v\n", uid, userInfo)

	return
}

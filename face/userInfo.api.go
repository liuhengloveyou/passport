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
	var uid uint64
	if r.Context().Value("session") != nil {
		uid = r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User).UID
	}
	if uid <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		logger.Error("GetMyInfo ERR uid nil.")
		return
	}
	logger.Infof("getMyInfo: %v", uid)

	rst, err := service.GetUserInfoService(uid)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Error("GetMyInfo ERR: " + err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rst)
	logger.Infof("GetMyInfo OK: %#v %#v\n", uid, rst)

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
	userInfo, err := service.GetUserInfoService(iuid)
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

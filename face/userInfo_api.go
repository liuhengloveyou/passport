package face

import (
	"net/http"

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

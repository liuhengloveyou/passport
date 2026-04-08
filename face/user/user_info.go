package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserInfo 查询当前登录用户详情。
func UserInfo(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		return
	}
	rst, err := service.GetUserInfoService(sessionUser.UID, sessionUser.TenantID)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rst)
}

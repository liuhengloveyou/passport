package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/face/core"
	"github.com/liuhengloveyou/passport/v4/service"
)

// UserInfo 查询当前登录用户详情。
func UserInfo(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		return
	}
	orgID := core.ParseOrgID(r)
	if service.UserInOrg(sessionUser.UID, sessionUser.TenantID, orgID) != nil {
		orgID = 0
	}
	rst, err := service.GetUserInfoService(sessionUser.UID, sessionUser.TenantID, orgID)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, rst)
}

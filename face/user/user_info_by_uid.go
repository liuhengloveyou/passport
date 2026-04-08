package user

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserInfoByUID 按 UID 查询用户详情，并按租户关系裁剪敏感租户信息。
func UserInfoByUID(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	if sessionUser.UID <= 0 {
		gocommon.HttpErr(w, http.StatusUnauthorized, -1, "")
		return
	}
	uid := strings.TrimSpace(r.FormValue("uid"))
	iuid, _ := strconv.ParseUint(uid, 10, 64)
	userInfo, err := service.GetUserInfoService(iuid, 0)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}
	if sessionUser.TenantID > 0 || userInfo.TenantID > 0 {
		if sessionUser.TenantID != userInfo.TenantID {
			userInfo.Tenant = nil
			userInfo.TenantID = 0
		}
	}
	gocommon.HttpErr(w, http.StatusOK, 0, userInfo)
}

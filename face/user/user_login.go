package user

import (
	"net/http"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserLogin 用户登录并写入会话，支持按请求头返回 token 模式。
func UserLogin(w http.ResponseWriter, r *http.Request) {
	useCookie := strings.ToLower(r.Header.Get("USE-COOKIE")) != "false"
	req := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	one, err := service.UserLogin(req)
	if err != nil || one == nil {
		if err != nil {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
		} else {
			gocommon.HttpErr(w, http.StatusOK, -1, "用户不存在")
		}
		return
	}
	normalizeUserExt(one)
	r.Header.Del("Cookie")
	session, err := core.SessionStore().New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		return
	}
	sessionUser := &protos.User{UID: one.UID, TenantID: one.TenantID, Cellphone: one.Cellphone, Email: one.Email, Nickname: one.Nickname, AvatarURL: one.AvatarURL, CreateTime: one.CreateTime, UpdateTime: one.UpdateTime, LoginTime: one.LoginTime}
	session.Values[common.SessUserInfoKey] = sessionUser
	session.Options.MaxAge = common.ServConfig.SessionExpire
	session.Options.Domain = common.ServConfig.Domain
	session.Options.Secure = false
	session.Options.SameSite = http.SameSiteDefaultMode
	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		return
	}
	if !useCookie {
		one.SetExt("TOKEN", strings.Split(w.Header().Get("Set-Cookie"), ";")[0][len(common.ServConfig.SessionKey)+1:])
		w.Header().Del("Set-Cookie")
	}
	gocommon.HttpErr(w, http.StatusOK, 0, one)
}

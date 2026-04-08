// http_api.go 提供微信登录与绑定相关接口。
package wx

import (
	"net/http"
	"strings"

	"github.com/liuhengloveyou/go-errors"
	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	"github.com/liuhengloveyou/passport/v3/weixin"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4/zero"
)

// MpAuth 处理微信公众号 OAuth 回调并写入会话。
func MpAuth(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := strings.TrimSpace(r.FormValue("code"))
	state := strings.TrimSpace(r.FormValue("state"))
	if state == "" || code == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	accessToken, err := weixin.GetAccessToken(common.ServConfig.AppID, common.ServConfig.AppSecret, code)
	if err != nil || accessToken == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrWxService)
		return
	}
	wxUserInfo, err := weixin.GetUserInfo(accessToken.AccessToken, accessToken.OpenId)
	if err != nil || wxUserInfo == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrWxService)
		return
	}
	loginReq := &protos.UserReq{WxOpenId: accessToken.OpenId}
	one, err := service.UserLoginByWeixin(loginReq)
	if err != nil {
		myErr, ok := err.(*errors.Error)
		if !ok || myErr.Code != common.ErrLogin.Code {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}
	if one == nil {
		nickname := zero.StringFrom(wxUserInfo.Nickname)
		avatarurl := zero.StringFrom(wxUserInfo.Headimgurl)
		sex := zero.IntFrom(wxUserInfo.Sex)
		wxOpenId := zero.StringFrom(accessToken.OpenId)
		one = &protos.User{UID: protos.WX_MP_AUTH_UID, WxOpenId: &wxOpenId, Nickname: &nickname, AvatarURL: &avatarurl, Gender: &sex}
	}
	SetWxUserToSession(w, r, one)
	http.Redirect(w, r, state, http.StatusTemporaryRedirect)
}

// WxMpBindCellphone 将公众号登录态绑定到手机号账号。
func WxMpBindCellphone(w http.ResponseWriter, r *http.Request) {
	sess, auth := core.AuthFilter(r)
	if !auth {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}
	userReq := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, userReq, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	sessUserInfo := sess.Values[common.SessUserInfoKey].(protos.User)
	if len(userReq.Cellphone) == 0 || len(userReq.SmsCode) == 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrParam)
		return
	}
	if sessUserInfo.UID != protos.WX_MP_AUTH_UID || sessUserInfo.WxOpenId == nil || len(sessUserInfo.WxOpenId.String) == 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrSession)
		return
	}
	if sessUserInfo.Cellphone != nil && len(sessUserInfo.Cellphone.String) > 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrSession)
		return
	}
	if _, err := service.UpdateUserWxOpenIdByCellphone(userReq.Cellphone, sessUserInfo.WxOpenId.String, userReq.SmsCode); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, "成功")
}

// WxMiniAppLogin 处理微信小程序登录并建立会话。
func WxMiniAppLogin(w http.ResponseWriter, r *http.Request) {
	var req weixin.WxMiniAppLoginReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	wxSession, err := weixin.WxMiniAppLogin(req.Code, common.ServConfig.MiniAppID, common.ServConfig.MiniAppSecret)
	if err != nil || wxSession == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	loginReq := &protos.UserReq{WxOpenId: wxSession.OpenId}
	one, err := service.UserLoginByWeixin(loginReq)
	if err != nil {
		myErr, ok := err.(*errors.Error)
		if !ok || myErr.Code != common.ErrLogin.Code {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}
	if one == nil {
		wxOpenId := zero.StringFrom(wxSession.OpenId)
		one = &protos.User{UID: protos.WX_MP_AUTH_UID, WxOpenId: &wxOpenId}
	}
	SetWxUserToSession(w, r, one)
	gocommon.HttpErr(w, http.StatusOK, 0, "成功")
}

// WxMiniAppUserInfoUpdate 更新小程序用户信息（占位实现）。
func WxMiniAppUserInfoUpdate(w http.ResponseWriter, r *http.Request) {
	sess, auth := core.AuthFilter(r)
	if !auth {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}
	_ = sess.Values[common.SessUserInfoKey].(protos.User)
	var req weixin.WxMiniAppUserInfoUpdateReq
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

// SetWxUserToSession 将微信用户写入 session。
func SetWxUserToSession(w http.ResponseWriter, r *http.Request, userInfo *protos.User) {
	if userInfo == nil {
		return
	}
	r.Header.Del("Cookie")
	session, err := core.SessionStore().New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		return
	}
	session.IsNew = true
	session.Options.MaxAge = 0
	session.Values[common.SessUserInfoKey] = userInfo
	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		return
	}
	core.Logger().Info("wx login ok: ", zap.Any("session", session.Values[common.SessUserInfoKey]))
}

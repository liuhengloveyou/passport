// http_api.go 提供微信登录与绑定相关接口。
package wx

import (
	"net/http"
	"net/url"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/face/core"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/service"
	"github.com/liuhengloveyou/passport/v4/weixin"
	"go.uber.org/zap"
)

const (
	pathWxLogin     = "/usercenter/wx/login"
	pathWxOAuthCB   = "/usercenter/wx/oauth"
	pathWxMiniLogin = "/usercenter/wx/mini/login"
)

// WxLogin 公众号网页登录入口。
//
// 已登录：按 success_url / bind_url 分流。
// 未登录：跳转微信授权，回调 /usercenter/wx/oauth。
//
// query:
//   - success_url：登录成功业务回跳（H5 点单可只传此项）
//   - bind_url：有微信会话但未绑手机时回跳（管理端可选）
//   - redirect_uri：微信回调地址（可选；默认当前 Host + /usercenter/wx/oauth）
//   - scope：snsapi_base（默认）| snsapi_userinfo
func WxLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	successURL := strings.TrimSpace(r.FormValue("success_url"))
	bindURL := strings.TrimSpace(r.FormValue("bind_url"))
	if successURL != "" && !isSafeReturnURL(successURL) {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if bindURL != "" && !isSafeReturnURL(bindURL) {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	sess := core.GetSessionUser(r)
	hasWx := sess.WxOpenId != nil && strings.TrimSpace(sess.WxOpenId.String) != ""
	hasPhone := sess.Cellphone != nil && len(sess.Cellphone.String) == 11

	if protos.IsRealUserUID(sess.UID) && (hasWx || hasPhone) {
		if hasPhone {
			redirectOrHome(w, r, successURL)
			return
		}
		if bindURL != "" {
			http.Redirect(w, r, bindURL, http.StatusTemporaryRedirect)
			return
		}
		if successURL != "" {
			http.Redirect(w, r, successURL, http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// 未登录：授权后
	// - 需要绑手机：先回本登录页再分流
	// - 仅 success_url：授权后直达业务页
	afterAuth := successURL
	if bindURL != "" || afterAuth == "" {
		afterAuth = absoluteURL(r, pathWxLogin, url.Values{
			"success_url":  {successURL},
			"bind_url":     {bindURL},
			"redirect_uri": {strings.TrimSpace(r.FormValue("redirect_uri"))},
			"scope":        {strings.TrimSpace(r.FormValue("scope"))},
		})
	}
	startMpOAuth(w, r, afterAuth)
}

// WxOAuthCallback 微信网页授权回调（redirect_uri）。
// query: code、state（完整回跳 URL）。
func WxOAuthCallback(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	code := strings.TrimSpace(r.FormValue("code"))
	state := strings.TrimSpace(r.FormValue("state"))
	core.Logger().Info("WxOAuthCallback start",
		zap.Bool("has_code", code != ""),
		zap.String("state", state),
	)
	if state == "" || code == "" {
		core.Logger().Error("WxOAuthCallback fail: missing code/state",
			zap.Bool("has_code", code != ""),
			zap.Bool("has_state", state != ""),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if !isSafeReturnURL(state) {
		core.Logger().Error("WxOAuthCallback fail: unsafe state", zap.String("state", state))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	accessToken, err := weixin.GetAccessToken(common.ServConfig.AppID, common.ServConfig.AppSecret, code)
	if err != nil || accessToken == nil || accessToken.OpenId == "" {
		openid := ""
		if accessToken != nil {
			openid = accessToken.OpenId
		}
		core.Logger().Error("WxOAuthCallback fail: get access_token",
			zap.Error(err),
			zap.Bool("token_nil", accessToken == nil),
			zap.String("openid", openid),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrWxService)
		return
	}
	openid := accessToken.OpenId

	loginReq := &protos.UserReq{WxOpenId: openid}
	if wxUserInfo, uerr := weixin.GetUserInfo(accessToken.AccessToken, openid); uerr == nil && wxUserInfo != nil {
		loginReq.Nickname = wxUserInfo.Nickname
		loginReq.AvatarURL = wxUserInfo.Headimgurl
		loginReq.Gender = int32(wxUserInfo.Sex)
	} else if uerr != nil {
		core.Logger().Warn("WxOAuthCallback GetUserInfo skip", zap.String("openid", openid), zap.Error(uerr))
	}
	loginReq.Ext = protos.MapStruct{"kind": "wechat", "wechat": 1}
	one, err := service.UserLoginByOpenID(loginReq)
	if err != nil || one == nil || !protos.IsRealUserUID(one.UID) {
		uid := uint64(0)
		if one != nil {
			uid = one.UID
		}
		core.Logger().Error("WxOAuthCallback fail: login",
			zap.Error(err),
			zap.String("openid", openid),
			zap.Uint64("uid", uid),
			zap.Bool("user_nil", one == nil),
		)
		if err == nil {
			err = common.ErrLogin
		}
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	if one.Ext == nil {
		one.Ext = protos.MapStruct{}
	}
	one.SetExt("kind", "wechat")
	one.SetExt("wechat", 1)
	if !SetWxUserToSession(w, r, one) {
		core.Logger().Error("WxOAuthCallback fail: write session cookie",
			zap.String("openid", openid),
			zap.Uint64("uid", one.UID),
		)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "wxOpenid",
		Value:    openid,
		Path:     "/",
		Domain:   common.ServConfig.Domain,
		MaxAge:   common.ServConfig.SessionExpire,
		HttpOnly: false,
		SameSite: http.SameSiteDefaultMode,
	})
	nickname := ""
	if one.Nickname != nil {
		nickname = one.Nickname.String
	}
	core.Logger().Info("WxOAuthCallback ok",
		zap.Uint64("uid", one.UID),
		zap.String("openid", openid),
		zap.String("nickname", nickname),
		zap.String("return_url", state),
	)
	http.Redirect(w, r, state, http.StatusTemporaryRedirect)
}

func startMpOAuth(w http.ResponseWriter, r *http.Request, returnURL string) {
	if !isSafeReturnURL(returnURL) {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if common.ServConfig.AppID == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrWxService)
		return
	}
	scope := strings.TrimSpace(r.FormValue("scope"))
	if scope != "snsapi_userinfo" {
		scope = "snsapi_base"
	}
	callback := strings.TrimSpace(r.FormValue("redirect_uri"))
	if callback == "" {
		callback = absoluteURL(r, pathWxOAuthCB, nil)
	}
	if !isSafeReturnURL(callback) {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	redirectURL := "https://open.weixin.qq.com/connect/oauth2/authorize?appid=" +
		url.QueryEscape(common.ServConfig.AppID) +
		"&redirect_uri=" + url.QueryEscape(callback) +
		"&response_type=code&scope=" + url.QueryEscape(scope) +
		"&state=" + url.QueryEscape(returnURL) +
		"#wechat_redirect"
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func redirectOrHome(w http.ResponseWriter, r *http.Request, successURL string) {
	if successURL != "" {
		http.Redirect(w, r, successURL, http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func absoluteURL(r *http.Request, path string, q url.Values) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		return ""
	}
	u := scheme + "://" + host + path
	if q != nil {
		clean := url.Values{}
		for k, vs := range q {
			for _, v := range vs {
				if strings.TrimSpace(v) != "" {
					clean.Add(k, v)
				}
			}
		}
		if enc := clean.Encode(); enc != "" {
			u += "?" + enc
		}
	}
	return u
}

func isSafeReturnURL(u string) bool {
	if u == "" || len(u) > 512 {
		return false
	}
	parsed, err := url.Parse(u)
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return true
}

// WxMpBindCellphone 将公众号登录态绑定到手机号账号。
func WxMpBindCellphone(w http.ResponseWriter, r *http.Request) {
	sessUserInfo := core.GetSessionUser(r)
	if !protos.IsRealUserUID(sessUserInfo.UID) {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}
	if sessUserInfo.WxOpenId == nil || len(sessUserInfo.WxOpenId.String) == 0 {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}
	userReq := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, userReq, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if len(userReq.Cellphone) == 0 || len(userReq.SmsCode) == 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrParam)
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
	loginReq.Ext = protos.MapStruct{"kind": "wechat", "wechat": 1}
	one, err := service.UserLoginByOpenID(loginReq)
	if err != nil || one == nil || !protos.IsRealUserUID(one.UID) {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	if one.Ext == nil {
		one.Ext = protos.MapStruct{}
	}
	one.SetExt("kind", "wechat")
	one.SetExt("wechat", 1)
	if !SetWxUserToSession(w, r, one) {
		return
	}
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

// SetWxUserToSession 将微信/渠道用户写入 session cookie，选项与 UserLogin 一致。
// 成功返回 true；失败时已写入错误响应，返回 false。
func SetWxUserToSession(w http.ResponseWriter, r *http.Request, userInfo *protos.User) bool {
	if userInfo == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return false
	}
	r.Header.Del("Cookie")
	session, err := core.SessionStore().New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		return false
	}
	session.IsNew = true
	session.Values[common.SessUserInfoKey] = userInfo
	session.Options.MaxAge = common.ServConfig.SessionExpire
	session.Options.Domain = common.ServConfig.Domain
	session.Options.Secure = false
	session.Options.SameSite = http.SameSiteDefaultMode
	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		return false
	}
	core.Logger().Info("wx login ok: ", zap.Any("session", session.Values[common.SessUserInfoKey]))
	return true
}

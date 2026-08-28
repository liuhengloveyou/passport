package ali

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/face/core"
	passportwx "github.com/liuhengloveyou/passport/v4/face/wx"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/service"
	alipay "github.com/smartwalle/alipay/v3"
	"go.uber.org/zap"
)

const (
	pathAliOAuthCB = "/usercenter/ali/oauth"
)

var (
	aliOnce    sync.Once
	aliClient  *alipay.Client
	aliInitErr error
)

func ensureClient() error {
	aliOnce.Do(func() {
		cfg := common.ServConfig
		core.Logger().Info("alipay ensureClient init",
			zap.String("app_id", cfg.AlipayAppID),
			zap.Bool("has_private_key", cfg.AlipayPrivateKey != ""),
			zap.Bool("has_public_key", cfg.AlipayPublicKey != ""),
			zap.Bool("has_encrypt_key", cfg.AlipayEncryptKey != ""),
		)
		if cfg.AlipayAppID == "" || cfg.AlipayPrivateKey == "" {
			aliInitErr = fmt.Errorf("alipay config empty")
			core.Logger().Error("alipay ensureClient fail: config empty")
			return
		}
		c, err := alipay.New(cfg.AlipayAppID, cfg.AlipayPrivateKey, true)
		if err != nil {
			aliInitErr = err
			core.Logger().Error("alipay ensureClient fail: New", zap.Error(err))
			return
		}
		if cfg.AlipayPublicKey != "" {
			if err = c.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
				aliInitErr = err
				core.Logger().Error("alipay ensureClient fail: LoadAliPayPublicKey", zap.Error(err))
				return
			}
		}
		if cfg.AlipayEncryptKey != "" {
			if err = c.SetEncryptKey(cfg.AlipayEncryptKey); err != nil {
				aliInitErr = err
				core.Logger().Error("alipay ensureClient fail: SetEncryptKey", zap.Error(err))
				return
			}
		}
		aliClient = c
		core.Logger().Info("alipay ensureClient ok")
	})
	return aliInitErr
}

// AliLogin 支付宝网页授权登录入口（对齐微信 /usercenter/wx/login）。
//
// - 带 code / auth_code：兑换登录并 JSON 返回（兼容 JSAPI / 旧前端）
// - 带 success_url：跳转支付宝网页授权，回调 /usercenter/ali/oauth
func AliLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	code := firstNonEmpty(r.FormValue("code"), r.FormValue("auth_code"))
	successURL := strings.TrimSpace(r.FormValue("success_url"))
	core.Logger().Info("AliLogin start",
		zap.String("method", r.Method),
		zap.String("ua", r.UserAgent()),
		zap.Bool("has_code", code != ""),
		zap.Int("code_len", len(code)),
		zap.String("success_url", successURL),
		zap.String("redirect_uri", strings.TrimSpace(r.FormValue("redirect_uri"))),
		zap.String("scope", strings.TrimSpace(r.FormValue("scope"))),
		zap.String("raw_query", r.URL.RawQuery),
	)
	if code != "" {
		completeAliLoginJSON(w, r, code)
		return
	}

	if successURL == "" || !isSafeReturnURL(successURL) {
		core.Logger().Error("AliLogin fail: bad success_url",
			zap.String("success_url", successURL),
			zap.Bool("safe", isSafeReturnURL(successURL)),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	sess := core.GetSessionUser(r)
	hasOpenID := sess.WxOpenId != nil && strings.TrimSpace(sess.WxOpenId.String) != ""
	core.Logger().Info("AliLogin session check",
		zap.Uint64("uid", sess.UID),
		zap.Bool("has_openid", hasOpenID),
	)
	if protos.IsRealUserUID(sess.UID) && hasOpenID {
		core.Logger().Info("AliLogin already logged in, redirect", zap.String("to", successURL))
		http.Redirect(w, r, successURL, http.StatusTemporaryRedirect)
		return
	}

	startAliOAuth(w, r, successURL)
}

// AliOAuthCallback 支付宝网页授权回调。
// query: auth_code（或 code）、state（业务回跳 URL）
func AliOAuthCallback(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	code := firstNonEmpty(r.FormValue("auth_code"), r.FormValue("code"))
	state := strings.TrimSpace(r.FormValue("state"))
	core.Logger().Info("AliOAuthCallback start",
		zap.String("ua", r.UserAgent()),
		zap.Bool("has_code", code != ""),
		zap.Int("code_len", len(code)),
		zap.String("state", state),
		zap.String("raw_query", r.URL.RawQuery),
		zap.String("error", strings.TrimSpace(r.FormValue("error"))),
		zap.String("error_description", strings.TrimSpace(r.FormValue("error_description"))),
	)
	if code == "" || state == "" || !isSafeReturnURL(state) {
		core.Logger().Error("AliOAuthCallback fail: bad code/state",
			zap.Bool("has_code", code != ""),
			zap.String("state", state),
			zap.Bool("safe_state", isSafeReturnURL(state)),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	one, openId, err := exchangeAliCode(code)
	if err != nil || one == nil {
		core.Logger().Error("AliOAuthCallback fail: exchange", zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, errOr(err, common.ErrService))
		return
	}
	if !passportwx.SetWxUserToSession(w, r, one) {
		core.Logger().Error("AliOAuthCallback fail: SetWxUserToSession",
			zap.Uint64("uid", one.UID),
			zap.String("openid", openId),
		)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "aliUserId",
		Value:    openId,
		Path:     "/",
		Domain:   common.ServConfig.Domain,
		MaxAge:   common.ServConfig.SessionExpire,
		SameSite: http.SameSiteDefaultMode,
	})
	core.Logger().Info("AliOAuthCallback ok",
		zap.Uint64("uid", one.UID),
		zap.String("openid", openId),
		zap.String("domain", common.ServConfig.Domain),
		zap.String("return_url", state),
	)
	http.Redirect(w, r, state, http.StatusTemporaryRedirect)
}

func completeAliLoginJSON(w http.ResponseWriter, r *http.Request, code string) {
	core.Logger().Info("completeAliLoginJSON start",
		zap.Int("code_len", len(code)),
		zap.String("ua", r.UserAgent()),
	)
	one, openId, err := exchangeAliCode(code)
	if err != nil || one == nil {
		core.Logger().Error("completeAliLoginJSON fail: exchange", zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, errOr(err, common.ErrService))
		return
	}
	if !passportwx.SetWxUserToSession(w, r, one) {
		core.Logger().Error("completeAliLoginJSON fail: SetWxUserToSession",
			zap.Uint64("uid", one.UID),
			zap.String("openid", openId),
		)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "aliUserId",
		Value:    openId,
		Path:     "/",
		Domain:   common.ServConfig.Domain,
		MaxAge:   common.ServConfig.SessionExpire,
		SameSite: http.SameSiteDefaultMode,
	})
	core.Logger().Info("completeAliLoginJSON ok",
		zap.Uint64("uid", one.UID),
		zap.String("openid", openId),
		zap.String("domain", common.ServConfig.Domain),
	)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func exchangeAliCode(code string) (*protos.User, string, error) {
	if err := ensureClient(); err != nil {
		core.Logger().Error("exchangeAliCode fail: ensureClient", zap.Error(err))
		return nil, "", err
	}
	core.Logger().Info("exchangeAliCode SystemOauthToken",
		zap.String("app_id", common.ServConfig.AlipayAppID),
		zap.Int("code_len", len(code)),
	)
	rsp, err := aliClient.SystemOauthToken(context.Background(), alipay.SystemOauthToken{
		GrantType: "authorization_code",
		Code:      code,
	})
	if err != nil || rsp == nil || rsp.OpenId == "" {
		msg, subMsg := "", ""
		if rsp != nil {
			msg = rsp.Error.Msg
			subMsg = rsp.Error.SubMsg
		}
		core.Logger().Error("exchangeAliCode fail: SystemOauthToken",
			zap.Error(err),
			zap.Bool("rsp_nil", rsp == nil),
			zap.String("msg", msg),
			zap.String("sub_msg", subMsg),
			zap.Bool("has_openid", rsp != nil && rsp.OpenId != ""),
			zap.Bool("has_access_token", rsp != nil && rsp.AccessToken != ""),
		)
		if err == nil {
			err = common.ErrService
		}
		return nil, "", err
	}
	openId := rsp.OpenId
	core.Logger().Info("exchangeAliCode token ok",
		zap.String("openid", openId),
		zap.Bool("has_access_token", rsp.AccessToken != ""),
	)
	loginReq := &protos.UserReq{WxOpenId: openId}
	if rsp.AccessToken != "" {
		info, uerr := aliClient.UserInfoShare(context.Background(), alipay.UserInfoShare{AuthToken: rsp.AccessToken})
		if uerr != nil {
			core.Logger().Warn("exchangeAliCode UserInfoShare err (skip)", zap.Error(uerr))
		} else if info == nil || info.Code != alipay.CodeSuccess {
			core.Logger().Warn("exchangeAliCode UserInfoShare not success (skip)",
				zap.Any("info", info),
			)
		} else {
			loginReq.Nickname = info.NickName
			loginReq.AvatarURL = info.Avatar
			core.Logger().Info("exchangeAliCode UserInfoShare ok",
				zap.String("nickname", info.NickName),
			)
		}
	}
	loginReq.Ext = protos.MapStruct{"kind": "alipay", "alipay": 1, "wechat": 0}
	one, err := service.UserLoginByOpenID(loginReq)
	if err != nil || one == nil || !protos.IsRealUserUID(one.UID) {
		uid := uint64(0)
		if one != nil {
			uid = one.UID
		}
		core.Logger().Error("exchangeAliCode fail: UserLoginByOpenID",
			zap.Error(err),
			zap.String("openid", openId),
			zap.Uint64("uid", uid),
			zap.Bool("user_nil", one == nil),
		)
		if err == nil {
			err = common.ErrLogin
		}
		return nil, "", err
	}
	if one.Ext == nil {
		one.Ext = protos.MapStruct{}
	}
	one.SetExt("kind", "alipay")
	one.SetExt("alipay", 1)
	one.SetExt("wechat", 0)
	core.Logger().Info("exchangeAliCode login ok",
		zap.Uint64("uid", one.UID),
		zap.String("openid", openId),
	)
	return one, openId, nil
}

func startAliOAuth(w http.ResponseWriter, r *http.Request, returnURL string) {
	if common.ServConfig.AlipayAppID == "" {
		core.Logger().Error("startAliOAuth fail: empty AlipayAppID")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}
	scope := strings.TrimSpace(r.FormValue("scope"))
	if scope != "auth_base" {
		scope = "auth_user"
	}
	callback := strings.TrimSpace(r.FormValue("redirect_uri"))
	if callback == "" {
		callback = absoluteURL(r, pathAliOAuthCB, nil)
	}
	if !isSafeReturnURL(callback) {
		core.Logger().Error("startAliOAuth fail: bad callback", zap.String("callback", callback))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	// https://opendocs.alipay.com/open/284/web
	redirectURL := "https://openauth.alipay.com/oauth2/publicAppAuthorize.htm?app_id=" +
		url.QueryEscape(common.ServConfig.AlipayAppID) +
		"&scope=" + url.QueryEscape(scope) +
		"&redirect_uri=" + url.QueryEscape(callback) +
		"&state=" + url.QueryEscape(returnURL)
	core.Logger().Info("startAliOAuth redirect",
		zap.String("app_id", common.ServConfig.AlipayAppID),
		zap.String("scope", scope),
		zap.String("callback", callback),
		zap.String("state", returnURL),
	)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func errOr(err error, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

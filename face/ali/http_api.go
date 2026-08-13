package ali

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	passportwx "github.com/liuhengloveyou/passport/v3/face/wx"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	alipay "github.com/smartwalle/alipay/v3"
	"go.uber.org/zap"
)

var (
	aliOnce   sync.Once
	aliClient *alipay.Client
	aliInitErr error
)

func ensureClient() error {
	aliOnce.Do(func() {
		cfg := common.ServConfig
		if cfg.AlipayAppID == "" || cfg.AlipayPrivateKey == "" {
			aliInitErr = fmt.Errorf("alipay config empty")
			return
		}
		c, err := alipay.New(cfg.AlipayAppID, cfg.AlipayPrivateKey, true)
		if err != nil {
			aliInitErr = err
			return
		}
		if cfg.AlipayPublicKey != "" {
			if err = c.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
				aliInitErr = err
				return
			}
		}
		if cfg.AlipayEncryptKey != "" {
			if err = c.SetEncryptKey(cfg.AlipayEncryptKey); err != nil {
				aliInitErr = err
				return
			}
		}
		aliClient = c
	})
	return aliInitErr
}

// AliLogin 支付宝网页授权登录（仅身份：code → session；不含支付）。
// GET/POST /usercenter/ali/login?code=
func AliLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	code := strings.TrimSpace(r.FormValue("code"))
	if code == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := ensureClient(); err != nil {
		core.Logger().Error("alipay client", zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	rsp, err := aliClient.SystemOauthToken(context.Background(), alipay.SystemOauthToken{
		GrantType: "authorization_code",
		Code:      code,
	})
	if err != nil || rsp == nil || rsp.OpenId == "" {
		core.Logger().Error("alipay oauth token", zap.Error(err), zap.Any("rsp", rsp))
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrService)
		return
	}
	openId := rsp.OpenId
	accessToken := rsp.AccessToken

	loginReq := &protos.UserReq{WxOpenId: openId}
	if accessToken != "" {
		if info, uerr := aliClient.UserInfoShare(context.Background(), alipay.UserInfoShare{AuthToken: accessToken}); uerr == nil && info != nil && info.Code == alipay.CodeSuccess {
			loginReq.Nickname = info.NickName
			loginReq.AvatarURL = info.Avatar
		}
	}
	loginReq.Ext = protos.MapStruct{"kind": "alipay", "alipay": 1, "wechat": 0}
	one, err := service.UserLoginByWeixin(loginReq)
	if err != nil || one == nil || !protos.IsRealUserUID(one.UID) {
		core.Logger().Error("alipay login", zap.Error(err))
		if err == nil {
			err = common.ErrLogin
		}
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	if one.Ext == nil {
		one.Ext = protos.MapStruct{}
	}
	one.SetExt("kind", "alipay")
	one.SetExt("alipay", 1)
	one.SetExt("wechat", 0)

	if !passportwx.SetWxUserToSession(w, r, one) {
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
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

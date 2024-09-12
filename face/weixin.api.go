package face

import (
	"net/http"
	"strings"

	"github.com/liuhengloveyou/go-errors"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"
	"github.com/liuhengloveyou/passport/weixin"

	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4/zero"
)

func initWXAPI() {
	// 微信小程序登录
	// apis["wx/miniapp/login"] = Api{
	// 	Handler: WxMiniAppLogin,
	// }
	apis["wx/miniapp/updateInfo"] = Api{
		Handler:   WxMiniAppUserInfoUpdate,
		NeedLogin: true,
	}
}

/*
微信公众平台auth
https://developers.weixin.qq.com/doc/offiaccount/OA_Web_Apps/Wechat_webpage_authorization.html

https://open.weixin.qq.com/connect/oauth2/authorize?appid=wx0fd775b6dfdfc7d5&redirect_uri=http%3A%2F%2Fdevelopers.weixin.qq.com&response_type=code&scope=snsapi_userinfo&state=STATE#wechat_redirect
*/
func mpAuth(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	code := strings.TrimSpace(r.FormValue("code"))
	// 最多128字节
	state := strings.TrimSpace(r.FormValue("state"))
	logger.Sugar().Infoln("mpAuth param: ", code, state)
	if state == "" || code == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Errorf("h5Auth param ERR: %v %v\n", code, state)
		return
	}

	accessToken, err := weixin.GetAccessToken(common.ServConfig.AppID, common.ServConfig.AppSecret, code)
	if err != nil || accessToken == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrWxService)
		logger.Error("mpAuth GetAccessToken ERR: ", zap.Any("accessToken", accessToken), zap.Error(err))
		return
	}
	logger.Info("mpAuth: ", zap.String("code", code), zap.String("appId", common.ServConfig.AppID), zap.Any("accessToken", accessToken))

	wxUserInfo, err := weixin.GetUserInfo(accessToken.AccessToken, accessToken.OpenId)
	if err != nil || wxUserInfo == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrWxService)
		logger.Error("mpAuth GetUserInfo ERR: ", zap.Any("wxUserInfo", wxUserInfo), zap.Error(err))
		return
	}
	logger.Info("mpAuth: ", zap.String("code", code), zap.String("appId", common.ServConfig.AppID), zap.String("openId", accessToken.OpenId), zap.Any("wxUserInfo", wxUserInfo))

	// 登录
	loginReq := &protos.UserReq{
		WxOpenId: accessToken.OpenId,
	}

	one, err := service.UserLoginByWeixin(loginReq)
	if err != nil {
		myErr, ok := err.(*errors.Error)
		logger.Sugar().Errorf("mpAuth userLogin ERR: %v %v \n", loginReq, err.Error())
		if ok && myErr.Code == common.ErrLogin.Code {
			//
		} else {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}
	if one == nil {
		logger.Info("mpAuth userLogin: ", zap.String("code", code), zap.String("appId", common.ServConfig.AppID), zap.Any("req", loginReq))
		nickname := zero.StringFrom(wxUserInfo.Nickname)
		avatarurl := zero.StringFrom(wxUserInfo.Headimgurl)
		sex := zero.IntFrom(wxUserInfo.Sex)
		wxOpenId := zero.StringFrom(accessToken.OpenId)
		one = &protos.User{
			UID:       protos.WX_MP_AUTH_UID,
			WxOpenId:  &wxOpenId,
			Nickname:  &nickname,
			AvatarURL: &avatarurl,
			Gender:    &sex,
		}
	}

	// 删除老的会话信息
	r.Header.Del("Cookie")

	// 新建会话
	session, err := sessionStore.New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Sugar().Error("mpAuth session ERR: ", err)
		return
	}
	session.IsNew = true
	session.Options.Domain = common.ServConfig.Domain
	session.Options.MaxAge = 0

	session.Values[common.SessUserInfoKey] = one
	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Sugar().Error("mpAuth session ERR: ", err)
		return
	}

	logger.Info("mpAuth login ok: ", zap.Any("session", session.Values[common.SessUserInfoKey]))
	http.Redirect(w, r, state, http.StatusTemporaryRedirect)
}

// func WxMiniAppLogin(w http.ResponseWriter, r *http.Request) {
// 	var req protos.WxMiniAppLoginReq
// 	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
// 		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
// 		logger.Sugar().Error("WxMiniAppLogin param ERR: ", err)
// 		return
// 	}

// 	info, err := service.MiniAppService.Login(req.Code)
// 	if err != nil {
// 		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
// 		logger.Sugar().Error("WxMiniAppLogin MiniAppService.Login ERR: ", err)
// 		return
// 	}

// 	if info == nil {
// 		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
// 		logger.Sugar().Warnf("WxMiniAppLogin MiniAppService.Login ERR: %v %v\n", info, err)
// 		return
// 	}

// 	r.Header.Del("Cookie") // 删除老的会话信息
// 	session, err := sessionStore.New(r, common.SessionKey)
// 	if err != nil {
// 		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
// 		logger.Sugar().Error("WxMiniAppLogin session ERR: ", err)
// 		return
// 	}

// 	sessionUser := &protos.User{UID: 1}
// 	sessionUser.SetExt(protos.MiniAppSessionInfoKey, *info)
// 	session.Values[common.SessUserInfoKey] = *sessionUser

// 	if err := session.Save(r, w); err != nil {
// 		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
// 		logger.Sugar().Error("userLogin session ERR: ", err)
// 		return
// 	}

// 	token := strings.Split(w.Header().Get("Set-Cookie"), ";")[0][len(common.SessionKey)+1:]
// 	w.Header().Del("Set-Cookie")

// 	logger.Sugar().Infof("WxMiniAppLogin ok: %#v\n", info)
// 	gocommon.HttpErr(w, http.StatusOK, 0, token)
// }

func WxMiniAppUserInfoUpdate(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	//if sessionUser.TenantID <= 0 {
	//	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
	//	logger.Sugar().Error("AddRoleForUser TenantID ERR")
	//	return
	//}

	var req protos.WxMiniAppUserInfoUpdateReq
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		logger.Sugar().Error("WxMiniAppUserInfoUpdate param ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	logger.Sugar().Infof("WxMiniAppUserInfoUpdate: %#v %#v", sessionUser, req)
	//
	//if _, err := service.MiniAppService.WxMiniAppUserInfoUpdate(req); err != nil {
	//	logger.Sugar().Error(*user, err)
	//	gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
	//	return
	//}return

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

func SetWxUserToSession(w http.ResponseWriter, r *http.Request, userInfo *protos.User) {
	if userInfo == nil {
		return
	}

	// 删除老的会话信息
	r.Header.Del("Cookie")

	// 新建会话
	session, err := sessionStore.New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Sugar().Error("mpAuth session ERR: ", err)
		return
	}
	session.IsNew = true
	session.Options.MaxAge = 0

	session.Values[common.SessUserInfoKey] = userInfo
	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Sugar().Error("SetWxUserToSession session ERR: ", err)
		return
	}
}

func wxMpBindCellphone(w http.ResponseWriter, r *http.Request) {
	sess, auth := AuthFilter(r)
	if !auth {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}

	userReq := &protos.UserReq{}
	if err := readJsonBodyFromRequest(r, userReq, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("wxMpBindCellphone param ERR: ", err)
		return
	}

	sessUserInfo := sess.Values[common.SessUserInfoKey].(protos.User)
	logger.Info("wxMpBindCellphone: ", zap.Any("info", sessUserInfo), zap.Any("req", userReq))

	if len(userReq.Cellphone) == 0 || len(userReq.SmsCode) == 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrParam)
		logger.Error("wxMpBindCellphone ERR: ", zap.Any("info", sessUserInfo), zap.Any("req", userReq))
		return
	}

	if sessUserInfo.UID != protos.WX_MP_AUTH_UID {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrSession)
		return
	}
	if sessUserInfo.WxOpenId == nil || len(sessUserInfo.WxOpenId.String) == 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrSession)
		return
	}
	if sessUserInfo.Cellphone != nil && len(sessUserInfo.Cellphone.String) > 0 {
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrSession)
		return
	}

	if _, err := service.UpdateUserWxOpenIdByCellphone(userReq.Cellphone, sessUserInfo.WxOpenId.String, userReq.SmsCode); err != nil {
		logger.Error("wxMpBindCellphone ERR: ", zap.Any("req", userReq), zap.String("openId", sessUserInfo.WxOpenId.String), zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "成功")
}

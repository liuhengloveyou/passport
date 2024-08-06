package face

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-sql-driver/mysql"
	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"
	"go.uber.org/zap"
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
https://open.weixin.qq.com/connect/oauth2/authorize?appid=wx0fd775b6dfdfc7d5&redirect_uri=http%3A%2F%2Fdevelopers.weixin.qq.com&response_type=code&scope=snsapi_userinfo&state=STATE#wechat_redirect

auth通过的添加到数据库
*/
func h5Auth(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	code := strings.TrimSpace(r.FormValue("code"))
	// 最多128字节
	state := strings.TrimSpace(r.FormValue("state"))
	logger.Sugar().Infoln("h5Auth: ", code, state)
	if state == "" || code == "" {
		logger.Sugar().Errorf("h5Auth param ERR: %v %v\n", code, state)
		return
	}

	weixinApiUrl := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		common.ServConfig.AppID, common.ServConfig.AppSecret, code)
	resp, body, err := gocommon.GetRequest(weixinApiUrl, nil)
	if err != nil {
		logger.Sugar().Errorf("huAuth weixin ERR: %v\n", err)
		return
	}
	logger.Sugar().Infof("h5Auth: code:%v; stat: %v; %v\n", code, state, string(body))

	if resp.StatusCode != 200 {
		logger.Sugar().Error("wx http ERR: ", resp.StatusCode)
		return
	}

	var wxResp protos.MiniAppSessionInfo
	if err := json.Unmarshal(body, &wxResp); err != nil {
		logger.Sugar().Error("wx resp json ERR: ", err)
		return
	}

	if wxResp.ErrCode != 0 {
		logger.Sugar().Error("wx resp ERR: ", wxResp.ErrCode, wxResp.ErrMsg)
		return
	}

	logger.Info("h5Auth: ", zap.String("code", code), zap.String("state", state), zap.String("resp", string(body)))

	//　保存用户信息
	user := &protos.UserReq{
		WxOpenId: wxResp.Openid,
		Nickname: "wx-" + wxResp.Openid,
		Password: "000000",
	}
	uid, err := service.AddUserService(user)
	user.UID = uid
	logger.Info("h5Auth: ", zap.String("code", code), zap.String("state", state), zap.Any("user", user))
	if err != nil && err != common.ErrWxOpenidDup && err != common.ErrNickDup && err != common.ErrMysql1062 {
		logger.Sugar().Error("h5Auth service.AddUser ERR: ", err)

		if merr, ok := err.(*mysql.MySQLError); ok {
			if merr.Number == 1062 {
				gocommon.HttpJsonErr(w, http.StatusOK, common.ErrMysql1062)
				return
			}
		}

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	// 登录账号
	user.Password = "000000"
	logined, err := service.UserLogin(user)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		logger.Sugar().Errorf("h5Auth ERR: %v %v \n", user, err.Error())
		return
	}
	if logined == nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "用户不存在")
		logger.Sugar().Warnf("h5Auth 用户不存在: %v\n", user)
		return
	}

	r.Header.Del("Cookie") // 删除老的会话信息
	session, err := sessionStore.New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Sugar().Error("h5Auth session ERR: ", err)
		return
	}

	session.Values[common.SessUserInfoKey] = logined

	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Sugar().Error("h5Auth session ERR: ", err)
		return
	}

	logger.Sugar().Infof("h5Auth login ok: %v sess :%#v\n", user, session)
	http.Redirect(w, r, state, http.StatusTemporaryRedirect)

	return
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

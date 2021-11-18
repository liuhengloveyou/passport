package face

import (
	"net/http"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"
)

func initWXAPI() {
	// 微信小程序登录
	apis["wx/miniapp/login"] = Api{
		Handler: WxMiniAppLogin,
	}
	apis["wx/miniapp/updateInfo"] = Api{
		Handler:   WxMiniAppUserInfoUpdate,
		NeedLogin: true,
	}
}

func WxMiniAppLogin(w http.ResponseWriter, r *http.Request) {
	var req protos.WxMiniAppLoginReq
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("WxMiniAppLogin param ERR: ", err)
		return
	}

	info, err := service.MiniAppService.Login(req.Code)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("WxMiniAppLogin MiniAppService.Login ERR: ", err)
		return
	}

	if info == nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Warnf("WxMiniAppLogin MiniAppService.Login ERR: %v %v\n", info, err)
		return
	}

	r.Header.Del("Cookie") // 删除老的会话信息
	session, err := sessionStore.New(r, common.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("WxMiniAppLogin session ERR: ", err)
		return
	}

	sessionUser := &protos.User{UID: 1}
	sessionUser.SetExt("MiniAppSessionInfo", *info)
	session.Values[common.SessUserInfoKey] = *sessionUser

	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("userLogin session ERR: ", err)
		return
	}

	token := strings.Split(w.Header().Get("Set-Cookie"), ";")[0][len(common.SessionKey)+1:]
	w.Header().Del("Set-Cookie")

	logger.Infof("WxMiniAppLogin ok: %#v\n", info)
	gocommon.HttpErr(w, http.StatusOK, 0, token)
}

func WxMiniAppUserInfoUpdate(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	//if sessionUser.TenantID <= 0 {
	//	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNotFound)
	//	logger.Error("AddRoleForUser TenantID ERR")
	//	return
	//}

	var req protos.WxMiniAppUserInfoUpdateReq
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		logger.Error("WxMiniAppUserInfoUpdate param ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	logger.Infof("WxMiniAppUserInfoUpdate: %#v %#v", sessionUser, req)
	//
	//if _, err := service.MiniAppService.WxMiniAppUserInfoUpdate(req); err != nil {
	//	logger.Error(*user, err)
	//	gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
	//	return
	//}return

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

func SetWxUserToSession(w http.ResponseWriter, r *http.Request, userInfo *protos.User) {
	if userInfo == nil {
		return
	}

	r.Header.Del("Cookie") // 删除老的会话信息
	session, err := sessionStore.New(r, common.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("SetWxUserToSession session ERR: ", err)
		return
	}

	session.Values[common.SessUserInfoKey] = userInfo

	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("SetWxUserToSession session ERR: ", err)
		return
	}
}

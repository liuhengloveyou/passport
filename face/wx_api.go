package face

import (
	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"net/http"
	"strings"
)

func initWXAPI() {
	// 微信小程序登录
	apis["wx/miniapp/login"] = Api{
		Handler: WxMiniAppLogin,
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
	session.Values[common.SessUserInfoKey] = sessionUser

	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("userLogin session ERR: ", err)
		return
	}

	token := strings.Split(w.Header().Get("Set-Cookie"), ";")[0][len(common.SessionKey)+1:]
	w.Header().Del("Set-Cookie")

	logger.Infof("WxMiniAppLogin ok: %#v %v\n", info, token)
	gocommon.HttpErr(w, http.StatusOK, 0, token)
}
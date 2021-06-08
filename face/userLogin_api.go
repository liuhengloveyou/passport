package face

import (
	"net/http"
	"strings"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"

	gocommon "github.com/liuhengloveyou/go-common"
)

func userLogin(w http.ResponseWriter, r *http.Request) {
	useCookie := true
	user := &protos.UserReq{}

	if strings.ToLower(r.Header.Get("USE-COOKIE")) == "false" {
		useCookie = false
	}

	if err := readJsonBodyFromRequest(r, user, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("userLogin param ERR: ", err)
		return
	}
	logger.Infof("userLogin: %#vv\n", user)

	one, err := service.UserLogin(user)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		logger.Errorf("userLogin ERR: %v %v \n", user, err.Error())
		return
	}
	if one == nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "用户不存在")
		logger.Warnf("userLogin 用户不存在: %v\n", user)
		return
	}

	r.Header.Del("Cookie") // 删除老的会话信息
	session, err := sessionStore.New(r, common.SessionKey)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("userLogin session ERR: ", err)
		return
	}

	session.Values[common.SessUserInfoKey] = one

	if err := session.Save(r, w); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrSession)
		logger.Error("userLogin session ERR: ", err)
		return
	}

	if !useCookie {
		one.Ext["TOKEN"] = strings.Split(w.Header().Get("Set-Cookie"), "=")[1]
		w.Header().Del("Set-Cookie")
	}

	logger.Infof("user login ok: %v sess :%#v\n", user, session)

	gocommon.HttpErr(w, http.StatusOK, 0, one)
}

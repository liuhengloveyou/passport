package face

import (
	"encoding/json"
	"github.com/liuhengloveyou/passport/protos"
	"io/ioutil"
	"net/http"

	"github.com/liuhengloveyou/passport/service"

	gocommon "github.com/liuhengloveyou/go-common"
)

func userLogin(w http.ResponseWriter, r *http.Request) {
	user := &protos.UserReq{}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("userLogin ioutil.ReadAll(r.Body) ERR: ", err)
		gocommon.HttpErr(w, http.StatusBadRequest, 0, err.Error())
		return
	}

	if err = json.Unmarshal(body, user); err != nil {
		logger.Error("userAdd json.Unmarshal(body, user) ERR: ", string(body))
		gocommon.HttpErr(w, http.StatusBadRequest, -1, err.Error())
		return
	}

	logger.Infof("userLogin: %#vv\n", user)

	one, err := service.UserLogin(user)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		logger.Errorf("userLogin ERR: %v %v \n", user, err.Error())
		return
	}
	if one == nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "用户不存在")
		logger.Warnf("userLogin 用户不存在: %v\n", user)
		return
	}

	session, err := store.Get(r, SessionKey)
	if err != nil {
		logger.Error("userLogin session ERR: ", err)
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}

	session.Values[SessUserInfoKey] = one

	if err := session.Save(r, w); err != nil {
		logger.Error("userLogin session ERR: ", err)
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}

	logger.Infof("user login ok: %v ccc :%#v\n", user, session)

	gocommon.HttpErr(w, http.StatusOK, 0, one)

	return
}

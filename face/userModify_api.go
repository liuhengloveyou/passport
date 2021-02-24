package face

import (
	"encoding/json"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"io/ioutil"
	"net/http"

	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func userModify(w http.ResponseWriter, r *http.Request) {
	sess, auth := AuthFilter(w, r)
	if auth == false {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}

	user := &protos.UserReq{}
	if err := readJsonBodyFromRequest(r, user); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("userLogin param ERR: ", err)
		return
	}
	logger.Infof("userModify: %#vv\n", user)

	info := sess.Values[SessUserInfoKey].(protos.User)
	logger.Info("userModify", user, info)

	if info.Cellphone != nil && user.Cellphone == info.Cellphone.String {
		user.Cellphone = "" // 不需要更新
	}

	if info.Email != nil && user.Email == info.Email.String {
		user.Email = "" // 不需要更新
	}

	if info.Nickname != nil && user.Nickname == info.Nickname.String {
		user.Nickname = "" // 不需要更新
	}

	if user.Gender < 0 || user.Gender > 2 {
		logger.Error("userModify gender ERR: ", user.Gender)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "性别取值错误")
		return
	}

	if user.AvatarURL != "" && len(user.AvatarURL) > 128 {
		logger.Error("userModify AvatarURL ERR: ", user.AvatarURL)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "头像地址过长")
		return
	}

	if user.Addr != "" && len(user.Addr) > 256 {
		logger.Error("userModify Addr ERR: ", user.Addr)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "地址过长")
		return
	}

	user.UID = info.UID

	if _, err := service.UpdateUserService(user); err != nil {
		logger.Error(*user, err)
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
	return
}

func modifyPWD(w http.ResponseWriter, r *http.Request) {
	var uid uint64

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("modifyPWD body ERR: ", err)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, err.Error())
		return
	}

	logger.Info("modifyPWD body: ", string(body))

	pwd := make(map[string]string)
	err = json.Unmarshal(body, &pwd)
	if err != nil {
		logger.Error("modifyPWD json ERR: ", err)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, err.Error())
		return
	}

	o := pwd["o"]
	n := pwd["n"]

	if o == "" {
		logger.Error("modifyPWD old nil")
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "旧密码不能为空")
		return
	} else if n == "" {
		logger.Error("modifyPWD new nil")
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "新密码不能为空")
		return
	}

	uid = r.Context().Value("session").(*sessions.Session).Values[SessUserInfoKey].(protos.User).UID

	logger.Infof("modifyPWD %d %s %s\n", uid, o, n)

	if _, err := service.UpdateUserPWD(uid, o, n); err != nil {
		logger.Errorf("modifyPWD %d %s %s %s\n", uid, o, n, err.Error())
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
	return
}

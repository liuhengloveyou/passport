package face

import (
	"net/http"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/service"
)

func userModify(w http.ResponseWriter, r *http.Request) {
	sess, auth := AuthFilter(r)
	if auth == false {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}

	user := &protos.UserReq{}
	if err := readJsonBodyFromRequest(r, user, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("userLogin param ERR: ", err)
		return
	}
	logger.Sugar().Infof("userModify: %#v\n", user)

	info := sess.Values[common.SessUserInfoKey].(protos.User)
	logger.Sugar().Info("userModify", user, info)

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
		logger.Sugar().Error("userModify gender ERR: ", user.Gender)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "性别取值错误")
		return
	}

	if user.AvatarURL != "" && len(user.AvatarURL) > 128 {
		logger.Sugar().Error("userModify AvatarURL ERR: ", user.AvatarURL)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "头像地址过长")
		return
	}

	if user.Addr != "" && len(user.Addr) > 256 {
		logger.Sugar().Error("userModify Addr ERR: ", user.Addr)
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "地址过长")
		return
	}

	user.UID = info.UID

	if _, err := service.UpdateUserService(user); err != nil {
		logger.Sugar().Error(*user, err)
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

func modifyPWD(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User).UID

	req := protos.ModifyPwdReq{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("modifyPWD param ERR: ", err)
		return
	}

	logger.Sugar().Infof("modifyPWD body: %v %v\n", uid, req)

	if _, err := service.UpdateUserPWD(uid, req.OldPwd, req.NewPwd); err != nil {
		logger.Sugar().Errorf("modifyPWD %d %v %s\n", uid, req, err.Error())
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func getbackPWD(w http.ResponseWriter, r *http.Request) {
	req := protos.GetbackPwdReq{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("getbackPWD param ERR: ", err)
		return
	}

	logger.Sugar().Infof("getbackPWD body: %v\n", req)

	if _, err := service.UpdateUserPWDBySms(req.Cellphone, req.SmsCode, req.NewPwd); err != nil {
		logger.Sugar().Errorf("getbackPWD ERR: %#v %s\n", req, err.Error())
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	return
}

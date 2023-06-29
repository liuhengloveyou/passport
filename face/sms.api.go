package face

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sms"
)

// 发送注册账号短信验证码
func SendUserAddSmsCode(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("SendUserAddSmsCode param ERR: ", err)
		return
	}
	logger.Infof("SendUserAddSmsCode body: %#v\n", req)

	if req.Cellphone == "" {
		logger.Error("手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendUserAddSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Error("SendUserAddSmsCode ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Info("SendUserAddSmsCode OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// 发送用户登录短信验证码
func SendUserLoginSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("SendUserLoginSms param ERR: ", err)
		return
	}
	logger.Infof("SendUserLoginSms body: %#v\n", req)

	if req.Cellphone == "" {
		logger.Error("手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendUserLoginSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Error("SendUserLoginSms ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Info("SendUserLoginSms OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// 发送用户登录短信验证码
func SendGetBackPwdSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("SendGetBackPwdSms param ERR: ", err)
		return
	}
	logger.Infof("SendGetBackPwdSms body: %#v\n", req)

	if req.Cellphone == "" {
		logger.Error("手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendGetBackPwdSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Error("SendGetBackPwdSms ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Info("SendGetBackPwdSms OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

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
		logger.Sugar().Error("SendUserAddSmsCode param ERR: ", err)
		return
	}
	logger.Sugar().Infof("SendUserAddSmsCode body: %#v\n", req)

	if req.Cellphone == "" {
		logger.Sugar().Error("手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendUserAddSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Sugar().Error("SendUserAddSmsCode ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Sugar().Info("SendUserAddSmsCode OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// 发送用户登录短信验证码
func SendUserLoginSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("SendUserLoginSms param ERR: ", err)
		return
	}
	logger.Sugar().Infof("SendUserLoginSms body: %#v\n", req)

	if req.Cellphone == "" {
		logger.Sugar().Error("手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendUserLoginSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Sugar().Error("SendUserLoginSms ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Sugar().Info("SendUserLoginSms OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// 发送找回密码短信验证码
func SendGetBackPwdSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("SendGetBackPwdSms param ERR: ", err)
		return
	}
	logger.Sugar().Infof("SendGetBackPwdSms body: %#v\n", req)

	if req.Cellphone == "" {
		logger.Sugar().Error("手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendGetBackPwdSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Sugar().Error("SendGetBackPwdSms ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Sugar().Info("SendGetBackPwdSms OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// 发送微信绑定手机号短信验证码
func SendWxBindSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("SendWxBindSms param ERR: ", err)
		return
	}
	logger.Sugar().Infof("SendWxBindSms body: %#v\n", req)

	if len(req.Cellphone) == 0 {
		logger.Sugar().Error("SendWxBindSms 手机号为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	code, err := sms.SendWxBindSms(req.Cellphone, req.AliveSec)
	if err != nil {
		logger.Sugar().Error("SendWxBindSms ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Sugar().Info("SendWxBindSms OK:", req.Cellphone, code)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

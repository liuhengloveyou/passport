// http_api.go 提供短信验证码发送接口。
package sms

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	servicesms "github.com/liuhengloveyou/passport/v3/sms"
)

// SendUserAddSmsCode 发送用户注册验证码。
func SendUserAddSmsCode(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.Cellphone == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, err := servicesms.SendUserAddSms(req.Cellphone, req.AliveSec); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// SendUserLoginSms 发送用户登录验证码。
func SendUserLoginSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.Cellphone == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, err := servicesms.SendUserLoginSms(req.Cellphone, req.AliveSec); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// SendGetBackPwdSms 发送找回密码验证码。
func SendGetBackPwdSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.Cellphone == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, err := servicesms.SendGetBackPwdSms(req.Cellphone, req.AliveSec); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// SendWxBindSms 发送微信绑定手机号验证码。
func SendWxBindSms(w http.ResponseWriter, r *http.Request) {
	req := &protos.SmsReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.Cellphone == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, err := servicesms.SendWxBindSms(req.Cellphone, req.AliveSec); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

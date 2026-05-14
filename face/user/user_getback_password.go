package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserGetBackPassword 通过短信验证码找回并重置密码。
func UserGetBackPassword(w http.ResponseWriter, r *http.Request) {
	req := protos.GetbackPwdReq{}
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, err := service.UpdateUserPWDBySms(req.Cellphone, req.SmsCode, req.NewPwd); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

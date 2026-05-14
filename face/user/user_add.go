package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	"github.com/liuhengloveyou/passport/v3/sms"
)

// UserAdd 注册用户，支持手机号短信校验后创建账号。
func UserAdd(w http.ResponseWriter, r *http.Request) {
	user := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, user, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		core.Logger().Sugar().Error("userAdd param ERR: ", err)
		return
	}
	if user.Cellphone == "" && user.Email == "" && user.Nickname == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if user.Password == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrPWD)
		return
	}
	if len(user.Cellphone) > 0 && len(user.SmsCode) > 0 {
		if err := sms.CheckSmsCode(user.Cellphone, user.SmsCode); err != nil && err != sms.ErrSmsNotInit {
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}
	uid, err := service.AddUserService(user)
	if err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, uid)
}

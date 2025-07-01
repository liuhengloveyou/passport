package face

import (
	"net/http"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sms"

	"github.com/jackc/pgx/v5/pgconn"
	gocommon "github.com/liuhengloveyou/go-common"
)

func userAdd(w http.ResponseWriter, r *http.Request) {
	user := &protos.UserReq{}
	if err := readJsonBodyFromRequest(r, user, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("userAdd param ERR: ", err)
		return
	}
	logger.Sugar().Infof("userAdd body: %#v\n", user)

	if user.Cellphone == "" && user.Email == "" && user.Nickname == "" {
		logger.Sugar().Error("userAdd ERR: 用户手机号和邮箱地址同时为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if user.Password == "" {
		logger.Sugar().Error("userAdd ERR: 用户密码为空")
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrPWD)
		return
	}

	if len(user.Cellphone) > 0 && len(user.SmsCode) > 0 {
		if err := sms.CheckSmsCode(user.Cellphone, user.SmsCode); err != nil && err != sms.ErrSmsNotInit {
			logger.Sugar().Error("userAdd ERR: 短信验证码错误")
			gocommon.HttpJsonErr(w, http.StatusOK, err)
			return
		}
	}

	uid, err := service.AddUserService(user)
	if err != nil {
		logger.Sugar().Error("userAdd service.AddUser ERR: ", err)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" { // 唯一约束冲突
			gocommon.HttpJsonErr(w, http.StatusOK, common.ErrPgDupKey)
			return
		}

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Sugar().Info("user add ok:", uid)
	gocommon.HttpErr(w, http.StatusOK, 0, uid)

	return
}

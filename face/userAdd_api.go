package face

import (
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"net/http"

	"github.com/liuhengloveyou/passport/service"

	"github.com/go-sql-driver/mysql"
	gocommon "github.com/liuhengloveyou/go-common"
)

func userAdd(w http.ResponseWriter, r *http.Request) {
	user := &protos.UserReq{}

	if err := readJsonBodyFromRequest(r, user); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("userAdd param ERR: ", err)
		return
	}
	logger.Infof("userAdd body: %#v\n", user)

	if user.Cellphone == "" && user.Email == "" {
		logger.Error("ERR: 用户手机号和邮箱地址同时为空")
		gocommon.HttpErr(w, http.StatusOK, -1, "手机号和邮箱同时为空")
		return
	}
	if user.Password == "" {
		logger.Error("ERR: 用户密码为空")
		gocommon.HttpErr(w, http.StatusOK, -1, "密码为空")
		return
	}

	uid, err := service.AddUserService(user)
	if err != nil {
		logger.Error("userAdd service.AddUser ERR: ", err)
		if merr, ok := err.(*mysql.MySQLError); ok {
			if merr.Number == 1062 {
				gocommon.HttpErr(w, http.StatusOK, -1, "重复注册")
				return
			}
		}

		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	logger.Info("user add ok:", uid)
	gocommon.HttpErr(w, http.StatusOK, 0, uid)

	return
}

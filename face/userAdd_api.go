package face

import (
	"encoding/json"
	"github.com/liuhengloveyou/passport/protos"
	"io/ioutil"
	"net/http"

	"github.com/liuhengloveyou/passport/service"

	"github.com/go-sql-driver/mysql"
	gocommon "github.com/liuhengloveyou/go-common"
)

func userAdd(w http.ResponseWriter, r *http.Request) {
	user := &protos.UserReq{}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error("userAdd ioutil.ReadAll(r.Body) ERR: ", err)
		gocommon.HttpErr(w, http.StatusBadRequest, 0, err.Error())
		return
	}

	if err = json.Unmarshal(body, user); err != nil {
		logger.Error("userAdd json.Unmarshal(body, user) ERR: ", string(body))
		gocommon.HttpErr(w, http.StatusBadRequest, -1, err.Error())
		return
	}

	logger.Infof("userAdd body: %s %#v\n", string(body), user)

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

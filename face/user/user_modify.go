package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserModify 修改当前登录用户基础资料。
func UserModify(w http.ResponseWriter, r *http.Request) {
	sess, auth := core.AuthFilter(r)
	if !auth {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}
	user := &protos.UserReq{}
	if err := core.ReadJSONBodyFromRequest(r, user, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	info := sess.Values[common.SessUserInfoKey].(protos.User)
	if info.Cellphone != nil && user.Cellphone == info.Cellphone.String {
		user.Cellphone = ""
	}
	if info.Email != nil && user.Email == info.Email.String {
		user.Email = ""
	}
	if info.Nickname != nil && user.Nickname == info.Nickname.String {
		user.Nickname = ""
	}
	user.UID = info.UID
	if _, err := service.UpdateUserService(user); err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
)

// UserModifyPassword 修改当前登录用户密码。
func UserModifyPassword(w http.ResponseWriter, r *http.Request) {
	sess, auth := core.AuthFilter(r)
	if !auth {
		gocommon.HttpErr(w, http.StatusForbidden, -1, "末登录用户")
		return
	}
	uid := sess.Values[common.SessUserInfoKey].(protos.User).UID
	req := protos.ModifyPwdReq{}
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if _, err := service.UpdateUserPWD(uid, req.OldPwd, req.NewPwd); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

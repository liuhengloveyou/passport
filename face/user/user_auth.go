package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
)

// UserAuth 校验当前会话登录状态并返回会话用户信息。
func UserAuth(w http.ResponseWriter, r *http.Request) {
	sess, auth := core.AuthFilter(r)
	if !auth || sess == nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}
	gocommon.HttpErr(w, http.StatusOK, 0, sess.Values[common.SessUserInfoKey].(protos.User))
}

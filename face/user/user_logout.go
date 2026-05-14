package user

import (
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/sessions"
)

// UserLogout 退出登录并清理当前会话。
func UserLogout(w http.ResponseWriter, r *http.Request) {
	var uid uint64
	if r.Context().Value("session") != nil {
		uid = r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User).UID
	}
	session, err := core.SessionStore().New(r, common.ServConfig.SessionKey)
	if err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}
	session.Values[common.SessUserInfoKey] = nil
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}
	core.Logger().Sugar().Infof("userLogout ok: %v\n", uid)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

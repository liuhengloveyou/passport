package face

import (
	"github.com/liuhengloveyou/passport/common"
	"net/http"

	"github.com/liuhengloveyou/passport/protos"

	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func userLogout(w http.ResponseWriter, r *http.Request) {
	var uid uint64
	if r.Context().Value("session") != nil {
		uid = r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User).UID
	}

	session, err := sessionStore.Get(r, common.SessionKey)
	if err != nil {
		logger.Error("userLogout session ERR: ", err)
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}

	session.Values[common.SessUserInfoKey] = nil
	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		logger.Error("userLogout session ERR: ", err)
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}

	logger.Infof("userLogout ok: %v\n", uid)

	return
}

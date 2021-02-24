package face

import (
	"github.com/liuhengloveyou/passport/protos"
	"net/http"

	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func userLogout(w http.ResponseWriter, r *http.Request) {
	_, auth := AuthFilter(w, r)
	if auth == false {
		gocommon.HttpErr(w, http.StatusForbidden, 0, true)
		return
	}

	var uid uint64
	if r.Context().Value("session") != nil {
		uid = r.Context().Value("session").(*sessions.Session).Values[SessUserInfoKey].(protos.User).UID
	}

	session, err := store.Get(r, SessionKey)
	if err != nil {
		logger.Error("userLogout session ERR: ", err)
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}

	session.Values[SessUserInfoKey] = nil
	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		logger.Error("userLogout session ERR: ", err)
		gocommon.HttpErr(w, http.StatusOK, -1, "会话错误")
		return
	}

	logger.Infof("userLogout ok: %v\n", uid)

	return
}

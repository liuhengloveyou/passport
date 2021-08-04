package face

import (
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"net/http"

	gocommon "github.com/liuhengloveyou/go-common"
)

func UserAuth(w http.ResponseWriter, r *http.Request) {
	sess, auth := AuthFilter(r)
	if auth == false || sess == nil {
		logger.Error("UserAuth auth false")
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}

	if false == AccessFilter(r) {
		logger.Error("UserAuth AccessFilter false")
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrNoAuth)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, sess.Values[common.SessUserInfoKey].(protos.User))
	logger.Infof("UserAuth OK: %#v", sess.Values[common.SessUserInfoKey].(protos.User))

	return
}
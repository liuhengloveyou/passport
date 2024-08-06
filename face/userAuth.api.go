package face

import (
	"net/http"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	gocommon "github.com/liuhengloveyou/go-common"
)

func UserAuth(w http.ResponseWriter, r *http.Request) {
	sess, auth := AuthFilter(r)
	if auth == false || sess == nil {
		logger.Sugar().Error("UserAuth auth false")
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}

	if false == AccessFilter(r) {
		logger.Sugar().Error("UserAuth AccessFilter false")
		gocommon.HttpJsonErr(w, http.StatusForbidden, common.ErrNoAuth)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, sess.Values[common.SessUserInfoKey].(protos.User))
	logger.Sugar().Infof("UserAuth OK: %#v", sess.Values[common.SessUserInfoKey].(protos.User))

	return
}

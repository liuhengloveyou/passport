package face

import (
	"net/http"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func tenantAdd(w http.ResponseWriter, r *http.Request) {
	req := &protos.Tenant{}
	if r.Context().Value("session") != nil {
		req.UID = r.Context().Value("session").(*sessions.Session).Values[SessUserInfoKey].(protos.User).UID
	}
	if req.UID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrSession)
		logger.Error("session ERR")
		return
	}

	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("tenantAdd param ERR: ", err)
		return
	}
	logger.Infof("tenantAdd body: %#v\n", req)

	if req.TenantName == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNameNull)
		logger.Error("tenantAdd param ERR: ", req)
		return
	}
	if req.TenantType == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantTypeNull)
		logger.Error("tenantAdd param ERR: ", req)
		return
	}

	uid, err := service.TenantAdd(req)
	if err != nil {
		logger.Error("tenantAdd service ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Info("user add ok:", uid)
	gocommon.HttpErr(w, http.StatusOK, 0, uid)

	return
}

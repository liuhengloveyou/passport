package face

import (
	"net/http"

	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	gocommon "github.com/liuhengloveyou/go-common"
)

func AddRoleForUser(w http.ResponseWriter, r *http.Request) {
	req := &protos.RoleReq{}

	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.AddRoleForUser(req.UID, req.Role); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("AddRoleForUser OK: %#v\n", req)

	return
}

func DeleteRoleForUser(w http.ResponseWriter, r *http.Request) {
	req := &protos.RoleReq{}

	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.DeleteRoleForUser(req.UID, req.Role); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("DeleteRoleForUser OK: %#v\n", req)

	return
}

func AddPolicy(w http.ResponseWriter, r *http.Request) {
	req := &protos.PolicyReq{}

	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.AddPolicy(req.UID, req.Sub, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("AddPolicy OK: %#v\n", req)

	return
}

func RemovePolicy(w http.ResponseWriter, r *http.Request) {
	req := &protos.PolicyReq{}

	if err := readJsonBodyFromRequest(r, req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	if err := accessctl.RemovePolicy(req.UID, req.Sub, req.Act); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("RemovePolicy OK: %#v\n", req)

	return
}

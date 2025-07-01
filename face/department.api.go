package face

import (
	"net/http"
	"strconv"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
)

func addDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req protos.Department
	if err := readJsonBodyFromRequest(r, &req, 2048); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Errorf("addDepartment param ERR: %v\n", err)
		return
	}

	req.TenantID = sessionUser.TenantID
	req.UserId = sessionUser.UID
	logger.Sugar().Info("addDepartment: ", req)

	lastInsertId, err := service.DepartmentCreate(&req)
	if err != nil {
		logger.Sugar().Errorf("addDepartment service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, lastInsertId)
	logger.Sugar().Info("addDepartment OK: ", lastInsertId, req)
}

func deleteDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	logger.Sugar().Infof("deleteDepartment id: %v\n", id)

	err := service.DepartmentDelete(id, sessionUser.TenantID)
	if err != nil {
		logger.Sugar().Errorf("deleteDepartment service ERR: %v\n", err)
		gocommon.HttpErr(w, http.StatusOK, -1, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Infof("deleteDepartment id: %v\n", id)
}

func listDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("page_size"), 10, 64)
	logger.Sugar().Infof("listDepartment id: %v\n", id)

	list, err := service.DepartmentFind(id, sessionUser.TenantID, page, pageSize)
	if err != nil {
		logger.Sugar().Errorf("listDepartment service ERR: %v\n", err)
		gocommon.HttpErr(w, http.StatusOK, -1, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, list)
	logger.Sugar().Infof("listDepartment id: %v %v\n", id, len(list))
}

func updateDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req protos.Department
	if err := readJsonBodyFromRequest(r, &req, 2048); err != nil {
		logger.Sugar().Errorf("updateDepartment param ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	logger.Sugar().Info("updateDepartment: ", req)
	req.TenantID = sessionUser.TenantID

	err := service.DepartmentUpdate(&req)
	if err != nil {
		logger.Sugar().Errorf("updateDepartment service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Info("updateDepartment OK: ", req)
}

func updateDepartmentConfig(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 || sessionUser.UID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req protos.KvReq
	if err := readJsonBodyFromRequest(r, &req, 2048); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("updateDepartmentConfig param ERR: ", err)
		return
	}
	logger.Sugar().Infof("updateDepartmentConfig %d %d %v\n", sessionUser.UID, sessionUser.TenantID, req)

	err := service.DepartmentUpdateConfig(req.ID, sessionUser.UID, sessionUser.TenantID, req.K, req.V)
	if err != nil {
		logger.Sugar().Errorf("updateDepartmentConfig service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Sugar().Info("updateDepartmentConfig OK: ", req)
}

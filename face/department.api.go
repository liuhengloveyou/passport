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
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Errorf("addDepartment param ERR: %v\n", err)
		return
	}

	req.TenantID = sessionUser.TenantID
	req.UserId = sessionUser.UID
	logger.Info("addDepartment: ", req)

	lastInsertId, err := service.DepartmentCreate(&req)
	if err != nil {
		logger.Errorf("addDepartment service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, lastInsertId)
	logger.Info("addDepartment OK: ", lastInsertId, req)
}

func deleteDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	logger.Infof("deleteDepartment id: %v\n", id)

	err := service.DepartmentDelete(id, sessionUser.TenantID)
	if err != nil {
		logger.Errorf("deleteDepartment service ERR: %v\n", err)
		gocommon.HttpErr(w, http.StatusOK, -1, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Infof("deleteDepartment id: %v\n", id)
}

func listDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 64)
	logger.Infof("listDepartment id: %v\n", id)

	list, err := service.DepartmentFind(id, sessionUser.TenantID)
	if err != nil {
		logger.Errorf("listDepartment service ERR: %v\n", err)
		gocommon.HttpErr(w, http.StatusOK, -1, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, list)
	logger.Infof("listDepartment id: %v %v\n", id, len(list))
}

func updateDepartment(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	var req protos.Department
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		logger.Errorf("updateDepartment param ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	logger.Info("updateDepartment: ", req)
	req.TenantID = sessionUser.TenantID

	err := service.DepartmentUpdate(&req)
	if err != nil {
		logger.Errorf("updateDepartment service ERR: %v\n", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
	logger.Info("updateDepartment OK: ", req)
}

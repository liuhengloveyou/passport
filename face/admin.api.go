package face

import (
	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/service"
	"net/http"
	"strconv"
	"strings"
)

func TenantList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	rr, e := service.TenantList(page, pageSize, hasTotal == 1)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		logger.Error("TenantList ERR: ", e)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rr)

	return
}

func UpdateTenantConfiguration(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param ERR: ", err)
		return
	}

	tenantID, ok := req["tenant_id"].(float64)
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param tenantID nil")
		return
	}

	data, ok := req["data"].(map[string]interface{})
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param data nil")
		return
	}

	logger.Infof("UpdateTenantConfiguration : %v\n", req)

	if err := service.TenantUpdateConfiguration(uint64(tenantID), data); err != nil {
		logger.Error("UpdateTenantConfiguration service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)

	return
}

func modifyPWDByUID(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("modifyPWDByID param ERR: ", err)
		return
	}

	uid, ok := req["uid"].(float64)
	if !ok || uint64(uid) <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("modifyPWDByUID param ERR: ", req)
		return
	}

	pwd, ok := req["pwd"].(string)
	pwd = strings.TrimSpace(pwd)
	if len(pwd) < 4 || len(pwd) > 16 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("modifyPWDByUID param ERR: ", req)
		return
	}

	logger.Infof("modifyPWDByUID %v %s\n", uid, pwd)

	if _, err := service.SetUserPWD(uint64(uid), 0, pwd); err != nil {
		logger.Errorf("modifyPWDByUID %v %s %s\n", uid, pwd, err.Error())
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
	return
}

package face

import (
	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/service"
	"net/http"
	"strings"
)

func UpdateTenantConfiguration(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param ERR: ", err)
		return
	}
	k, ok := req["k"].(string)
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param k nil")
		return
	}
	if len(strings.TrimSpace(k)) == 0 || len(strings.TrimSpace(k)) > 64 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param k nil")
		return
	}

	tenantID, ok := req["tenant_id"].(float64)
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Error("UpdateTenantConfiguration param tenantID nil")
		return
	}

	logger.Infof("UpdateTenantConfiguration : %v\n", req)

	if err := service.TenantUpdateConfiguration(uint64(tenantID), k, req["v"]); err != nil {
		logger.Error("UpdateTenantConfiguration service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)

	return
}

func modifyPWDByUID(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req); err != nil {
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

	if _, err := service.SetUserPWD(uint64(uid), pwd); err != nil {
		logger.Errorf("modifyPWDByUID %v %s %s\n", uid, pwd, err.Error())
		gocommon.HttpErr(w, http.StatusOK, -1, err.Error())
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
	return
}

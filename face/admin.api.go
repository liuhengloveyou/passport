package face

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"
)

/*
只有root租户的超级管理员登录，才能通过该接口添加租户和管理员
*/
func AdminTenantNew(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	// 只有root租户的超级管理员登录，才能通过该接口添加租户和管理员
	if sessionUser.TenantID != common.ServConfig.RootTenantID {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoAuth)
		return
	}

	req := &protos.NewTenantReq{}
	if err := readJsonBodyFromRequest(r, req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("TenantNew param ERR: ", err)
		return
	}
	logger.Sugar().Infof("TenantNew body: %#v\n", req)

	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantType = strings.TrimSpace(req.TenantType)
	req.Cellphone = strings.TrimSpace(req.Cellphone)
	req.Password = strings.TrimSpace(req.Password)
	if req.TenantName == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantNameNull)
		logger.Sugar().Error("TenantNew param ERR: ", req)
		return
	}
	if req.TenantType == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantTypeNull)
		logger.Sugar().Error("TenantNew param ERR: ", req)
		return
	}
	if req.TenantType == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantTypeNull)
		logger.Sugar().Error("TenantNew param ERR: ", req)
		return
	}
	if req.Cellphone == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrTenantAdminCellphoneNull)
		logger.Sugar().Error("TenantNew param ERR: ", req)
		return
	}

	uid, tenantID, err := service.AdminTenantNew(&sessionUser, req)
	if err != nil {
		logger.Sugar().Error("TenantNew service ERR: ", err)

		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	logger.Sugar().Info("TenantNew ok:", uid, tenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

func AdminSetParent(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	// 只有root租户的超级管理员登录，才能通过该接口设置租户的父租户
	if common.ServConfig.RootTenantID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		logger.Sugar().Error("AdminSetParent param ERR: ", sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	tenantID, _ := strconv.ParseUint(r.FormValue("tid"), 10, 64)
	parentID, _ := strconv.ParseUint(r.FormValue("pid"), 10, 64)

	if tenantID <= 0 || parentID <= 0 {
		logger.Sugar().Error("AdminSetParent param ERR: ", tenantID, parentID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	logger.Sugar().Infof("AdminSetParent : tenantID=%v, parentID=%v\n", tenantID, parentID)

	// 调用service设置parentID
	if err := service.AdminTenantSetParent(&sessionUser, tenantID, parentID); err != nil {
		logger.Sugar().Error("AdminSetParent service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

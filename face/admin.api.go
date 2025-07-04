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

// AdminTenantList 查询所有租户
// 只有root租户的用户才能查询所有租户
// 支持分页查询
func AdminTenantList(w http.ResponseWriter, r *http.Request) {
	// 获取会话用户信息
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	// 只有root租户的用户才能查询所有租户
	if common.ServConfig.RootTenantID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		common.Logger.Sugar().Error("AdminTenantList auth ERR: ", sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	// 解析分页参数
	page := uint64(1)
	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		var err error
		page, err = strconv.ParseUint(pageStr, 10, 64)
		if err != nil || page < 1 {
			page = 1
		}
	}

	pageSize := uint64(10)
	pageSizeStr := r.URL.Query().Get("page_size")
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.ParseUint(pageSizeStr, 10, 64)
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}
	}

	// 解析是否需要总数
	hasTotal := true
	hasTotalStr := r.URL.Query().Get("hasTotal")
	if hasTotalStr != "" {
		hasTotal = hasTotalStr == "1"
	}

	// 使用service层查询租户列表
	// 使用根租户ID(0)作为ancestorID参数，表示查询所有租户
	response, err := service.TenantTreeList(&sessionUser, 0, page, pageSize, hasTotal)
	if err != nil {
		if err == common.ErrNull {
			// 没有数据返回空列表
			gocommon.HttpJsonErr(w, http.StatusOK, protos.PageResponse{
				Total: 0,
				List:  []protos.Tenant{},
			})
			return
		}
		common.Logger.Sugar().Error("AdminTenantList service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrService)
		return
	}

	// 返回结果
	gocommon.HttpJsonErr(w, http.StatusOK, response)
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

	logger.Sugar().Infof("AdminSetParent : tenantID=%v, parentID=%v", tenantID, parentID)

	// 调用service设置parentID
	if err := service.AdminTenantSetParent(&sessionUser, tenantID, parentID); err != nil {
		logger.Sugar().Error("AdminSetParent service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// AdminTenantUpdateConfig 更新租户配置
func AdminTenantUpdateConfig(w http.ResponseWriter, r *http.Request) {
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

	// 解析请求体
	req := &protos.UpdateTenantConfigReq{}
	if err := readJsonBodyFromRequest(r, req, 10240); err != nil {
		logger.Sugar().Error("AdminTenantUpdateConfig param ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	// 参数校验
	if req.TenantID <= 0 || req.Configuration == nil || req.LastUpdateTime == "" {
		logger.Sugar().Error("AdminTenantUpdateConfig param ERR: ", req)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}

	// 调用服务层更新租户配置
	if err := service.AdminTenantUpdateConfig(&sessionUser, req); err != nil {
		logger.Sugar().Error("AdminTenantUpdateConfig service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	// 返回成功响应
	logger.Sugar().Infof("AdminTenantUpdateConfig success: tenant_id=%d", req.TenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

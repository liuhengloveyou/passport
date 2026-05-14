// admin_tenant.go 提供平台管理员租户管理接口。
package admin

import (
	"net/http"
	"strconv"
	"strings"

	gocommon "github.com/liuhengloveyou/go-common"
	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	"go.uber.org/zap"
)

func authorizeAdminTenant(sessionUser protos.User, action string, targetTenantID uint64) error {
	if sessionUser.UID <= 0 || sessionUser.TenantID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		core.Logger().Error(action+" auth ERR: ", zap.Uint64("uid", sessionUser.UID), zap.Uint64("tenantID", sessionUser.TenantID))
		return common.ErrNoAuth
	}

	myTenant, err := service.AdminTenantTake(sessionUser.TenantID)
	if err != nil {
		core.Logger().Error(action+" get tenant ERR: ", zap.Uint64("tenantID", sessionUser.TenantID), zap.Error(err))
		return err
	}

	isRootAdmin := myTenant.UID == sessionUser.UID
	if !isRootAdmin {
		roles := accessctl.GetRoleForUserInDomain(sessionUser.UID, sessionUser.TenantID)
		core.Logger().Debug(action+" roles: ", zap.Uint64("uid", sessionUser.UID), zap.Uint64("tenantID", sessionUser.TenantID), zap.Any("roles", roles))
		for _, role := range roles {
			if role == "root" {
				isRootAdmin = true
				break
			}
		}
	}
	if !isRootAdmin {
		core.Logger().Error(action+" role auth ERR: ",
			zap.Uint64("uid", sessionUser.UID),
			zap.Uint64("tenantID", sessionUser.TenantID),
			zap.Uint64("targetTenantID", targetTenantID),
		)
		return common.ErrNoAuth
	}

	return nil
}

// AdminTenantNew 创建新租户；可同时在租户域内创建初始管理员账号（昵称+密码均提供时），
// 该账号在租户域内绑定 root；否则仅创建租户并由当前操作者暂为归属。
func AdminTenantNew(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	core.Logger().Info("AdminTenantNew start",
		zap.String("method", r.Method),
		zap.String("uri", r.RequestURI),
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
	)
	if err := authorizeAdminTenant(sessionUser, "AdminTenantNew", 0); err != nil {
		core.Logger().Error("AdminTenantNew auth ERR: ", zap.Uint64("uid", sessionUser.UID), zap.Uint64("tenantID", sessionUser.TenantID), zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}
	req := &protos.NewTenantReq{}
	if err := core.ReadJSONBodyFromRequest(r, req, 1024); err != nil {
		core.Logger().Error("AdminTenantNew read JSON body ERR: ", zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantType = strings.TrimSpace(req.TenantType)
	req.Cellphone = strings.TrimSpace(req.Cellphone)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Password = strings.TrimSpace(req.Password)
	wantAdmin := req.Nickname != "" || req.Password != "" || req.Cellphone != ""
	core.Logger().Info("AdminTenantNew parsed request",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.String("tenant_name", req.TenantName),
		zap.String("tenant_type", req.TenantType),
		zap.Uint64("parent_id", req.ParentID),
		zap.Bool("want_admin", wantAdmin),
		zap.Bool("has_cellphone", req.Cellphone != ""),
		zap.Bool("has_nickname", req.Nickname != ""),
	)
	if req.TenantName == "" || req.TenantType == "" {
		core.Logger().Warn("AdminTenantNew param ERR: tenantName or tenantType empty",
			zap.Uint64("operator_uid", sessionUser.UID),
			zap.Uint64("operator_tenant_id", sessionUser.TenantID),
			zap.String("tenant_name", req.TenantName),
			zap.String("tenant_type", req.TenantType),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if wantAdmin && ((req.Nickname == "" && req.Cellphone == "") || req.Password == "") {
		core.Logger().Warn("AdminTenantNew param ERR: invalid optional admin payload",
			zap.Uint64("operator_uid", sessionUser.UID),
			zap.Uint64("operator_tenant_id", sessionUser.TenantID),
			zap.Bool("has_cellphone", req.Cellphone != ""),
			zap.Bool("has_nickname", req.Nickname != ""),
			zap.Bool("has_password", req.Password != ""),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	newUID, newTenantID, err := service.AdminTenantNew(&sessionUser, req)
	if err != nil {
		core.Logger().Error("AdminTenantNew service ERR",
			zap.Uint64("operator_uid", sessionUser.UID),
			zap.Uint64("operator_tenant_id", sessionUser.TenantID),
			zap.String("tenant_name", req.TenantName),
			zap.Uint64("parent_id", req.ParentID),
			zap.Error(err),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	core.Logger().Info("AdminTenantNew success",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.Uint64("new_tenant_id", newTenantID),
		zap.Uint64("new_admin_uid", newUID),
		zap.Bool("want_admin", wantAdmin),
	)
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

// AdminTenantQuery 分页查询租户列表。
func AdminTenantQuery(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	core.Logger().Info("AdminTenantQuery start",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.String("method", r.Method),
		zap.String("uri", r.RequestURI),
	)
	if err := authorizeAdminTenant(sessionUser, "AdminTenantQuery", 0); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}
	tenantName := r.FormValue("name")
	cellphone := r.FormValue("cellphone")
	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	rr, e := service.AdminTenantQuery(tenantName, cellphone, page, pageSize, hasTotal == 1)
	if e != nil {
		core.Logger().Error("AdminTenantQuery ERR",
			zap.Uint64("operator_uid", sessionUser.UID),
			zap.Uint64("operator_tenant_id", sessionUser.TenantID),
			zap.String("tenant_name", tenantName),
			zap.String("cellphone", cellphone),
			zap.Uint64("page", page),
			zap.Uint64("page_size", pageSize),
			zap.Bool("has_total", hasTotal == 1),
			zap.Error(e),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		return
	}
	core.Logger().Info("AdminTenantQuery success",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.String("tenant_name", tenantName),
		zap.String("cellphone", cellphone),
		zap.Uint64("page", page),
		zap.Uint64("page_size", pageSize),
		zap.Bool("has_total", hasTotal == 1),
	)
	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

// AdminSetParent 设置租户父节点关系。
func AdminSetParent(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	tenantID, _ := strconv.ParseUint(r.FormValue("tid"), 10, 64)
	parentID, _ := strconv.ParseUint(r.FormValue("pid"), 10, 64)
	core.Logger().Info("AdminSetParent start",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.Uint64("tenant_id", tenantID),
		zap.Uint64("parent_id", parentID),
	)
	if tenantID <= 0 || parentID <= 0 {
		core.Logger().Error("AdminSetParent param ERR: ", zap.Uint64("tenantID", tenantID), zap.Uint64("parentID", parentID))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := authorizeAdminTenant(sessionUser, "AdminSetParent", tenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}

	if err := service.AdminTenantSetParent(&sessionUser, tenantID, parentID); err != nil {
		core.Logger().Error("AdminSetParent ERR: ", zap.Uint64("tenantID", tenantID), zap.Uint64("parentID", parentID), zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	core.Logger().Sugar().Infof("AdminSetParent: success, tenantID: %d, parentID: %d", tenantID, parentID)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// AdminTenantDelete 删除指定租户（不允许删除根租户）。
func AdminTenantDelete(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	tenantID, _ := strconv.ParseUint(r.FormValue("tid"), 10, 64)
	core.Logger().Info("AdminTenantDelete start",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.Uint64("tenant_id", tenantID),
	)
	if tenantID <= 0 || tenantID == common.ServConfig.RootTenantID {
		core.Logger().Error("AdminTenantDelete param ERR: ", zap.Uint64("tenantID", tenantID))
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := authorizeAdminTenant(sessionUser, "AdminTenantDelete", tenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}

	core.Logger().Sugar().Infof("AdminTenantDelete: tenantID: %d, uid: %d, tenantID: %d", tenantID, sessionUser.UID, sessionUser.TenantID)

	if err := service.AdminTenantDelete(&sessionUser, tenantID); err != nil {
		core.Logger().Error("AdminTenantDelete ERR: ", zap.Error(err))
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	core.Logger().Sugar().Infof("AdminTenantDelete: success, tenantID: %d, uid: %d, tenantID: %d", tenantID, sessionUser.UID, sessionUser.TenantID)
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// AdminUpdateTenantConfiguration 按租户 ID 更新配置数据。
func AdminUpdateTenantConfiguration(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	var req map[string]interface{}
	if err := core.ReadJSONBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	tenantID, ok := req["tenant_id"].(float64)
	data, ok2 := req["data"].(map[string]interface{})
	if !ok || !ok2 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := authorizeAdminTenant(sessionUser, "AdminUpdateTenantConfiguration", uint64(tenantID)); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}
	if err := service.TenantUpdateConfiguration(uint64(tenantID), data); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

// AdminTenantUpdateConfig 更新租户完整配置并校验更新时间。
func AdminTenantUpdateConfig(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	req := &protos.UpdateTenantConfigReq{}
	core.Logger().Info("AdminTenantUpdateConfig start",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.String("method", r.Method),
		zap.String("uri", r.RequestURI),
	)
	if err := core.ReadJSONBodyFromRequest(r, req, 10240); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.TenantID <= 0 || req.Configuration == nil || req.LastUpdateTime == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := authorizeAdminTenant(sessionUser, "AdminTenantUpdateConfig", req.TenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}
	if err := service.AdminTenantUpdateConfig(&sessionUser, req); err != nil {
		core.Logger().Error("AdminTenantUpdateConfig ERR",
			zap.Uint64("operator_uid", sessionUser.UID),
			zap.Uint64("operator_tenant_id", sessionUser.TenantID),
			zap.Uint64("tenant_id", req.TenantID),
			zap.Error(err),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	core.Logger().Info("AdminTenantUpdateConfig success",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.Uint64("tenant_id", req.TenantID),
	)
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

// AdminTenantUpdate 更新组织基础信息（名称、类型、info）。
func AdminTenantUpdate(w http.ResponseWriter, r *http.Request) {
	sessionUser := core.GetSessionUser(r)
	req := &protos.UpdateTenantReq{}
	core.Logger().Info("AdminTenantUpdate start",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.String("method", r.Method),
		zap.String("uri", r.RequestURI),
	)
	if err := core.ReadJSONBodyFromRequest(r, req, 10240); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if req.TenantID <= 0 || strings.TrimSpace(req.TenantName) == "" || strings.TrimSpace(req.TenantType) == "" {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		return
	}
	if err := authorizeAdminTenant(sessionUser, "AdminTenantUpdate", req.TenantID); err != nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, err)
		return
	}
	req.TenantName = strings.TrimSpace(req.TenantName)
	req.TenantType = strings.TrimSpace(req.TenantType)
	if err := service.AdminTenantUpdate(&sessionUser, req); err != nil {
		core.Logger().Error("AdminTenantUpdate ERR",
			zap.Uint64("operator_uid", sessionUser.UID),
			zap.Uint64("operator_tenant_id", sessionUser.TenantID),
			zap.Uint64("tenant_id", req.TenantID),
			zap.String("tenant_name", req.TenantName),
			zap.String("tenant_type", req.TenantType),
			zap.Error(err),
		)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}
	core.Logger().Info("AdminTenantUpdate success",
		zap.Uint64("operator_uid", sessionUser.UID),
		zap.Uint64("operator_tenant_id", sessionUser.TenantID),
		zap.Uint64("tenant_id", req.TenantID),
	)
	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

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
如果是root租户，可以查所有。
如果是其它租户，只能查后代。
*/
func TenantTreeList(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	r.ParseForm()
	parentID, _ := strconv.ParseUint(r.FormValue("pid"), 10, 64)
	hasTotal, _ := strconv.ParseInt(r.FormValue("hasTotal"), 10, 64)
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

	// parentID为0时，默认查询RootTenant的直接子租户
	if parentID == 0 {
		parentID = sessionUser.TenantID
	}

	common.Logger.Sugar().Debugf("TenantTreeList params: parentID=%d, page=%d, pageSize=%d, hasTotal=%d\n", parentID, page, pageSize, hasTotal)
	rr, e := service.TenantTreeList(&sessionUser, parentID, page, pageSize, hasTotal == 1)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		logger.Sugar().Error("TenantList ERR: ", e)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

func AdminTenantQuery(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	// 只有root租户的超级管理员登录，才能通过该接口添加租户和管理员
	if common.ServConfig.RootTenantID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		logger.Sugar().Error("AdminTenantList param ERR: ", sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoAuth)
		return
	}

	r.ParseForm()

	// 获取查询参数
	tenantName := r.FormValue("name")
	cellphone := r.FormValue("cellphone")
	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)

	// 设置默认值
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	logger.Sugar().Infof("AdminTenantQuery params: tenantName=%s, cellphone=%s, page=%d, pageSize=%d, hasTotal=%d",
		tenantName, cellphone, page, pageSize, hasTotal)

	// 调用服务层函数进行查询
	rr, e := service.AdminTenantQuery(tenantName, cellphone, page, pageSize, hasTotal == 1)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		logger.Sugar().Error("AdminTenantQuery ERR: ", e)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

func UpdateTenantConfiguration(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("UpdateTenantConfiguration param ERR: ", err)
		return
	}

	tenantID, ok := req["tenant_id"].(float64)
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("UpdateTenantConfiguration param tenantID nil")
		return
	}

	data, ok := req["data"].(map[string]interface{})
	if !ok {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("UpdateTenantConfiguration param data nil")
		return
	}

	logger.Sugar().Infof("UpdateTenantConfiguration : %v\n", req)

	if err := service.TenantUpdateConfiguration(uint64(tenantID), data); err != nil {
		logger.Sugar().Error("UpdateTenantConfiguration service ERR: ", err)
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpJsonErr(w, http.StatusOK, common.ErrOK)
}

func modifyPWDByUID(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := readJsonBodyFromRequest(r, &req, 1024); err != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("modifyPWDByID param ERR: ", err)
		return
	}

	uid, ok := req["uid"].(float64)
	if !ok || uint64(uid) <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("modifyPWDByUID param ERR: ", req)
		return
	}

	pwd, ok := req["pwd"].(string)
	pwd = strings.TrimSpace(pwd)
	if len(pwd) < 4 || len(pwd) > 16 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("modifyPWDByUID param ERR: ", req)
		return
	}

	logger.Sugar().Infof("modifyPWDByUID %v %s\n", uid, pwd)

	if _, err := service.SetUserPWD(uint64(uid), 0, pwd); err != nil {
		logger.Sugar().Errorf("modifyPWDByUID %v %s %s\n", uid, pwd, err.Error())
		gocommon.HttpJsonErr(w, http.StatusOK, err)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, "OK")
}

func TenantGetRole(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		logger.Sugar().Error("GetRole TenantID ERR")
		return
	}

	roles := service.TenantGetRole(sessionUser.TenantID)
	gocommon.HttpErr(w, http.StatusOK, 0, roles)
}

func AdminUserList(w http.ResponseWriter, r *http.Request) {
	sessionUser := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey].(protos.User)
	if sessionUser.TenantID <= 0 {
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrNoAuth)
		return
	}

	// 只有root租户的超级管理员登录，才能通过该接口查看用户列表
	if common.ServConfig.RootTenantID <= 0 || sessionUser.TenantID != common.ServConfig.RootTenantID {
		logger.Sugar().Error("AdminUserList param ERR: ", sessionUser.TenantID)
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoAuth)
		return
	}

	r.ParseForm()

	hasTotal, _ := strconv.ParseUint(r.FormValue("hasTotal"), 10, 64)
	page, _ := strconv.ParseUint(r.FormValue("page"), 10, 64)
	pageSize, _ := strconv.ParseUint(r.FormValue("pageSize"), 10, 64)
	tenantID, _ := strconv.ParseUint(r.FormValue("tenantID"), 10, 64)
	nickname := strings.TrimSpace(r.FormValue("nickname"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	// 如果没有指定租户ID，默认查询所有租户的用户（设置为0表示查询所有）
	if tenantID == 0 {
		// 这里可以根据业务需求决定是否允许查询所有租户的用户
		// 暂时设置一个默认值或返回错误
		gocommon.HttpJsonErr(w, http.StatusOK, common.ErrParam)
		logger.Sugar().Error("AdminUserList: tenantID is required")
		return
	}

	rr, e := service.AdminUserList(tenantID, page, pageSize, hasTotal == 1, nickname)
	if e != nil {
		gocommon.HttpJsonErr(w, http.StatusOK, e)
		logger.Sugar().Error("AdminUserList ERR: ", e)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, rr)
}

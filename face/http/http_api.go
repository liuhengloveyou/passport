package http

import (
	"context"
	"crypto/md5"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	faceAccess "github.com/liuhengloveyou/passport/v3/face/access"
	faceAdmin "github.com/liuhengloveyou/passport/v3/face/admin"
	"github.com/liuhengloveyou/passport/v3/face/core"
	faceSms "github.com/liuhengloveyou/passport/v3/face/sms"
	faceTenant "github.com/liuhengloveyou/passport/v3/face/tenant"
	"github.com/liuhengloveyou/passport/v3/face/user"
	faceWx "github.com/liuhengloveyou/passport/v3/face/wx"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"
)

type Api struct {
	Handler    func(http.ResponseWriter, *http.Request)
	NeedLogin  bool
	NeedAccess bool
}

var (
	apis         map[string]Api
	sessionStore sessions.Store
	logger       *zap.Logger
)

func init() {
	apis = map[string]Api{
		// 用户相关接口
		"user/register":          {Handler: user.UserAdd},
		"user/login":             {Handler: user.UserLogin},
		"user/auth":              {Handler: user.UserAuth, NeedLogin: true},
		"user/logout":            {Handler: user.UserLogout, NeedLogin: true},
		"user/info":              {Handler: user.UserInfo, NeedLogin: true},
		"user/infoByUID":         {Handler: user.UserInfoByUID, NeedLogin: true},
		"user/modify":            {Handler: user.UserModify, NeedLogin: true},
		"user/modify/password":   {Handler: user.UserModifyPassword, NeedLogin: true},
		"user/modify/getbackpwd": {Handler: user.UserGetBackPassword},
		"user/modify/avatarForm": {Handler: user.UserModifyAvatarForm, NeedLogin: true},
		"user/s/1":               {Handler: user.UserSearchLite},

		// 权限与访问控制接口
		"access/addRoleForUser":       {Handler: faceAccess.AddRoleForUser, NeedLogin: true, NeedAccess: true},
		"access/updateRoleForUser":    {Handler: faceAccess.UpdateRoleForUser, NeedLogin: true, NeedAccess: true},
		"access/removeRoleForUser":    {Handler: faceAccess.RemoveRoleForUser, NeedLogin: true, NeedAccess: true},
		"access/getRolesForMe":        {Handler: faceAccess.GetRolesForMe, NeedLogin: true},
		"access/getRolesForUser":      {Handler: faceAccess.GetRolesForUser, NeedLogin: true, NeedAccess: true},
		"access/getUsersForRole":      {Handler: faceAccess.GetUsersForRole, NeedLogin: true, NeedAccess: true},
		"access/addPolicyToRole":      {Handler: faceAccess.AddPolicyToRole, NeedLogin: true, NeedAccess: true},
		"access/removePolicyFromRole": {Handler: faceAccess.RemovePolicyFromRole, NeedLogin: true, NeedAccess: true},
		"access/getPolicy":            {Handler: faceAccess.GetPolicy, NeedLogin: true, NeedAccess: true},
		"access/getPolicyForUser":     {Handler: faceAccess.GetPolicyForUser, NeedLogin: true},
		"access/createPermission":     {Handler: faceAccess.PermissionCreate, NeedLogin: true, NeedAccess: true},
		"access/deletePermission":     {Handler: faceAccess.PermissionDelete, NeedLogin: true, NeedAccess: true},
		"access/listPermission":       {Handler: faceAccess.PermissionList, NeedLogin: true, NeedAccess: true},

		// 租户与组织结构接口
		"tenant/add":                  {Handler: faceTenant.Add, NeedLogin: true},
		"tenant/user/add":             {Handler: faceTenant.UserAdd, NeedLogin: true, NeedAccess: true},
		"tenant/delUser":              {Handler: faceTenant.UserDel, NeedLogin: true, NeedAccess: true},
		"tenant/getUsers":             {Handler: faceTenant.UserGet, NeedLogin: true, NeedAccess: true},
		"tenant/userDisableByUID":     {Handler: faceTenant.UserDisableByUID, NeedLogin: true, NeedAccess: true},
		"tenant/userModifyExtInfo":    {Handler: faceTenant.UserModifyExtInfo, NeedLogin: true, NeedAccess: true},
		"tenant/modifyUserPassword":   {Handler: faceTenant.ModifyUserPassword, NeedLogin: true, NeedAccess: true},
		"tenant/user/setDepartment":   {Handler: faceTenant.UserSetDepartment, NeedLogin: true, NeedAccess: true},
		"tenant/addRole":              {Handler: faceTenant.AddRole, NeedLogin: true, NeedAccess: true},
		"tenant/delRole":              {Handler: faceTenant.DelRole, NeedLogin: true, NeedAccess: true},
		"tenant/getRoles":             {Handler: faceTenant.GetRole, NeedLogin: true, NeedAccess: true},
		"tenant/updateConfiguration":  {Handler: faceTenant.UpdateConfiguration, NeedLogin: true, NeedAccess: true},
		"tenant/loadConfiguration":    {Handler: faceTenant.LoadConfiguration, NeedLogin: true},
		"tenant/tree/list":            {Handler: faceTenant.TreeList, NeedLogin: true},
		"tenant/department/add":       {Handler: faceTenant.DepartmentAdd, NeedLogin: true, NeedAccess: true},
		"tenant/department/delete":    {Handler: faceTenant.DepartmentDelete, NeedLogin: true, NeedAccess: true},
		"tenant/department/update":    {Handler: faceTenant.DepartmentUpdate, NeedLogin: true, NeedAccess: true},
		"tenant/department/updatecfg": {Handler: faceTenant.DepartmentUpdateConfig, NeedLogin: true, NeedAccess: true},
		"tenant/department/list":      {Handler: faceTenant.DepartmentList, NeedLogin: true},

		// SAAS平台管理员接口
		"admin/tenant/new":                {Handler: faceAdmin.AdminTenantNew, NeedLogin: true, NeedAccess: false},
		"admin/user/list":                 {Handler: faceAdmin.UserList, NeedLogin: true, NeedAccess: false},
		"admin/user/del":                  {Handler: faceAdmin.UserDel, NeedLogin: true, NeedAccess: false},
		"admin/user/edit":                 {Handler: faceAdmin.UserEdit, NeedLogin: true, NeedAccess: false},
		"admin/tenant/query":              {Handler: faceAdmin.AdminTenantQuery, NeedLogin: true, NeedAccess: false},
		"admin/tenant/setParent":          {Handler: faceAdmin.AdminSetParent, NeedLogin: true, NeedAccess: false},
		"admin/tenant/delete":             {Handler: faceAdmin.AdminTenantDelete, NeedLogin: true, NeedAccess: false},
		"admin/tenant/update":             {Handler: faceAdmin.AdminTenantUpdate, NeedLogin: true, NeedAccess: false},
		"admin/updateTenantConfiguration": {Handler: faceAdmin.AdminUpdateTenantConfiguration, NeedLogin: true, NeedAccess: false},
		"admin/modifyUserPassword":        {Handler: faceAdmin.ModifyUserPassword, NeedLogin: true, NeedAccess: false},
		"admin/tenant/update_config":      {Handler: faceAdmin.AdminTenantUpdateConfig, NeedLogin: true, NeedAccess: false},

		// 短信验证码接口
		"sms/sendUserAddSmsCode": {Handler: faceSms.SendUserAddSmsCode},
		"sms/sendUserLoginSms":   {Handler: faceSms.SendUserLoginSms},
		"sms/sendGetBackPwdSms":  {Handler: faceSms.SendGetBackPwdSms},
		"sms/sendWxBindSms":      {Handler: faceSms.SendWxBindSms},

		// 微信相关接口
		"wx/bindCellphone":      {Handler: faceWx.WxMpBindCellphone, NeedLogin: true},
		"wx/miniapp/updateInfo": {Handler: faceWx.WxMiniAppUserInfoUpdate, NeedLogin: true},
	}
}

func InitAndRunHttpApi(options *protos.OptionStruct) (handler http.Handler) {
	if options != nil {
		if err := common.InitWithOption(options); err != nil {
			panic(err)
		}
	}
	// configJSON, _ := json.MarshalIndent(common.ServConfig, "", "  ")
	// fmt.Printf("InitAndRunHttpApi:\n%s\n", string(configJSON))

	if common.ServConfig.DBDriver != "" && common.ServConfig.DBDSN != "" {
		if e := accessctl.InitAccessControl("rbac_with_domains_model.conf", common.ServConfig.DBDriver, common.ServConfig.DBDSN); e != nil {
			panic(e)
		}
	}

	logger = common.Logger
	core.SetLogger(logger)

	sessPWD := md5.Sum([]byte(common.SYS_PWD))
	sessionStore = sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
	sessionStore.(*sessions.CookieStore).MaxAge(common.ServConfig.SessionExpire)
	core.InitSessionStore(sessionStore)

	handler = &PassportHttpServer{}
	http.HandleFunc("/usercenter/wx/mpauth", faceWx.MpAuth)
	http.HandleFunc("/usercenter/wx/mini/login", faceWx.WxMiniAppLogin)
	http.Handle("/usercenter", handler)

	if common.ServConfig.Addr != "" {
		s := &http.Server{
			Addr:           common.ServConfig.Addr,
			ReadTimeout:    10 * time.Minute,
			WriteTimeout:   10 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		}
		if err := s.ListenAndServe(); err != nil {
			panic("ListenAndServe: " + err.Error())
		}
	}
	return
}

type PassportHttpServer struct{}

// GetSessionUser 返回当前请求中的会话用户信息。
// 对外封装在 face/http 包，避免外部项目直接依赖 core 包细节。
func GetSessionUser(r *http.Request) protos.User {
	return core.GetSessionUser(r)
}

func (p *PassportHttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		r.Header.Add("Access-Control-Allow-Origin", origin)
		r.Header.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		r.Header.Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, X-Extra-Header, Content-Type, Accept, Authorization")
		r.Header.Add("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		r.Header.Add("Access-Control-Allow-Credentials", "true")
		r.Header.Add("Access-Control-Max-Age", "86400")
		r.Header.Set("content-type", "application/json")
	}
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	apiName := r.Header.Get("X-API")
	if apiName == "" {
		URL, _ := url.Parse(r.RequestURI)
		apiName = URL.Path
	}
	if apiName == "" {
		gocommon.HttpErr(w, http.StatusMethodNotAllowed, -1, "?API")
		return
	}
	logger.Sugar().Infof("passport http api: %v %v %v\n", r.Method, apiName, r.URL)

	apiHandler, ok := apis[apiName]
	if !ok {
		logger.Warn("passport no found api", zap.String("apiName", apiName))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if apiHandler.NeedLogin {
		sess, auth := core.AuthFilter(r)
		if !auth && sess == nil {
			logger.Sugar().Errorf("passport http api no login: %v %v %v\n", r.Method, apiName, r.URL)
			gocommon.HttpErr(w, http.StatusUnauthorized, -1, "请登录")
			return
		} else if !auth && sess != nil {
			logger.Sugar().Infof("passport http api no access: %v %v %v\n", r.Method, apiName, r.URL)
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}
		r = r.WithContext(context.WithValue(context.Background(), "session", sess))
	}

	if apiHandler.NeedAccess {
		if !AccessFilter(r) {
			logger.Sugar().Errorf("passport http api no access: %v %v %v\n", r.Method, apiName, r.URL)
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}
	}

	logger.Sugar().Debugf("passport http api access: %v %v %v\n", r.Method, apiName, r.URL)

	apiHandler.Handler(w, r)
}

func AccessFilter(r *http.Request) bool {
	sess, err := core.SessionStore().Get(r, common.ServConfig.SessionKey)
	if err != nil {
		return false
	}
	sessUser := sess.Values[common.SessUserInfoKey].(protos.User)
	if sessUser.UID <= 0 {
		return false
	}
	obj := r.Header.Get("X-Requested-By")
	if obj == "" {
		obj = r.Header.Get("X-API")
	}
	if obj == "" {
		obj = r.RequestURI
	}
	if obj == "" {
		return false
	}
	if strings.HasPrefix(obj, "admin/") && sessUser.TenantID != common.ServConfig.RootTenantID {
		return false
	}
	needAccess := false
	if apiHandler, ok := apis[obj]; ok {
		needAccess = apiHandler.NeedAccess
	} else {
		apiConf, ok := common.ServConfig.ApiConf[obj]
		if !ok {
			if apiConf, ok = common.ServConfig.ApiConf["*"]; !ok {
				return false
			}
		}
		needAccess = apiConf.NeedAccess
	}
	if needAccess {
		access, err := accessctl.Enforce(sessUser.UID, sessUser.TenantID, obj, r.Method)
		if err != nil {
			return false
		}
		return access
	}
	return true
}

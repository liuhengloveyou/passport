package face

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	/*
		go tool pprof -http=:8080 http://127.0.0.1:10000/debug/pprof/profile
		/debug/pprof/profile：访问这个链接会自动进行 CPU profiling，持续 30s，并生成一个文件供下载
		/debug/pprof/block：Goroutine阻塞事件的记录。默认每发生一次阻塞事件时取样一次。
		/debug/pprof/goroutines：活跃Goroutine的信息的记录。仅在获取时取样一次。
		/debug/pprof/heap： 堆内存分配情况的记录。默认每分配512K字节时取样一次。
		/debug/pprof/mutex: 查看争用互斥锁的持有者。
		/debug/pprof/threadcreate: 系统线程创建情况的记录。 仅在获取时取样一次。
	*/
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/service"
	"github.com/liuhengloveyou/passport/sessions"

	"github.com/go-playground/validator/v10"
	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"
)

var (
	apis         map[string]Api
	sessionStore sessions.Store
	logger       *zap.Logger

	// 登录用户信息缓存
	loginUserCache sync.Map
	//  map[uint64]*protos.User

)

type Api struct {
	Handler    func(http.ResponseWriter, *http.Request)
	NeedLogin  bool
	NeedAccess bool
}

func init() {
	sessionStore = sessions.NewCookieStore([]byte(common.ServConfig.SessionKey))
	loginUserCache = sync.Map{}

	apis = map[string]Api{
		"user/register": {
			Handler: userAdd,
		},
		"user/login": {
			Handler: userLogin,
		},
		"user/auth": {
			Handler:   UserAuth,
			NeedLogin: true,
		},
		"user/logout": {
			Handler:   userLogout,
			NeedLogin: true,
		},
		"user/info": {
			Handler:   getMyInfo,
			NeedLogin: true,
		},
		"user/infoByUID": {
			Handler:   getInfoByUID,
			NeedLogin: true,
		},
		"user/modify": {
			Handler:   userModify,
			NeedLogin: true,
		},
		"user/modify/password": {
			Handler:   modifyPWD,
			NeedLogin: true,
		},
		"user/modify/getbackpwd": {
			Handler:   getbackPWD,
			NeedLogin: false,
		},
		"user/modify/avatarForm": {
			Handler:   modifyAvatarByForm,
			NeedLogin: true,
		},
		"user/s/1": {
			Handler:   searchLite,
			NeedLogin: false,
		},

		//访问控制
		"access/addRoleForUser": {
			Handler:    AddRoleForUser,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/updateRoleForUser": {
			Handler:    updateRoleForUser,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/removeRoleForUser": {
			Handler:    RemoveRoleForUser,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/getRolesForMe": {
			Handler:   GetRolesForMe,
			NeedLogin: true,
		},
		"access/getRolesForUser": {
			Handler:    GetRolesForUser,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/getUsersForRole": {
			Handler:    GetUsersForRole,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/addPolicyToRole": {
			Handler:    AddPolicyToRole,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/removePolicyFromRole": {
			Handler:    RemovePolicyFromRole,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/getPolicy": {
			Handler:    GetPolicy,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/getPolicyForUser": {
			Handler:   GetPolicyForUser,
			NeedLogin: true,
		},
		"access/createPermission": {
			Handler:    PermissionCreate,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/deletePermission": {
			Handler:    PermissionDelete,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/listPermission": {
			Handler:    PermissionList,
			NeedLogin:  true,
			NeedAccess: true,
		},

		// 多租户
		"tenant/add": {
			Handler:   TenantAdd,
			NeedLogin: true,
		},
		"tenant/user/add": {
			Handler:    TenantUserAdd,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/delUser": {
			Handler:    TenantUserDel,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/getUsers": {
			Handler:    TenantUserGet,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/userDisableByUID": {
			Handler:    TenantUserDisableByUID,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/userModifyExtInfo": {
			Handler:    TenantUserModifyExtInfo,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/modifyUserPassword": {
			Handler:    TenantModifyPWDByUID,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/user/setDepartment": {
			Handler:    TenantUserSetDepartment,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/addRole": {
			Handler:    TenantRoleAdd,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/delRole": {
			Handler:    TenantRoleDel,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/getRoles": {
			Handler:    TenantGetRole,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/updateConfiguration": {
			Handler:    UpdateConfiguration,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/loadConfiguration": {
			Handler:   LoadConfiguration,
			NeedLogin: true,
		},

		// 部门
		"tenant/department/add": {
			Handler:    addDepartment,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/department/delete": {
			Handler:    deleteDepartment,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/department/update": {
			Handler:    updateDepartment,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/department/updatecfg": {
			Handler:    updateDepartmentConfig,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/department/list": {
			Handler:   listDepartment,
			NeedLogin: true,
		},

		// 管理接口
		"admin/tenantList": {
			Handler:    TenantList,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"admin/updateTenantConfiguration": {
			Handler:    UpdateTenantConfiguration,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"admin/modifyUserPassword": {
			Handler:    modifyPWDByUID,
			NeedLogin:  true,
			NeedAccess: true,
		},

		// 短信
		"sms/sendUserAddSmsCode": {
			Handler:    SendUserAddSmsCode,
			NeedLogin:  false,
			NeedAccess: false,
		},
		"sms/sendUserLoginSms": {
			Handler:    SendUserLoginSms,
			NeedLogin:  false,
			NeedAccess: false,
		},
		"sms/sendGetBackPwdSms": {
			Handler:    SendGetBackPwdSms,
			NeedLogin:  false,
			NeedAccess: false,
		},
		"sms/sendWxBindSms": {
			Handler:    SendWxBindSms,
			NeedLogin:  false,
			NeedAccess: false,
		},

		// 微信
		"wx/bindCellphone": {
			Handler:    wxMpBindCellphone,
			NeedLogin:  true,
			NeedAccess: false,
		},
	}
}

func InitAndRunHttpApi(options *protos.OptionStruct) (handler http.Handler) {
	if options != nil {
		if err := common.InitWithOption(options); err != nil {
			panic(err)
		}
	}

	// common.InitWithOption 后面
	if e := accessctl.InitAccessControl("rbac_with_domains_model.conf", common.ServConfig.MysqlURN); e != nil {
		fmt.Println("InitAccessControl ERR: ", e)
	}

	logger = common.Logger

	sessPWD := md5.Sum([]byte(common.SYS_PWD))
	switch common.ServConfig.SessionStoreType {
	case "mem":
		// sessionStore = sessions.NewMemStore([]byte(common.SYS_PWD), sessPWD[:])
	default:
		sessionStore = sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
		sessionStore.(*sessions.CookieStore).MaxAge(common.ServConfig.SessionExpire)
	}

	handler = &PassportHttpServer{}

	// 微信公众号网页授权
	// https://developers.weixin.qq.com/doc/offiaccount/OA_Web_Apps/Wechat_webpage_authorization.html
	http.HandleFunc("/usercenter/wx/mpauth", mpAuth)
	http.HandleFunc("/usercenter/wx/mini/login", WxMiniAppLogin)

	http.Handle("/usercenter", handler)

	if common.ServConfig.Addr != "" {
		s := &http.Server{
			Addr:           common.ServConfig.Addr,
			ReadTimeout:    10 * time.Minute,
			WriteTimeout:   10 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		}

		fmt.Println("passport GO..." + common.ServConfig.Addr)
		if err := s.ListenAndServe(); err != nil {
			panic("ListenAndServe: " + err.Error())
		}
	}

	return
}

type PassportHttpServer struct {
}

func (p *PassportHttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin") //请求头部
	if origin != "" {
		r.Header.Add("Access-Control-Allow-Origin", origin)
		r.Header.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		r.Header.Add("Access-Control-Allow-Headers", "Origin, X-Requested-With, X-Extra-Header, Content-Type, Accept, Authorization")
		r.Header.Add("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		r.Header.Add("Access-Control-Allow-Credentials", "true")
		r.Header.Add("Access-Control-Max-Age", "86400")  // 可选
		r.Header.Set("content-type", "application/json") // 可选
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

	apiHandler, ok := apis[apiName]
	if !ok {
		logger.Warn("no found api: %v\n", zap.String("apiName", apiName))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// t1 := time.Now()
	if apiHandler.NeedLogin {
		sess, auth := AuthFilter(r)
		logger.Sugar().Debug("passport session:", sess, auth, apiName)

		if !auth && sess == nil {
			logger.Warn("session nil.", zap.String("api", apiName))
			gocommon.HttpErr(w, http.StatusUnauthorized, -1, "请登录")
			return
		} else if !auth && sess != nil {
			logger.Warn("auth false.")
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}

		r = r.WithContext(context.WithValue(context.Background(), "session", sess))

		// fmt.Println("passport:STATE: authFilter: ", time.Since(t1).Milliseconds())
	}

	if apiHandler.NeedAccess {
		if !AccessFilter(r) {
			logger.Warn("access false.")
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}
		// fmt.Println("passport:STATE: accessFilter: ", time.Since(t1).Milliseconds())
	}

	apiHandler.Handler(w, r)
	// fmt.Printf("passport:STATE: %v %v\n\n", apiName, time.Since(t1).Milliseconds())
}

func GetSessionUser(r *http.Request) (sessionUser protos.User) {
	sess, _ := AuthFilter(r)
	// logger.Debug("passport session:", sess, auth)
	if sess == nil {
		return
	}

	if sess.Values[common.SessUserInfoKey] == nil {
		return
	}

	if _, ok := sess.Values[common.SessUserInfoKey].(protos.User); !ok {
		return
	}

	return sess.Values[common.SessUserInfoKey].(protos.User)
}

func AuthFilter(r *http.Request) (sess *sessions.Session, auth bool) {
	var err error
	sess, err = sessionStore.Get(r, common.ServConfig.SessionKey)
	if err != nil {
		logger.Error("session ERR: ", zap.Error(err))
		return nil, false
	}

	if sess.Values[common.SessUserInfoKey] == nil {
		return nil, false
	}
	if _, ok := sess.Values[common.SessUserInfoKey].(protos.User); !ok {
		return nil, false
	}

	if sess.Values[common.SessUserInfoKey].(protos.User).UID <= 0 {
		return nil, false
	}

	// 是否已经禁用
	uid := sess.Values[common.SessUserInfoKey].(protos.User).UID
	userInfo, ok := loginUserCache.Load(uid)
	if userInfo == nil || !ok || time.Now().Unix()-userInfo.(*protos.User).CacheTime > 600 {
		userInfo, _ = service.GetUserInfo(uid)
	}
	if userInfo == nil {
		logger.Sugar().Warnf("AuthFilter: user %v not found\n", uid)
		return nil, false // 用户已经不存在
	}

	userInfo.(*protos.User).CacheTime = time.Now().Unix()
	loginUserCache.Store(uid, userInfo) // 缓存账号信息

	disabled, ok := userInfo.(*protos.User).Ext["disabled"].(float64)
	if ok && int8(disabled) == 1 {
		return nil, false // 账号已经禁用
	}

	return sess, true
}

func AccessFilter(r *http.Request) bool {
	sess, err := sessionStore.Get(r, common.ServConfig.SessionKey)
	if err != nil {
		logger.Error("AccessFilter sessionStore.Get ERR ", zap.Error(err))
		return false
	}
	sessUser := sess.Values[common.SessUserInfoKey].(protos.User)
	if sessUser.UID <= 0 {
		logger.Error("AccessFilter sessUser ERR: ", zap.Any("sess", sessUser))
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
		logger.Error("AccessFilter obj ERR")
		return false // 不知道需要访问什么资源
	}
	logger.Sugar().Debugf("AccessFilter obj: %v\n", obj)

	// 管理接口只有指定的租户可用
	if strings.HasPrefix(obj, "admin/") {
		if sessUser.TenantID != common.ServConfig.AdminTenantID {
			logger.Sugar().Warnf("only admin; obj: %v; %v; sess: %v\n", obj, common.ServConfig.AdminTenantID, sessUser)
			return false
		}
	}

	needAccess := false
	if apiHandler, ok := apis[obj]; ok {
		needAccess = apiHandler.NeedAccess
	} else {
		apiConf, ok := common.ServConfig.ApiConf[obj]
		if !ok {
			if apiConf, ok = common.ServConfig.ApiConf["*"]; !ok {
				logger.Error("AccessFilter conf ERR")
				return false
			}
		}

		needAccess = apiConf.NeedAccess
	}
	logger.Sugar().Debugf("AccessFilter needAccess: %v %v", obj, needAccess)

	if needAccess {
		access, err := accessctl.Enforce(sessUser.UID, sessUser.TenantID, obj, r.Method)
		logger.Sugar().Debugf("AccessFilter: %v %v %v %v %v\n", sessUser.UID, sessUser.TenantID, obj, r.Method, access)
		if err != nil {
			logger.Sugar().Errorf("AccessFilter Enforce ERR: %v\n", err)
			return false
		}

		return access
	}

	return true
}

func readJsonBodyFromRequest(r *http.Request, dst interface{}, bodyMaxLen int) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	logger.Debug("readJsonBodyFromRequest body: ", zap.ByteString("body", body))
	if len(body) >= bodyMaxLen {
		logger.Error("readJsonBodyFromRequest len ERR: ", zap.Int("body", len(body)), zap.Int("bodyMaxLen", bodyMaxLen))
		return common.ErrParam
	}

	if err = json.Unmarshal(body, dst); err != nil {
		return err
	}

	if err = common.Validate.Struct(dst); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); !ok {
			logger.Error("readJsonBodyFromRequest Validate ERR: ", zap.Error(err))
			return common.ErrParam
		}
	}

	return nil
}

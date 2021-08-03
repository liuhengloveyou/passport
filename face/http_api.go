package face

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"
)

var (
	apis         map[string]Api
	sessionStore sessions.Store
	logger       *zap.SugaredLogger
)

type Api struct {
	Handler    func(http.ResponseWriter, *http.Request)
	NeedLogin  bool
	NeedAccess bool
}

func init() {
	sessionStore = sessions.NewCookieStore([]byte(common.SessionKey))

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
		"user/modify/avatarForm": {
			Handler:   modifyAvatarByForm,
			NeedLogin: true,
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
		"access/getRolesForUser": {
			Handler:   GetRolesForUser,
			NeedLogin: true,
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
		"tenant/addUser": {
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
			Handler:    tenantModifyPWDByUID,
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
	}

	initWXAPI()
}

func InitAndRunHttpApi(options *protos.OptionStruct) (handler http.Handler) {
	if options != nil {
		if err := common.InitWithOption(options); err != nil {
			panic(err)
		}
	}

	// common.InitWithOption 后面
	if e := accessctl.InitAccessControl("rbac_with_domains_model.conf", common.ServConfig.MysqlURN); e != nil {
		panic(e)
	}
	logger = common.Logger.Sugar()

	sessPWD := md5.Sum([]byte(common.SYS_PWD))
	switch common.ServConfig.SessionStoreType {
	case "mem":
		sessionStore = sessions.NewMemStore([]byte(common.SYS_PWD), sessPWD[:])
	default:
		sessionStore = sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
	}
	sessionStore.(*sessions.CookieStore).MaxAge(common.ServConfig.SessionExpire)

	handler = &PassportHttpServer{}

	if common.ServConfig.Addr != "" {
		http.Handle("/usercenter", handler)
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
	apiName := r.Header.Get("X-API")
	logger.Debugf("passport api: %v", apiName)
	if apiName == "" {
		//gocommon.HttpErr(w, http.StatusBadRequest, -1, "?API")
		logger.Warnf("?API: %v", r.RemoteAddr)
		return
	}

	apiHandler, ok := apis[apiName]
	if !ok {
		logger.Warnf("no found api: %v\n", apiName)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if apiHandler.NeedLogin {
		sess, auth := AuthFilter(r)
		logger.Debug("passport session:", sess, auth)

		if auth == false && sess == nil {
			gocommon.HttpErr(w, http.StatusUnauthorized, -1, "请登录")
			return
		} else if auth == false && sess != nil {
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}

		r = r.WithContext(context.WithValue(context.Background(), "session", sess))
	}

	if apiHandler.NeedAccess {
		if false == AccessFilter(r) {
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}
	}

	apiHandler.Handler(w, r)
}

func GetSessionUser(r *http.Request) (sessionUser protos.User) {
	sess, auth := AuthFilter(r)
	logger.Debug("passport session:", sess, auth)
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

	sess, err = sessionStore.Get(r, common.SessionKey)
	if err != nil {
		logger.Error("session ERR: ", err)
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

	return sess, true
}

func AccessFilter(r *http.Request) bool {
	sess, err := sessionStore.Get(r, common.SessionKey)
	if err != nil {
		logger.Errorf("AccessFilter sessionStore.Get ERR: %v", err)
		return false
	}
	sessUser := sess.Values[common.SessUserInfoKey].(protos.User)
	if sessUser.UID <= 0 {
		logger.Errorf("AccessFilter sessUser ERR: %#v", sessUser)
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
		logger.Errorf("AccessFilter obj ERR")
		return false // 不知道需要访问什么资源
	}
	logger.Debugf("AccessFilter obj: %v\n", obj)

	// 管理接口只有指定的租户可用
	if strings.HasPrefix(obj, "admin/") {
		if sessUser.TenantID != common.ServConfig.AdminTenantID {
			logger.Warnf("only admin; obj: %v; %v; sess: %v", obj, common.ServConfig.AdminTenantID, sessUser)
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
	logger.Debugf("AccessFilter needAccess: %v %v", obj, needAccess)

	if needAccess {
		access, err := accessctl.Enforce(sessUser.UID, sessUser.TenantID, obj, r.Method)
		logger.Debugf("AccessFilter: %v %v %v %v %v\n", sessUser.UID, sessUser.TenantID, obj, r.Method, access)
		if err != nil {
			logger.Errorf("AccessFilter Enforce ERR: %v\n", err)
			return false
		}

		return access
	}

	return true
}

func readJsonBodyFromRequest(r *http.Request, dst interface{}, bodyMaxLen int) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	logger.Debugf("request body: '%v'\n", string(body))
	if len(body) >= bodyMaxLen {
		logger.Errorf("readJsonBodyFromRequest len ERR: %d %d\n", len(body), bodyMaxLen)
		return common.ErrParam
	}

	if err = json.Unmarshal(body, dst); err != nil {
		return err
	}

	if err = common.Validate.Struct(dst); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); !ok {
			logger.Errorf("readJsonBodyFromRequest Validate ERR: %v \n", err.Error())
			return common.ErrParam
		}
	}

	return nil
}

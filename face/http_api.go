package face

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

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
		"access/removeRoleForUser": {
			Handler:    RemoveRoleForUser,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"access/getRolesForUser": {
			Handler:    GetRolesForUser,
			NeedLogin:  true,
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
			Handler:    GetPolicyForUser,
			NeedLogin:  true,
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
			Handler:   TenantUserGet,
			NeedLogin: true,
		},
		"tenant/addRole": {
			Handler:    RoleAdd,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/getRoles": {
			Handler:   GetRole,
			NeedLogin: true,
		},
		"tenant/updateConfiguration": {
			Handler:    UpdateConfiguration,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"tenant/loadConfiguration": {
			Handler:    LoadConfiguration,
			NeedLogin:  true,
		},

		// 管理接口
		"admin/updateTenantConfiguration": {
			Handler:    UpdateTenantConfiguration,
			NeedLogin:  true,
			NeedAccess: true,
		},
		"admin/modifyUserPassword": {
			Handler:   modifyPWDByUID,
			NeedLogin: true,
		},
	}
}

func InitAndRunHttpApi(options *protos.OptionStruct) (handler http.Handler) {
	if options != nil {
		if err := common.InitWithOption(options); err != nil {
			panic(err)
		}
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

	if "" != options.Addr {
		http.Handle("/usercenter", handler)
		s := &http.Server{
			Addr:           options.Addr,
			ReadTimeout:    10 * time.Minute,
			WriteTimeout:   10 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		}

		fmt.Println("passport GO..." + options.Addr)
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
	logger.Debugf("passport api: %v\n", apiName)
	if apiName == "" {
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "?API")
		return
	}

	apiHandler, ok := apis[apiName]
	if !ok {
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

	uid := sess.Values[common.SessUserInfoKey].(protos.User).UID
	if uid <= 0 {
		return nil, false
	}

	return sess, true
}

func AccessFilter(r *http.Request) bool {
	var err error

	if common.ServConfig.AccessControl == false {
		return true
	}

	obj := r.Header.Get("X-API")
	if obj == "" {
		obj = r.Header.Get("X-DATA")
	}
	if obj == "" {
		obj = r.RequestURI
	}
	if obj == "" {
		return false // 不知道需要访问什么资源
	}

	sess, err := sessionStore.Get(r, common.SessionKey)
	if err != nil {
		return false
	}

	sessUser := sess.Values[common.SessUserInfoKey].(protos.User)
	if sessUser.UID <= 0 || sessUser.TenantID <= 0 {
		return false
	}

	logger.Debugf("obj: %v\n", obj)
	// 管理接口只有指定的租户可用
	if strings.HasPrefix(obj, "admin") {
		if sessUser.TenantID != common.ServConfig.AdminTenantID {
			logger.Warnf("obj: %v; user: %v; %v", obj, sessUser, common.ServConfig.AdminTenantID)
			return false
		}
	}

	access, err := accessctl.Enforce(sessUser.UID, sessUser.TenantID, obj, r.Method)
	logger.Debugf("AccessFilter: %v %v %v %v %v\n", sessUser.UID, sessUser.TenantID, obj, r.Method, access)
	if err != nil {
		panic(err)
	}

	return access
}

func GetUserInfoFromSession(w http.ResponseWriter, r *http.Request) (user protos.User) {
	sess, err := sessionStore.New(r, common.SessionKey)
	if err != nil {
		logger.Error("session ERR: ", err)
		return
	}

	if sess.Values[common.SessUserInfoKey] == nil {
		return
	}

	user, _ = sess.Values[common.SessUserInfoKey].(protos.User)

	return
}

func UserAuth(w http.ResponseWriter, r *http.Request) {
	sess, auth := AuthFilter(r)
	if auth == false || sess == nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}

	if sess.Values[common.SessUserInfoKey] == nil {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}
	if _, ok := sess.Values[common.SessUserInfoKey].(protos.User); !ok {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}
	uid := sess.Values[common.SessUserInfoKey].(protos.User).UID
	if uid <= 0 {
		gocommon.HttpJsonErr(w, http.StatusUnauthorized, common.ErrNoLogin)
		return
	}

	gocommon.HttpErr(w, http.StatusOK, 0, sess.Values[common.SessUserInfoKey].(protos.User))
	logger.Infof("auth OK: %#v", sess.Values[common.SessUserInfoKey].(protos.User))

	return
}

func readJsonBodyFromRequest(r *http.Request, dst interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	logger.Debugf("request body: '%v'\n", string(body))

	if err = json.Unmarshal(body, dst); err != nil {
		return err
	}

	if err = common.Validate.Struct(dst); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); !ok {
			return common.ErrParam
		}
	}

	return nil
}

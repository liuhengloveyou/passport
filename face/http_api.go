package face

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/liuhengloveyou/passport/accessctl"
	"io/ioutil"
	"net/http"
	"time"

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
			Handler:   userAuth,
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
		"access/role/add": {
			Handler:   AddRoleForUser,
			NeedLogin: true,
			NeedAccess: true,
		},
		"access/role/del": {
			Handler:   RemoveRoleForUser,
			NeedLogin: true,
			NeedAccess: true,
		},
		"access/policy/add": {
			Handler:   AddPolicy,
			NeedLogin: true,
			NeedAccess: true,
		},
		"access/policy/del": {
			Handler:   RemovePolicy,
			NeedLogin: true,
			NeedAccess: true,
		},

		// 多租户
		"tenant/add": {
			Handler:   TenantAdd,
			NeedLogin: true,
		},
		"tenant/role/get": {
			Handler:   GetRole,
			NeedLogin: true,
			NeedAccess: true,
		},
		"tenant/role/add": {
			Handler:   RoleAdd,
			NeedLogin: true,
			NeedAccess: true,
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

	handler = &PassportHttpServer{}

	if "" != options.Addr {
		http.Handle("/user", handler)
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
		gocommon.HttpErr(w, http.StatusMethodNotAllowed, 0, "")
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

	sess, err = sessionStore.New(r, common.SessionKey)
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
		return  false
	}

	sessUser := sess.Values[common.SessUserInfoKey].(protos.User)
	if sessUser.UID <= 0 || sessUser.TenantID <= 0 {
		return false
	}

	access, err := accessctl.Enforce(sessUser.UID, sessUser.TenantID, obj, r.Method)
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

func userAuth(w http.ResponseWriter, r *http.Request) {
	userInfo := r.Context().Value("session").(*sessions.Session).Values[common.SessUserInfoKey]
	gocommon.HttpErr(w, http.StatusOK, 0, userInfo)
	logger.Infof("auth OK: %#v", userInfo)
	return
}

func readJsonBodyFromRequest(r *http.Request, dst interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, dst); err != nil {
		return err
	}

	if err = validator.New().Struct(dst); err != nil {
		logger.Errorf("readJsonBodyFromRequest validator ERR: ", err)
		return common.ErrParam
	}

	return nil
}

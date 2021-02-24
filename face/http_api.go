package face

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sessions"

	gocommon "github.com/liuhengloveyou/go-common"
	"go.uber.org/zap"
)

const (
	SessionKey      = "goSessionID"
	SessUserInfoKey = "sessionUserInfo"
	MAX_UPLOAD_LEN  = (5 * 1024 * 1024) // 最大上传文件大小
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
	sessionStore = sessions.NewCookieStore([]byte(SessionKey))

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
		"modify/password": {
			Handler:   modifyPWD,
			NeedLogin: true,
		},
		"modify/avatarForm": {
			Handler:   modifyAvatarByForm,
			NeedLogin: true,
		},

		//访问控制
		"role/add": {
			Handler:   AddRoleForUser,
			NeedLogin: true,
		},
		"role/del": {
			Handler:   DeleteRoleForUser,
			NeedLogin: true,
		},
		"policy/add": {
			Handler:   AddPolicy,
			NeedLogin: true,
		},
		"policy/del": {
			Handler:   RemovePolicy,
			NeedLogin: true,
		},

		// 多租户
		"tenant/add": {
			Handler:   tenantAdd,
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

	switch common.ServConfig.SessionStoreType {
	case "mem":
		sessionStore = sessions.NewMemStore(
			[]byte(common.SYS_PWD),
		)
	default:
		sessionStore = sessions.NewCookieStore([]byte(SessionKey))
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
		gocommon.HttpErr(w, http.StatusNotFound, 0, "")
		return
	}

	if apiHandler.NeedLogin {
		sess, auth := AuthFilter(w, r)
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

	apiHandler.Handler(w, r)
}

func AuthFilter(w http.ResponseWriter, r *http.Request) (sess *sessions.Session, auth bool) {
	var err error

	sess, err = sessionStore.New(r, SessionKey)
	if err != nil {
		logger.Error("session ERR: ", err)
		return nil, false
	}

	if sess.Values[SessUserInfoKey] == nil {
		return nil, false
	}

	if _, ok := sess.Values[SessUserInfoKey].(protos.User); !ok {
		return nil, false
	}

	uid := sess.Values[SessUserInfoKey].(protos.User).UID

	// 数据库里有这用户吗 @@@
	one, e := dao.UserSelect(&protos.UserReq{UID: uid}, 1, 1)
	if e != nil {
		panic(e)
	}
	if len(one) == 0 {
		return nil, false
	}
	if one[0].UID != uid {
		return nil, false
	}

	// 权限验证
	// if common.ServConfig.AccessControl == true &&  {
	// 	obj := r.Header.Get("X-API")
	// 	if obj == "" {
	// 		obj = r.Header.Get("X-DATA")
	// 	}
	// 	if obj == "" {
	// 		return sess, false // 不知道需要访问什么资源
	// 	}

	// 	access, err := accessctl.Enforce(uid, obj, "access")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	if access == false {
	// 		return sess, false
	// 	}
	// }

	return sess, true
}

func GetUserInfoFromSession(w http.ResponseWriter, r *http.Request) (user protos.User) {
	sess, err := sessionStore.New(r, SessionKey)
	if err != nil {
		logger.Error("session ERR: ", err)
		return
	}

	if sess.Values[SessUserInfoKey] == nil {
		return
	}

	user, _ = sess.Values[SessUserInfoKey].(protos.User)

	return
}

func userAuth(w http.ResponseWriter, r *http.Request) {
	userInfo := r.Context().Value("session").(*sessions.Session).Values[SessUserInfoKey]
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

	return nil
}

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
	store  = sessions.NewCookieStore([]byte(SessionKey))
	logger *zap.SugaredLogger
)

func InitAndRunHttpApi(options *protos.OptionStruct) (handler http.Handler) {
	if options != nil {
		if err := common.InitWithOption(options); err != nil {
			panic(err)
		}
	}
	logger = common.Logger.Sugar()

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
	api := r.Header.Get("X-API")
	logger.Debugf("passport api: %v\n", api)
	if api == "" {
		gocommon.HttpErr(w, http.StatusBadRequest, -1, "?API")
		return
	}

	if api != "login" && api != "register" && api != "weapp/login" {
		sess, auth := AuthFilter(w, r)
		logger.Debug("passport session:", sess, auth)

		if auth == false && sess == nil {
			gocommon.HttpErr(w, http.StatusUnauthorized, -1, "请登录")
			return
		} else if auth == false && sess != nil {
			gocommon.HttpErr(w, http.StatusForbidden, -1, "您没有权限")
			return
		}

		if sess != nil {
			r = r.WithContext(context.WithValue(context.Background(), "session", sess))
		}
	}

	switch api {
	case "register":
		userAdd(w, r)
	case "login":
		userLogin(w, r)
	case "auth":
		userAuth(w, r)
	case "logout":
		userLogout(w, r)
	case "modify":
		userModify(w, r)
	case "modify/password":
		modifyPWD(w, r)
	case "modify/avatarForm":
		modifyAvatarByForm(w, r)
	case "info":
		getMyInfo(w, r)
	case "role/add":
		AddRoleForUser(w, r)
	case "role/del":
		DeleteRoleForUser(w, r)
	case "policy/add":
		AddPolicy(w, r)
	case "policy/del":
		RemovePolicy(w, r)
	default:
		gocommon.HttpErr(w, http.StatusNotFound, 0, "")
		return
	}
}

func AuthFilter(w http.ResponseWriter, r *http.Request) (sess *sessions.Session, auth bool) {
	var err error

	sess, err = store.New(r, SessionKey)
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

	// 数据库里有这用户吗
	one, _ := dao.UserSelect(&protos.UserReq{UID: sess.Values[SessUserInfoKey].(protos.User).UID}, 1, 100)
	if len(one) == 0 {
		return nil, false
	}
	if one[0].UID != sess.Values[SessUserInfoKey].(protos.User).UID {
		return nil, false
	}

	// 权限验证
	if common.ServConfig.AccessControl {

	}

	return sess, true
}

func GetUserInfoFromSession(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	sess, err := store.New(r, SessionKey)
	if err != nil {
		logger.Error("session ERR: ", err)
		return nil
	}

	if sess.Values[SessUserInfoKey] == nil {
		return nil
	}

	infoUser, ok := sess.Values[SessUserInfoKey].(protos.User)
	if !ok {
		return nil
	}

	info := make(map[string]interface{}, 1)
	info["uid"] = infoUser.UID
	if infoUser.Cellphone != nil && infoUser.Cellphone.Valid {
		info["cellphone"] = infoUser.Cellphone.String
	}
	if infoUser.Email != nil && infoUser.Email.Valid {
		info["email"] = infoUser.Email.String
	}
	if infoUser.Nickname != nil && infoUser.Nickname.Valid {
		info["nickname"] = infoUser.Nickname.String
	}
	if infoUser.AvatarURL != nil && infoUser.AvatarURL.Valid {
		info["avatar_url"] = infoUser.AvatarURL.String
	}
	if infoUser.Gender != nil && infoUser.Gender.Valid {
		info["gender"] = infoUser.Gender.Int64
	}
	if infoUser.Addr != nil && infoUser.Addr.Valid {
		info["addr"] = infoUser.Addr.String
	}

	return info
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

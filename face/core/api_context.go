package core

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/service"
	"github.com/liuhengloveyou/passport/v3/sessions"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger

	// 登录用户信息缓存
	loginUserCache sync.Map
	sessionStore   sessions.Store
)

func InitSessionStore(store sessions.Store) {
	sessionStore = store
}

func SessionStore() sessions.Store {
	return sessionStore
}

func SetLogger(l *zap.Logger) {
	logger = l
}

func Logger() *zap.Logger {
	if logger != nil {
		return logger
	}
	return common.Logger
}

func GetSessionUser(r *http.Request) (sessionUser protos.User) {
	sess, _ := AuthFilter(r)
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
		Logger().Error("session ERR: ", zap.Error(err))
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

	uid := sess.Values[common.SessUserInfoKey].(protos.User).UID
	userInfo, ok := loginUserCache.Load(uid)
	if userInfo == nil || !ok || time.Now().Unix()-userInfo.(*protos.User).CacheTime > 600 {
		userInfo, _ = service.GetUserInfo(uid)
	}
	if userInfo == nil {
		Logger().Sugar().Warnf("AuthFilter: user %v not found\n", uid)
		return nil, false
	}

	userInfo.(*protos.User).CacheTime = time.Now().Unix()
	loginUserCache.Store(uid, userInfo)

	disabled, ok := userInfo.(*protos.User).Ext["disabled"].(float64)
	if ok && protos.UserDisableStatus(int8(disabled)) == protos.UserDisabled {
		return nil, false
	}
	return sess, true
}

func ReadJSONBodyFromRequest(r *http.Request, dst interface{}, bodyMaxLen int) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	Logger().Debug("readJsonBodyFromRequest body: ", zap.ByteString("body", body))
	if len(body) >= bodyMaxLen {
		Logger().Error("readJsonBodyFromRequest len ERR: ", zap.Int("body", len(body)), zap.Int("bodyMaxLen", bodyMaxLen))
		return common.ErrParam
	}

	if err = json.Unmarshal(body, dst); err != nil {
		return err
	}

	if err = common.Validate.Struct(dst); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); !ok {
			Logger().Error("readJsonBodyFromRequest Validate ERR: ", zap.Error(err))
			return common.ErrParam
		}
	}
	return nil
}

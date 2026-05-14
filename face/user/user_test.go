package user

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/face/core"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/sessions"
)

var initOnce sync.Once

func initUserTests() {
	initOnce.Do(func() {
		core.SetLogger(common.Logger)
		sessPWD := md5.Sum([]byte(common.SYS_PWD))
		store := sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
		store.MaxAge(common.ServConfig.SessionExpire)
		core.InitSessionStore(store)
	})
}

func uniqueCellphone() string {
	return "13" + time.Now().Format("150405000")
}

func createUser(t *testing.T, cellphone string, password string) {
	t.Helper()
	reqBody, _ := json.Marshal(&protos.UserReq{Cellphone: cellphone, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	UserAdd(w, req)
}

func loginUser(t *testing.T, cellphone string, password string) *httptest.ResponseRecorder {
	t.Helper()
	reqBody, _ := json.Marshal(&protos.UserReq{Cellphone: cellphone, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	UserLogin(w, req)
	return w
}

func TestUserRegisterAndLogin(t *testing.T) {
	initUserTests()
	cellphone := uniqueCellphone()
	createUser(t, cellphone, "123456")

	w := loginUser(t, cellphone, "123456")
	var result map[string]interface{}
	_ = json.NewDecoder(w.Result().Body).Decode(&result)
	if code, _ := result["code"].(float64); code != 0 {
		t.Fatalf("登录失败: %+v", result)
	}
}

func TestUserModifyAndInfo(t *testing.T) {
	initUserTests()
	cellphone := uniqueCellphone()
	createUser(t, cellphone, "123456")
	loginW := loginUser(t, cellphone, "123456")

	reqBody, _ := json.Marshal(map[string]interface{}{"nickname": "n1"})
	modifyReq := httptest.NewRequest(http.MethodPost, "/user/modify", bytes.NewBuffer(reqBody))
	for _, c := range loginW.Result().Cookies() {
		modifyReq.AddCookie(c)
	}
	modifyW := httptest.NewRecorder()
	UserModify(modifyW, modifyReq)

	infoReq := httptest.NewRequest(http.MethodGet, "/user/info", nil)
	for _, c := range loginW.Result().Cookies() {
		infoReq.AddCookie(c)
	}
	infoW := httptest.NewRecorder()
	UserInfo(infoW, infoReq)

	var result map[string]interface{}
	_ = json.NewDecoder(infoW.Result().Body).Decode(&result)
	if code, _ := result["code"].(float64); code != 0 {
		t.Fatalf("获取信息失败: %+v", result)
	}
}

package admin

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
	faceuser "github.com/liuhengloveyou/passport/v3/face/user"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/sessions"
)

var adminInitOnce sync.Once

func initAdminTests() {
	adminInitOnce.Do(func() {
		core.SetLogger(common.Logger)
		sessPWD := md5.Sum([]byte(common.SYS_PWD))
		store := sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
		store.MaxAge(common.ServConfig.SessionExpire)
		core.InitSessionStore(store)
	})
}

func TestAdminAPIsSmoke(t *testing.T) {
	initAdminTests()

	cell := "13" + time.Now().Format("150405000")
	regBody, _ := json.Marshal(&protos.UserReq{Cellphone: cell, Password: "123456"})
	regReq := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(regBody))
	regW := httptest.NewRecorder()
	faceuser.UserAdd(regW, regReq)

	loginBody, _ := json.Marshal(&protos.UserReq{Cellphone: cell, Password: "123456"})
	loginReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
	loginW := httptest.NewRecorder()
	faceuser.UserLogin(loginW, loginReq)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/tenant/query?page=1&pageSize=1", nil)
	for _, c := range loginW.Result().Cookies() {
		req.AddCookie(c)
	}
	AdminTenantQuery(w, req)

	var result map[string]interface{}
	_ = json.NewDecoder(w.Result().Body).Decode(&result)
	if _, ok := result["code"]; !ok {
		t.Fatalf("admin/tenant/query 返回不是标准格式: %+v", result)
	}
}

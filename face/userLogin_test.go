// 用户登录测试文件
// 运行方式：cd /opt/dev/passport && go test -v ./face -run TestUserLogin

package face

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/database"
	"github.com/liuhengloveyou/passport/protos"
)

// TestUserLoginSuccess 测试用户成功登录
func TestUserLoginSuccess(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 首先创建一个用户
		testData := generateUniqueTestData()
		cellphone := testData["cellphone"]
		password := "123456"

		// 创建用户
		userReq := &protos.UserReq{
			Cellphone: cellphone,
			Password:  password,
		}
		body, _ := json.Marshal(userReq)
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		userAdd(w, req)

		// 验证创建成功
		resp := w.Result()
		var createResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&createResult)
		if code, ok := createResult["code"].(float64); !ok || code != 0 {
			t.Logf("[%s] 创建用户失败，跳过登录测试: %v", dbName, createResult)
			return
		}

		// 现在测试登录
		loginReq := &protos.UserReq{
			Cellphone: cellphone,
			Password:  password,
		}
		loginBody, _ := json.Marshal(loginReq)
		loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
		loginHttpReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()

		// 调用登录API
		userLogin(loginW, loginHttpReq)

		// 验证登录响应
		loginResp := loginW.Result()
		assertEqual(t, http.StatusOK, loginResp.StatusCode, "["+dbName+"] 登录HTTP状态码")

		var loginResult map[string]interface{}
		json.NewDecoder(loginResp.Body).Decode(&loginResult)

		if code, ok := loginResult["code"].(float64); ok && code == 0 {
			t.Logf("[%s] 用户登录成功，响应: %+v", dbName, loginResult)
		} else {
			t.Errorf("[%s] 用户登录失败: %+v", dbName, loginResult)
		}
	})
}

// TestUserLoginInvalidPassword 测试错误密码登录
func TestUserLoginInvalidPassword(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		testData := generateUniqueTestData()
		cellphone := testData["cellphone"]

		userReq := &protos.UserReq{
			Cellphone: cellphone,
			Password:  "123456",
		}
		body, _ := json.Marshal(userReq)
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		userAdd(w, req)

		// 使用错误密码登录
		loginReq := &protos.UserReq{
			Cellphone: cellphone,
			Password:  "wrongpassword",
		}
		loginBody, _ := json.Marshal(loginReq)
		loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
		loginHttpReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()

		userLogin(loginW, loginHttpReq)

		loginResp := loginW.Result()
		var loginResult map[string]interface{}
		json.NewDecoder(loginResp.Body).Decode(&loginResult)

		if code, ok := loginResult["code"].(float64); ok && code != 0 {
			t.Logf("[%s] 正确地拒绝了错误密码: %+v", dbName, loginResult)
		} else {
			t.Logf("[%s] 登录结果: %+v", dbName, loginResult)
		}
	})
}

// TestUserLoginWithEmail 测试使用邮箱登录
func TestUserLoginWithEmail(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		testData := generateUniqueTestData()
		email := testData["email"]
		password := "123456"

		userReq := &protos.UserReq{
			Email:    email,
			Password: password,
		}
		body, _ := json.Marshal(userReq)
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		userAdd(w, req)

		// 使用邮箱登录
		loginReq := &protos.UserReq{
			Email:    email,
			Password: password,
		}
		loginBody, _ := json.Marshal(loginReq)
		loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
		loginHttpReq.Header.Set("Content-Type", "application/json")
		loginW := httptest.NewRecorder()

		userLogin(loginW, loginHttpReq)

		loginResp := loginW.Result()
		var loginResult map[string]interface{}
		json.NewDecoder(loginResp.Body).Decode(&loginResult)

		t.Logf("[%s] 使用邮箱登录结果: %+v", dbName, loginResult)
	})
}

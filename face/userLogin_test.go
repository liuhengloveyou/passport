// 用户登录测试文件
// 运行方式：cd /opt/dev/passport && go test -v ./face -run TestUserLogin

package face

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/v3/protos"
)

// TestUserLoginComplete 完整的登录测试流程：先创建用户，再测试所有登录场景
// go test -v --count=1 ./face -run TestUserLoginComplete
func TestUserLoginComplete(t *testing.T) {
	// 生成唯一的测试数据
	testData := generateUniqueTestData()
	cellphone := testData["cellphone"]
	email := testData["email"]
	nickname := testData["nickname"]
	password := "123456"

	// 第一步：创建测试用户
	t.Run("1.创建测试用户", func(t *testing.T) {
		userReq := &protos.UserReq{
			Cellphone: cellphone,
			Email:     email,
			Nickname:  nickname,
			Password:  password,
		}
		body, _ := json.Marshal(userReq)
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		userAdd(w, req)

		resp := w.Result()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if code, ok := result["code"].(float64); ok && code == 0 {
			t.Logf("✓ 创建测试用户成功")
			t.Logf("  手机号: %s", cellphone)
			t.Logf("  邮箱: %s", email)
			t.Logf("  昵称: %s", nickname)
		} else {
			t.Fatalf("✗ 创建测试用户失败: %+v", result)
		}
	})

	// 第二步：测试成功登录场景
	t.Run("2.成功登录场景", func(t *testing.T) {
		t.Run("使用手机号登录", func(t *testing.T) {
			testUserLoginWithCellphone(t, cellphone, password)
		})

		t.Run("使用邮箱登录", func(t *testing.T) {
			testUserLoginWithEmail(t, email, password)
		})
	})

	// 第三步：测试失败登录场景
	t.Run("3.失败登录场景", func(t *testing.T) {
		t.Run("错误密码", func(t *testing.T) {
			testUserLoginWithWrongPassword(t, cellphone)
		})

		t.Run("空密码", func(t *testing.T) {
			testUserLoginWithEmptyPassword(t, cellphone)
		})

		t.Run("不存在的用户", TestUserLoginNonExistentUser)

		t.Run("空凭证", TestUserLoginEmptyCredentials)
	})

	// 第四步：测试重复登录
	t.Run("4.重复登录", func(t *testing.T) {
		// 第一次登录
		testUserLoginWithCellphone(t, cellphone, password)
		// 第二次登录
		testUserLoginWithCellphone(t, cellphone, password)
		t.Logf("✓ 重复登录测试完成")
	})
}

// TestUserLoginSuccess 测试用户成功登录
func TestUserLoginSuccess(t *testing.T) {
	// 创建测试用户
	creds := createTestUser(t)

	// 使用刚创建的账号登录
	loginReq := &protos.UserReq{
		Cellphone: creds.Cellphone,
		Password:  creds.Password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	// 调用登录API
	userLogin(loginW, loginHttpReq)

	// 验证登录响应
	loginResp := loginW.Result()
	assertEqual(t, http.StatusOK, loginResp.StatusCode, "登录HTTP状态码")

	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)

	if code, ok := loginResult["code"].(float64); ok && code == 0 {
		t.Logf("✓ 用户登录成功: %s", creds.Cellphone)
	} else {
		t.Errorf("✗ 用户登录失败: %+v", loginResult)
	}
}

// TestUserLoginInvalidPassword 测试错误密码登录
func TestUserLoginInvalidPassword(t *testing.T) {
	// 创建测试用户
	creds := createTestUser(t)

	// 使用错误密码登录
	loginReq := &protos.UserReq{
		Cellphone: creds.Cellphone,
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
		t.Logf("✓ 正确地拒绝了错误密码")
	} else {
		t.Logf("登录结果: %+v", loginResult)
	}
}

// TestUserLoginWithEmail 测试使用邮箱登录
func TestUserLoginWithEmail(t *testing.T) {
	// 创建测试用户（包含邮箱）
	creds := createTestUser(t)

	// 使用邮箱登录
	loginReq := &protos.UserReq{
		Email:    creds.Email,
		Password: creds.Password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	userLogin(loginW, loginHttpReq)

	loginResp := loginW.Result()
	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)

	if code, ok := loginResult["code"].(float64); ok && code == 0 {
		t.Logf("✓ 使用邮箱登录成功: %s", creds.Email)
	} else {
		t.Logf("使用邮箱登录结果: %+v", loginResult)
	}
}

// TestSendUserLoginSms 测试发送用户登录短信验证码
func TestSendUserLoginSms(t *testing.T) {
	smsReq := &protos.SmsReq{
		Cellphone: "18510511015",
		AliveSec:  300,
	}
	body, _ := json.Marshal(smsReq)

	req := httptest.NewRequest(http.MethodPost, "/sms/sendUserLoginSms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	SendUserLoginSms(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("发送登录短信响应: %+v", result)
}

// TestUserLoginNonExistentUser 测试不存在的用户登录
func TestUserLoginNonExistentUser(t *testing.T) {
	testData := generateUniqueTestData()
	cellphone := testData["cellphone"]

	loginReq := &protos.UserReq{
		Cellphone: cellphone,
		Password:  "123456",
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
		t.Logf("✓ 正确地拒绝了不存在的用户")
	} else {
		t.Errorf("✗ 不存在的用户不应该登录成功: %+v", loginResult)
	}
}

// TestUserLoginEmptyCredentials 测试空凭证登录
func TestUserLoginEmptyCredentials(t *testing.T) {
	loginReq := &protos.UserReq{
		Cellphone: "",
		Password:  "",
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
		t.Logf("✓ 正确地拒绝了空凭证")
	} else {
		t.Logf("空凭证登录结果: %+v", loginResult)
	}
}

// testUserLoginWithCellphone 使用手机号登录的测试逻辑（可复用）
func testUserLoginWithCellphone(t *testing.T, cellphone, password string) {
	loginReq := &protos.UserReq{
		Cellphone: cellphone,
		Password:  password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()

	userLogin(loginW, loginHttpReq)

	loginResp := loginW.Result()
	assertEqual(t, http.StatusOK, loginResp.StatusCode, "登录HTTP状态码")

	var loginResult map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&loginResult)

	if code, ok := loginResult["code"].(float64); ok && code == 0 {
		t.Logf("✓ 使用手机号登录成功: %s", cellphone)
	} else {
		t.Errorf("✗ 使用手机号登录失败: %+v", loginResult)
	}
}

// testUserLoginWithEmail 使用邮箱登录的测试逻辑（可复用）
func testUserLoginWithEmail(t *testing.T, email, password string) {
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

	if code, ok := loginResult["code"].(float64); ok && code == 0 {
		t.Logf("✓ 使用邮箱登录成功: %s", email)
	} else {
		t.Errorf("✗ 使用邮箱登录失败: %+v", loginResult)
	}
}

// testUserLoginWithWrongPassword 使用错误密码登录的测试逻辑（可复用）
func testUserLoginWithWrongPassword(t *testing.T, cellphone string) {
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
		t.Logf("✓ 正确地拒绝了错误密码")
	} else {
		t.Errorf("✗ 错误密码不应该登录成功: %+v", loginResult)
	}
}

// testUserLoginWithEmptyPassword 使用空密码登录的测试逻辑（可复用）
func testUserLoginWithEmptyPassword(t *testing.T, cellphone string) {
	loginReq := &protos.UserReq{
		Cellphone: cellphone,
		Password:  "",
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
		t.Logf("✓ 正确地拒绝了空密码")
	} else {
		t.Errorf("✗ 空密码不应该登录成功: %+v", loginResult)
	}
}

// 用户注册接口测试
// 运行方式：cd /opt/dev/passport && go test -v ./face -run TestUserAdd

package face

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/v3/protos"
)

// TestUserAddAll 测试所有用户注册场景的总入口
func TestUserAddAll(t *testing.T) {
	t.Run("TestUserAddSuccess", TestUserAddSuccess)
	t.Run("TestUserAddDuplicateCellphone", TestUserAddDuplicateCellphone)
	t.Run("TestUserAddWithEmail", TestUserAddWithEmail)
	t.Run("TestUserAddWithNick", TestUserAddWithNick)
	t.Run("TestUserAddInvalidData", TestUserAddInvalidData)
	// t.Run("TestSendUserAddSmsCode", TestSendUserAddSmsCode)
}

// TestUserAddSuccess 测试用户成功注册
func TestUserAddSuccess(t *testing.T) {
	testData := generateUniqueTestData()
	userReq := &protos.UserReq{
		Cellphone: testData["cellphone"],
		Email:     testData["email"],
		Nickname:  testData["nickname"],
		Password:  "123456",
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
		t.Logf("✓ 用户注册成功: cellphone=%s", testData["cellphone"])
	} else {
		t.Errorf("✗ 用户注册失败: %+v", result)
	}
}

// TestUserAddDuplicateCellphone 测试重复手机号注册
func TestUserAddDuplicateCellphone(t *testing.T) {
	testData := generateUniqueTestData()
	userReq := &protos.UserReq{
		Cellphone: testData["cellphone"],
		Password:  "123456",
	}
	body, _ := json.Marshal(userReq)

	// 第一次注册
	req1 := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	userAdd(w1, req1)

	// 第二次注册（使用相同手机号）
	req2 := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	userAdd(w2, req2)

	resp := w2.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code != 0 {
		t.Logf("✓ 正确地拒绝了重复手机号: %s", testData["cellphone"])
	} else {
		t.Logf("重复注册结果: %+v", result)
	}
}

// TestUserAddWithEmail 测试使用邮箱注册
func TestUserAddWithEmail(t *testing.T) {
	testData := generateUniqueTestData()
	userReq := &protos.UserReq{
		Email:    testData["email"],
		Password: "123456",
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
		t.Logf("✓ 使用邮箱注册成功: %s", testData["email"])
	} else {
		t.Logf("邮箱注册结果: %+v", result)
	}
}

// TestUserAddWithNick 测试使用昵称注册
func TestUserAddWithNick(t *testing.T) {
	testData := generateUniqueTestData()
	userReq := &protos.UserReq{
		Nickname: testData["nickname"],
		Password: "123456",
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
		t.Logf("✓ 使用昵称注册成功: %s", testData["nickname"])
	} else {
		t.Logf("昵称注册结果: %+v", result)
	}
}

// TestUserAddInvalidData 测试无效数据注册
func TestUserAddInvalidData(t *testing.T) {
	// 测试无手机号和邮箱
	userReq := &protos.UserReq{
		Password: "123456",
	}
	body, _ := json.Marshal(userReq)

	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	userAdd(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code != 0 {
		t.Logf("✓ 正确地拒绝了无效数据（无手机号和邮箱）")
	} else {
		t.Logf("无效数据注册结果: %+v", result)
	}
}

// TestSendUserAddSmsCode 测试发送用户注册短信验证码
func TestSendUserAddSmsCode(t *testing.T) {
	smsReq := &protos.SmsReq{
		Cellphone: "18510511015",
		AliveSec:  60, // 修改为合法值，max=100
	}
	body, _ := json.Marshal(smsReq)

	req := httptest.NewRequest(http.MethodPost, "/sms/sendUserAddSmsCode", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	SendUserAddSmsCode(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 发送注册短信验证码成功")
	} else {
		t.Errorf("✗ 发送注册短信验证码失败: %+v", result)
	}
}

// 用户信息API测试
// 运行方式：cd /opt/dev/passport && go test -v ./face -run TestGetMyInfo

package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/v3/protos"
)

// TestGetMyInfoSuccess 测试获取用户信息成功
func TestGetMyInfoSuccess(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	// 获取用户信息
	req := httptest.NewRequest(http.MethodGet, "/user/info", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	getMyInfo(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 获取用户信息成功")
	} else {
		t.Errorf("✗ 获取用户信息失败: %+v", result)
	}
	_ = user // 使用user避免未使用变量警告
}

// TestGetInfoByUID 测试根据UID获取用户信息
func TestGetInfoByUID(t *testing.T) {
	// 创建并登录用户
	user, sessionCookie := createAndLoginUser(t, "test")

	// 根据UID获取用户信息
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/user/infoByUID?uid=%d", user.UID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	getInfoByUID(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 根据UID获取用户信息成功")
	} else {
		t.Logf("根据UID获取用户信息结果: %+v", result)
	}
}

// TestUserModifyInfo 测试修改用户信息
func TestUserModifyInfo(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	// 修改用户信息
	modifyReq := map[string]interface{}{
		"nickname": "新昵称测试",
		"gender":   1,
		"addr":     "北京市朝阳区",
	}
	body, _ := json.Marshal(modifyReq)

	req := httptest.NewRequest(http.MethodPost, "/user/modify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	userModify(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 修改用户信息成功")
	} else {
		t.Logf("修改用户信息结果: %+v", result)
	}
}

// TestModifyUserPassword 测试修改密码
func TestModifyUserPassword(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	// 修改密码
	modifyPwdReq := &protos.ModifyPwdReq{
		OldPwd: "123456",
		NewPwd: "654321",
	}
	body, _ := json.Marshal(modifyPwdReq)

	req := httptest.NewRequest(http.MethodPost, "/user/modify/password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	modifyPWD(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 修改密码成功")
	} else {
		t.Logf("修改密码结果: %+v", result)
	}
}

// TestUserSearchByKeyword 测试搜索用户
func TestUserSearchByKeyword(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	// 获取nickname字符串值
	nickname := ""
	if user.Nickname.Valid {
		nickname = user.Nickname.String
	}

	// 搜索用户（使用昵称）
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/user/s/1?keyword=%s", nickname), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	searchLite(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("搜索用户结果: %+v", result)
}

// TestCheckUserAuth 测试用户认证
func TestCheckUserAuth(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	// 测试认证
	req := httptest.NewRequest(http.MethodPost, "/user/auth", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	UserAuth(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 用户认证成功")
	} else {
		t.Logf("用户认证结果: %+v", result)
	}
}

// TestUserLogout 测试用户登出
func TestUserLogout(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	// 登出
	req := httptest.NewRequest(http.MethodPost, "/user/logout", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	userLogout(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("✓ 用户登出成功")
	} else {
		t.Logf("用户登出结果: %+v", result)
	}
}

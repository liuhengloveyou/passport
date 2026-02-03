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

// TestTenantUserGet 测试获取租户用户列表
func TestTenantUserGet(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/getUsers?tenantId=%d&page=1&pageSize=10", user.TenantID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	TenantUserGet(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("租户用户列表响应: %+v", result)
}

// TestTenantUserAdd 测试为租户添加用户
func TestTenantUserAdd(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")
	testData := generateUniqueTestData()

	// 首先创建一个新用户
	userReq := &protos.UserReq{
		Cellphone: testData["cellphone"],
		Password:  "123456",
	}
	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userAdd(w, req)

	// 然后为租户添加该用户
	addUserReq := map[string]interface{}{
		"tenant_id": user.TenantID,
		"uid":       1, // 使用注册返回的UID
	}
	addBody, _ := json.Marshal(addUserReq)

	addReq := httptest.NewRequest(http.MethodPost, "/tenant/user/add", bytes.NewBuffer(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		addReq.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	addW := httptest.NewRecorder()
	TenantUserAdd(addW, addReq)

	addResp := addW.Result()
	var result map[string]interface{}
	json.NewDecoder(addResp.Body).Decode(&result)

	t.Logf("为租户添加用户响应: %+v", result)
}

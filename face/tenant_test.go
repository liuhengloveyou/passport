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

// TestCreateTenant 测试创建租户
func TestCreateTenant(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")
	testData := generateUniqueTestData()

	tenantReq := &protos.NewTenantReq{
		TenantName: "测试租户",
		Cellphone:  testData["cellphone"],
		Password:   "123456",
	}
	body, _ := json.Marshal(tenantReq)

	req := httptest.NewRequest(http.MethodPost, "/tenant/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	TenantAdd(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("创建租户响应: %+v (user.TenantID=%d)", result, user.TenantID)
}

// TestGetTenantRoles 测试获取租户角色
func TestGetTenantRoles(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/getRoles?tenantId=%d", user.TenantID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	TenantGetRole(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("获取租户角色响应: %+v", result)
}

// TestLoadConfiguration 测试加载租户配置
func TestLoadConfiguration(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/loadConfiguration?tenantId=%d", user.TenantID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	LoadConfiguration(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("加载租户配置响应: %+v", result)
}

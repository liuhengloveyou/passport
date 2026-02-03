package face

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAdminListUsers 测试管理员列出用户
func TestAdminListUsers(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, "/admin/user/list?page=1&pageSize=10", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	AdminUserList(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("管理员列出用户响应: %+v", result)
}

// TestAdminQueryTenant 测试管理员查询租户
func TestAdminQueryTenant(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, "/admin/tenant/query?page=1&pageSize=10", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	AdminTenantQuery(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("管理员查询租户响应: %+v", result)
}

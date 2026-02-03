package face

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTenantTreeList 测试获取租户树形结构
func TestTenantTreeList(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/tree/list?tenantId=%d&page=1&pageSize=10", user.TenantID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	TenantTreeList(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("租户树形结构响应: %+v", result)
}

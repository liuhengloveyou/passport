package face

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreatePermission 测试创建权限
func TestCreatePermission(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	permissionReq := map[string]interface{}{
		"name":        "用户查看权限",
		"description": "查看用户信息",
	}
	body, _ := json.Marshal(permissionReq)

	req := httptest.NewRequest(http.MethodPost, "/access/createPermission", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	PermissionCreate(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("创建权限响应: %+v", result)
}

// TestListPermission 测试列出权限
func TestListPermission(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, "/access/listPermission", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	PermissionList(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("列出权限响应: %+v", result)
}

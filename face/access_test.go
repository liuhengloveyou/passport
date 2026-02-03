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

// TestAddRoleForUser 测试为用户分配角色
func TestAddRoleForUser(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	roleReq := &protos.RoleReq{
		RoleValue: "admin",
		UID:       user.UID,
	}
	body, _ := json.Marshal(roleReq)

	req := httptest.NewRequest(http.MethodPost, "/access/addRoleForUser", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	AddRoleForUser(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("分配角色响应: %+v", result)
}

// TestGetRolesForUser 测试获取用户角色
func TestGetRolesForUser(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/access/getRolesForUser?uid=%d", user.UID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	GetRolesForUser(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("获取用户角色响应: %+v", result)
}

// TestGetRolesForMe 测试获取当前用户的角色
func TestGetRolesForMe(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, "/access/getRolesForMe", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	GetRolesForMe(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("获取当前用户角色响应: %+v", result)
}

// TestAddPolicyToRole 测试为角色分配权限
func TestAddPolicyToRole(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	policyReq := &protos.PolicyReq{
		Role: "user",
		Obj:  "/api/users",
		Act:  "GET",
	}
	body, _ := json.Marshal(policyReq)

	req := httptest.NewRequest(http.MethodPost, "/access/addPolicyToRole", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	AddPolicyToRole(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("分配权限响应: %+v", result)
}

// TestGetPolicy 测试获取权限策略
func TestGetPolicy(t *testing.T) {
	_, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, "/access/getPolicy", nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	GetPolicy(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("获取权限策略响应: %+v", result)
}

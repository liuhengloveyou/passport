package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAddDepartment 测试添加部门
func TestAddDepartment(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	deptReq := map[string]interface{}{
		"tenant_id": user.TenantID,
		"name":      "技术部",
		"leader":    "张三",
	}
	body, _ := json.Marshal(deptReq)

	req := httptest.NewRequest(http.MethodPost, "/tenant/department/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	addDepartment(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("添加部门响应: %+v", result)
}

// TestListDepartment 测试列出部门
func TestListDepartment(t *testing.T) {
	user, sessionCookie := createAndLoginUser(t, "test")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/department/list?tenantId=%d", user.TenantID), nil)
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}

	w := httptest.NewRecorder()
	listDepartment(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("列出部门响应: %+v", result)
}

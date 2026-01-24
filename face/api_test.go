package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
)

// ==================== SMS API Tests ====================

// TestSendUserAddSmsCode 测试发送用户注册短信验证码
func TestSendUserAddSmsCode(t *testing.T) {
	setupTest()

	smsReq := &protos.SmsReq{
		Cellphone: "18510511015",
		AliveSec:  600,
	}
	body, _ := json.Marshal(smsReq)

	req := httptest.NewRequest(http.MethodPost, "/sms/sendUserAddSmsCode", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	SendUserAddSmsCode(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("发送注册短信响应: %+v", result)
}

// TestSendUserLoginSms 测试发送用户登录短信验证码
func TestSendUserLoginSms(t *testing.T) {
	setupTest()

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

// ==================== Tenant API Tests ====================

// TestCreateTenant 测试创建租户
func TestCreateTenant(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		testData := generateUniqueTestData()
		tenantReq := &protos.NewTenantReq{
			TenantName: "测试租户-" + dbName,
			Cellphone:  testData["cellphone"],
			Password:   "123456",
		}
		body, _ := json.Marshal(tenantReq)

		req := httptest.NewRequest(http.MethodPost, "/tenant/add", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		TenantAdd(w, req)

		resp := w.Result()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		t.Logf("[%s] 创建租户响应: %+v", dbName, result)
	})
}

// TestGetTenantRoles 测试获取租户角色
func TestGetTenantRoles(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 获取租户角色响应: %+v", dbName, result)
	})
}

// ==================== Access Control API Tests ====================

// TestAddRoleForUser 测试为用户分配角色
func TestAddRoleForUser(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 分配角色响应: %+v", dbName, result)
	})
}

// TestGetRolesForUser 测试获取用户角色
func TestGetRolesForUser(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 获取用户角色响应: %+v", dbName, result)
	})
}

// TestAddPolicyToRole 测试为角色分配权限
func TestAddPolicyToRole(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 分配权限响应: %+v", dbName, result)
	})
}

// TestGetPolicy 测试获取权限策略
func TestGetPolicy(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 获取权限策略响应: %+v", dbName, result)
	})
}

// ==================== Permission API Tests ====================

// TestCreatePermission 测试创建权限
func TestCreatePermission(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 创建权限响应: %+v", dbName, result)
	})
}

// TestListPermission 测试列出权限
func TestListPermission(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 列出权限响应: %+v", dbName, result)
	})
}

// ==================== Department API Tests ====================

// TestAddDepartment 测试添加部门
func TestAddDepartment(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

		deptReq := map[string]interface{}{
			"tenant_id": user.TenantID,
			"name":      "技术部-" + dbName,
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

		t.Logf("[%s] 添加部门响应: %+v", dbName, result)
	})
}

// TestListDepartment 测试列出部门
func TestListDepartment(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 列出部门响应: %+v", dbName, result)
	})
}

// ==================== Admin API Tests ====================

// TestAdminListUsers 测试管理员列出用户
func TestAdminListUsers(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 管理员列出用户响应: %+v", dbName, result)
	})
}

// TestAdminQueryTenant 测试管理员查询租户
func TestAdminQueryTenant(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 管理员查询租户响应: %+v", dbName, result)
	})
}

// TestTenantTreeList 测试获取租户树形结构
func TestTenantTreeList(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 租户树形结构响应: %+v", dbName, result)
	})
}

// TestTenantUserGet 测试获取租户用户列表
func TestTenantUserGet(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 租户用户列表响应: %+v", dbName, result)
	})
}

// TestTenantUserAdd 测试为租户添加用户
func TestTenantUserAdd(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)
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

		t.Logf("[%s] 为租户添加用户响应: %+v", dbName, result)
	})
}

// TestGetRolesForMe 测试获取当前用户的角色
func TestGetRolesForMe(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		_, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 获取当前用户角色响应: %+v", dbName, result)
	})
}

// TestLoadConfiguration 测试加载租户配置
func TestLoadConfiguration(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		user, sessionCookie := createAndLoginUser(t, dbName)

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

		t.Logf("[%s] 加载租户配置响应: %+v", dbName, result)
	})
}

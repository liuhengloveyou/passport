// 完整API集成测试脚本
// 按顺序测试所有接口
// 运行方式: cd /opt/dev/passport && go test -v ./face -run TestIntegrationAllAPIs

package face

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
	"gopkg.in/guregu/null.v4/zero"
)

// TestIntegrationAllAPIs 完整的API集成测试流程
func TestIntegrationAllAPIs(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		t.Logf("\n========== 开始API集成测试 [%s] ==========\n", dbName)

		// ==================== 第1阶段: 用户注册 ====================
		t.Log("\n[第1阶段] 测试用户注册API")
		user, sessionCookie := testUserRegisterAPI(t, dbName)
		if user == nil {
			t.Fatalf("[%s] 用户注册失败，无法继续测试", dbName)
		}

		// ==================== 第2阶段: 用户登录 ====================
		t.Log("\n[第2阶段] 测试用户登录API")
		sessionCookie = testUserLoginAPI(t, dbName, user)
		if sessionCookie == "" {
			t.Fatalf("[%s] 用户登录失败，无法继续测试", dbName)
		}

		// ==================== 第3阶段: 用户信息 ====================
		t.Log("\n[第3阶段] 测试用户信息API")
		testUserInfoAPIs(t, dbName, sessionCookie, user)

		// ==================== 第4阶段: 用户修改 ====================
		t.Log("\n[第4阶段] 测试用户修改API")
		testUserModifyAPIs(t, dbName, sessionCookie)

		// ==================== 第5阶段: 创建租户 ====================
		t.Log("\n[第5阶段] 测试租户创建API")
		tenantID := testTenantCreateAPI(t, dbName, user)

		// ==================== 第6阶段: 租户管理 ====================
		t.Log("\n[第6阶段] 测试租户管理API")
		testTenantManagementAPIs(t, dbName, sessionCookie, tenantID)

		// ==================== 第7阶段: 权限管理 ====================
		t.Log("\n[第7阶段] 测试权限管理API")
		testAccessControlAPIs(t, dbName, sessionCookie, user)

		// ==================== 第8阶段: 部门管理 ====================
		t.Log("\n[第8阶段] 测试部门管理API")
		testDepartmentAPIs(t, dbName, sessionCookie, user)

		// ==================== 第9阶段: 管理员API ====================
		t.Log("\n[第9阶段] 测试管理员API")
		testAdminAPIs(t, dbName, sessionCookie)

		// ==================== 第10阶段: 用户登出 ====================
		t.Log("\n[第10阶段] 测试用户登出API")
		testUserLogoutAPI(t, dbName, sessionCookie)

		t.Logf("\n========== API集成测试完成 [%s] ==========\n", dbName)
	})
}

// ==================== 辅助测试函数 ====================

func testUserRegisterAPI(t *testing.T, dbName string) (*protos.User, string) {
	t.Log("→ 测试用户注册")
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

	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	if code, ok := result["code"].(float64); ok && code == 0 {
		t.Logf("  └─ 注册成功: 手机=%s", testData["cellphone"])
		userData, _ := json.Marshal(result["data"])
		var user protos.User
		json.Unmarshal(userData, &user)
		// 确保Cellphone被正确设置
		if user.Cellphone == nil {
			cellphone := zero.NewString(testData["cellphone"], true)
			user.Cellphone = &cellphone
			return &user, ""
		}
		t.Logf("  └─ 注册失败: %v", result)
	}
	return nil, ""
}

func testUserLoginAPI(t *testing.T, dbName string, user *protos.User) string {
	t.Log("→ 测试用户登录")
	loginReq := &protos.UserReq{
		Cellphone: user.Cellphone.String,
		Password:  "123456",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userLogin(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)

	// Debug: print all response headers
	t.Logf("  └─ Response headers: %v", w.Result().Header)

	cookies := w.Result().Cookies()
	sessionCookie := ""
	for _, cookie := range cookies {
		t.Logf("  └─ Found cookie: %s = %s", cookie.Name, cookie.Value)
		// Look for session cookie - it can have different names based on configuration
		if cookie.Name == common.ServConfig.SessionKey || cookie.Name == "SESSION_KEY" || cookie.Name == "session" {
			sessionCookie = cookie.Value
			break
		}
	}

	if code, ok := result["code"].(float64); ok && code == 0 {
		if sessionCookie != "" {
			t.Logf("  └─ 登录成功，获得session: %s", sessionCookie[:min(20, len(sessionCookie))]+"...")
		} else {
			t.Logf("  └─ 登录成功，但没有收到sessionCookie")
		}
	} else {
		t.Logf("  └─ 登录失败: %v", result)
	}
	return sessionCookie
}

func testUserInfoAPIs(t *testing.T, dbName string, sessionCookie string, user *protos.User) {
	t.Log("→ 测试获取当前用户信息")
	req := httptest.NewRequest(http.MethodGet, "/user/info", nil)
	addSessionCookie(req, sessionCookie)
	req = loadSessionFromRequest(req)
	w := httptest.NewRecorder()
	getMyInfo(w, req)
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试根据UID获取用户信息")
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/user/infoByUID?uid=%d", user.UID), nil)
	addSessionCookie(req2, sessionCookie)
	req2 = loadSessionFromRequest(req2)
	w2 := httptest.NewRecorder()
	getInfoByUID(w2, req2)
	json.NewDecoder(w2.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试搜索用户")
	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/user/s/1?keyword=%s", user.Nickname.String), nil)
	addSessionCookie(req3, sessionCookie)
	w3 := httptest.NewRecorder()
	searchLite(w3, req3)
	json.NewDecoder(w3.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试用户认证")
	req4 := httptest.NewRequest(http.MethodPost, "/user/auth", nil)
	addSessionCookie(req4, sessionCookie)
	req4 = loadSessionFromRequest(req4)
	w4 := httptest.NewRecorder()
	UserAuth(w4, req4)
	json.NewDecoder(w4.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

func testUserModifyAPIs(t *testing.T, dbName string, sessionCookie string) {
	t.Log("→ 测试修改用户信息")
	modifyReq := map[string]interface{}{
		"nickname": "NewNickname",
		"gender":   1,
		"addr":     "北京市朝阳区",
	}
	body, _ := json.Marshal(modifyReq)
	req := httptest.NewRequest(http.MethodPost, "/user/modify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addSessionCookie(req, sessionCookie)
	w := httptest.NewRecorder()
	userModify(w, req)
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试修改密码")
	modifyPwdReq := &protos.ModifyPwdReq{
		OldPwd: "123456",
		NewPwd: "654321",
	}
	pwdBody, _ := json.Marshal(modifyPwdReq)
	req2 := httptest.NewRequest(http.MethodPost, "/user/modify/password", bytes.NewBuffer(pwdBody))
	req2.Header.Set("Content-Type", "application/json")
	addSessionCookie(req2, sessionCookie)
	w2 := httptest.NewRecorder()
	modifyPWD(w2, req2)
	json.NewDecoder(w2.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

func testTenantCreateAPI(t *testing.T, dbName string, user *protos.User) uint64 {
	t.Log("→ 测试创建租户")
	testData := generateUniqueTestData()
	tenantReq := &protos.NewTenantReq{
		TenantName: "TestTenant-" + dbName,
		Cellphone:  testData["cellphone"],
		Password:   "123456",
	}
	body, _ := json.Marshal(tenantReq)
	req := httptest.NewRequest(http.MethodPost, "/tenant/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	TenantAdd(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	return user.TenantID
}

func testTenantManagementAPIs(t *testing.T, dbName string, sessionCookie string, tenantID uint64) {
	t.Log("→ 测试获取租户角色")
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/getRoles?tenantId=%d", tenantID), nil)
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w := httptest.NewRecorder()
	TenantGetRole(w, req)
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试租户树形列表")
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/tree/list?tenantId=%d&page=1&pageSize=10", tenantID), nil)
	if sessionCookie != "" {
		req2.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w2 := httptest.NewRecorder()
	TenantTreeList(w2, req2)
	json.NewDecoder(w2.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试获取租户用户列表")
	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/getUsers?tenantId=%d&page=1&pageSize=10", tenantID), nil)
	if sessionCookie != "" {
		req3.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w3 := httptest.NewRecorder()
	TenantUserGet(w3, req3)
	json.NewDecoder(w3.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试加载租户配置")
	req4 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/loadConfiguration?tenantId=%d", tenantID), nil)
	if sessionCookie != "" {
		req4.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w4 := httptest.NewRecorder()
	LoadConfiguration(w4, req4)
	json.NewDecoder(w4.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

func testAccessControlAPIs(t *testing.T, dbName string, sessionCookie string, user *protos.User) {
	t.Log("→ 测试为用户分配角色")
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
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试获取用户角色")
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/access/getRolesForUser?uid=%d", user.UID), nil)
	if sessionCookie != "" {
		req2.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w2 := httptest.NewRecorder()
	GetRolesForUser(w2, req2)
	json.NewDecoder(w2.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试获取当前用户角色")
	req3 := httptest.NewRequest(http.MethodGet, "/access/getRolesForMe", nil)
	if sessionCookie != "" {
		req3.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w3 := httptest.NewRecorder()
	GetRolesForMe(w3, req3)
	json.NewDecoder(w3.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试为角色分配权限")
	policyReq := &protos.PolicyReq{
		Role: "user",
		Obj:  "/api/users",
		Act:  "GET",
	}
	policyBody, _ := json.Marshal(policyReq)
	req4 := httptest.NewRequest(http.MethodPost, "/access/addPolicyToRole", bytes.NewBuffer(policyBody))
	req4.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req4.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w4 := httptest.NewRecorder()
	AddPolicyToRole(w4, req4)
	json.NewDecoder(w4.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试获取权限策略")
	req5 := httptest.NewRequest(http.MethodGet, "/access/getPolicy", nil)
	if sessionCookie != "" {
		req5.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w5 := httptest.NewRecorder()
	GetPolicy(w5, req5)
	json.NewDecoder(w5.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

func testDepartmentAPIs(t *testing.T, dbName string, sessionCookie string, user *protos.User) {
	t.Log("→ 测试添加部门")
	deptReq := map[string]interface{}{
		"tenant_id": user.TenantID,
		"name":      "TechDept-" + dbName,
		"leader":    "Leader",
	}
	body, _ := json.Marshal(deptReq)
	req := httptest.NewRequest(http.MethodPost, "/tenant/department/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w := httptest.NewRecorder()
	addDepartment(w, req)
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试列出部门")
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenant/department/list?tenantId=%d", user.TenantID), nil)
	if sessionCookie != "" {
		req2.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w2 := httptest.NewRecorder()
	listDepartment(w2, req2)
	json.NewDecoder(w2.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

func testAdminAPIs(t *testing.T, dbName string, sessionCookie string) {
	t.Log("→ 测试管理员列出用户")
	req := httptest.NewRequest(http.MethodGet, "/admin/user/list?page=1&pageSize=10", nil)
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w := httptest.NewRecorder()
	AdminUserList(w, req)
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])

	t.Log("→ 测试管理员查询租户")
	req2 := httptest.NewRequest(http.MethodGet, "/admin/tenant/query?page=1&pageSize=10", nil)
	if sessionCookie != "" {
		req2.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w2 := httptest.NewRecorder()
	AdminTenantQuery(w2, req2)
	json.NewDecoder(w2.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

func testUserLogoutAPI(t *testing.T, dbName string, sessionCookie string) {
	t.Log("→ 测试用户登出")
	req := httptest.NewRequest(http.MethodPost, "/user/logout", nil)
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "SESSION_KEY", Value: sessionCookie})
	}
	w := httptest.NewRecorder()
	userLogout(w, req)
	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	t.Logf("  └─ 响应码: %v", result["code"])
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// addSessionCookie adds session cookie to request with proper cookie name
func addSessionCookie(req *http.Request, sessionCookie string) {
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: common.ServConfig.SessionKey, Value: sessionCookie})
	}
}

// loadSessionFromRequest loads session from cookie and sets it in request context
func loadSessionFromRequest(req *http.Request) *http.Request {
	sess, err := sessionStore.Get(req, common.ServConfig.SessionKey)
	if err != nil {
		return req
	}
	return req.WithContext(context.WithValue(context.Background(), "session", sess))
}

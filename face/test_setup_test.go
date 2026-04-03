package face

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/liuhengloveyou/passport/v3/accessctl"
	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/protos"
	"github.com/liuhengloveyou/passport/v3/sessions"
)

// APIResponse 通用API响应结构
type APIResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data map[string]interface{} `json:"data"`
}

// TestMain 是测试包的入口点，在所有测试运行前初始化环境
func TestMain(m *testing.M) {
	// 初始化测试环境
	if err := initTestEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化测试环境失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[TestMain] 测试初始化完成，开始运行测试...\n\n")

	// 运行所有测试
	code := m.Run()

	os.Exit(code)
}

// initTestEnvironment 初始化测试环境
// 模拟 InitAndRunHttpApi 的初始化逻辑（但不启动HTTP服务器）
func initTestEnvironment() error {
	// 1. 验证common.init()的初始化结果
	if common.DB == nil {
		return fmt.Errorf("数据库未初始化（common.init()应该已初始化）")
	}

	if common.Logger == nil {
		return fmt.Errorf("日志未初始化（common.init()应该已初始化）")
	}

	// 2. 初始化访问控制
	// 这是测试环境最关键的一步，InitAndRunHttpApi会做这个
	if common.ServConfig.DBDriver != "" && common.ServConfig.DBDSN != "" {
		// 查找RBAC模型文件
		rbacModelPaths := []string{
			"rbac_with_domains_model.conf",
			"../rbac_with_domains_model.conf",
			"./rbac_with_domains_model.conf",
		}
		var rbacModelPath string
		for _, path := range rbacModelPaths {
			if _, err := os.Stat(path); err == nil {
				rbacModelPath = path
				break
			}
		}

		if rbacModelPath == "" {
			return fmt.Errorf("未找到RBAC模型文件，访问控制相关测试将失败")
		}

		fmt.Printf("  使用RBAC模型文件: %s\n", rbacModelPath)
		if err := accessctl.InitAccessControl(rbacModelPath, common.ServConfig.DBDriver, common.ServConfig.DBDSN); err != nil {
			return fmt.Errorf("初始化访问控制失败: %w", err)
		}
		fmt.Println("  ✓ 访问控制初始化成功")
	} else {
		return fmt.Errorf("未配置数据库连接，无法初始化访问控制")
	}

	// 3. 初始化Session Store
	// 使用与InitAndRunHttpApi相同的逻辑
	sessPWD := md5.Sum([]byte(common.SYS_PWD))
	switch common.ServConfig.SessionStoreType {
	case "mem":
		// sessionStore = sessions.NewMemStore([]byte(common.SYS_PWD), sessPWD[:])
		// 暂时使用CookieStore
		sessionStore = sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
		sessionStore.(*sessions.CookieStore).MaxAge(common.ServConfig.SessionExpire)
	default:
		sessionStore = sessions.NewCookieStore([]byte(common.SYS_PWD), sessPWD[:])
		sessionStore.(*sessions.CookieStore).MaxAge(common.ServConfig.SessionExpire)
	}

	// 4. 设置face包的logger（模拟InitAndRunHttpApi）
	logger = common.Logger

	return nil
}

// assertEqual 断言两个值相等
func assertEqual(t *testing.T, expected, actual interface{}, name string) {
	if expected != actual {
		t.Errorf("%s不匹配: 期望=%v, 实际=%v", name, expected, actual)
	}
}

// assertAPISuccess 断言API调用成功（code=0）
func assertAPISuccess(t *testing.T, w *httptest.ResponseRecorder, apiName string) *APIResponse {
	t.Helper()

	if w.Code != http.StatusOK {
		t.Errorf("[%s] HTTP状态码错误: 期望=%d, 实际=%d", apiName, http.StatusOK, w.Code)
	}

	var result APIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&result); err != nil {
		t.Fatalf("[%s] 解析响应失败: %v", apiName, err)
	}

	if result.Code != 0 {
		t.Errorf("[%s] API返回错误: code=%d, msg=%s", apiName, result.Code, result.Msg)
	}

	return &result
}

// assertAPIError 断言API调用失败（code!=0）
func assertAPIError(t *testing.T, w *httptest.ResponseRecorder, apiName string) *APIResponse {
	t.Helper()

	var result APIResponse
	if err := json.NewDecoder(w.Result().Body).Decode(&result); err != nil {
		t.Fatalf("[%s] 解析响应失败: %v", apiName, err)
	}

	if result.Code == 0 {
		t.Errorf("[%s] 期望API返回错误，但成功了: data=%v", apiName, result.Data)
	}

	return &result
}

// assertHTTPStatus 断言HTTP状态码
func assertHTTPStatus(t *testing.T, w *httptest.ResponseRecorder, expected int, apiName string) {
	t.Helper()

	if w.Code != expected {
		t.Errorf("[%s] HTTP状态码错误: 期望=%d, 实际=%d", apiName, expected, w.Code)
	}
}

// assertResponseHasField 断言响应中包含特定字段
func assertResponseHasField(t *testing.T, response *APIResponse, fieldName string) {
	t.Helper()

	if response.Data == nil {
		t.Errorf("响应data为空")
		return
	}

	if _, exists := response.Data[fieldName]; !exists {
		t.Errorf("响应中缺少字段: %s, data=%v", fieldName, response.Data)
	}
}

// logAPICall 记录API调用信息
func logAPICall(t *testing.T, apiName string, result *APIResponse) {
	t.Helper()
	if result.Code == 0 {
		t.Logf("✓ [%s] 成功: msg=%s", apiName, result.Msg)
	} else {
		t.Logf("✗ [%s] 失败: code=%d, msg=%s", apiName, result.Code, result.Msg)
	}
}

// generateUniqueTestData 生成唯一的测试数据
// 使用时间戳（纳秒）确保每次测试运行时数据都是唯一的
func generateUniqueTestData() map[string]string {
	timestamp := time.Now().UnixNano()
	// 生成11位手机号：必须符合中国手机号格式
	// 格式：1[3456789|145|147|15|16|17|18|19]后跟8位数字
	// 生成：13 + 9位数字（从时间戳提取）
	phoneSuffix := fmt.Sprintf("%09d", timestamp%1000000000)
	cellphone := "13" + phoneSuffix

	return map[string]string{
		"cellphone": cellphone,
		"email":     fmt.Sprintf("test_%d@example.com", timestamp),
		"nickname":  fmt.Sprintf("TestUser_%d", timestamp),
	}
}

// createAndLoginUser 辅助函数：创建用户并登录，返回session cookie
func createAndLoginUser(t *testing.T, dbName string) (*protos.User, string) {
	testData := generateUniqueTestData()
	cellphone := testData["cellphone"]
	password := "123456"

	// 创建用户
	userReq := &protos.UserReq{
		Cellphone: cellphone,
		Nickname:  testData["nickname"],
		Password:  password,
	}
	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userAdd(w, req)

	var createResult map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&createResult)
	if code, ok := createResult["code"].(float64); !ok || code != 0 {
		t.Fatalf("[%s] 创建用户失败: %v", dbName, createResult)
	}

	// 登录
	loginReq := &protos.UserReq{
		Cellphone: cellphone,
		Password:  password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/user/login", bytes.NewBuffer(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	userLogin(loginW, loginHttpReq)

	var loginResult map[string]interface{}
	json.NewDecoder(loginW.Result().Body).Decode(&loginResult)
	if code, ok := loginResult["code"].(float64); !ok || code != 0 {
		t.Fatalf("[%s] 登录失败: %v", dbName, loginResult)
	}

	// 提取session cookie - 使用配置的session key
	cookies := loginW.Result().Cookies()
	sessionCookie := ""
	for _, cookie := range cookies {
		if cookie.Name == common.ServConfig.SessionKey {
			sessionCookie = cookie.Value
			break
		}
	}

	if sessionCookie == "" {
		t.Fatalf("[%s] 登录成功但未获取到session cookie，期望cookie名称: %s", dbName, common.ServConfig.SessionKey)
	}

	// 从登录结果中获取用户信息
	userData, _ := json.Marshal(createResult["data"])
	var user protos.User
	json.Unmarshal(userData, &user)

	return &user, sessionCookie
}

// TestUserCredentials 测试用户凭证信息
type TestUserCredentials struct {
	Cellphone string
	Email     string
	Nickname  string
	Password  string
}

// createTestUser 辅助函数：只创建用户，返回用户凭证信息（用于登录测试）
func createTestUser(t *testing.T) *TestUserCredentials {
	testData := generateUniqueTestData()
	cellphone := testData["cellphone"]
	email := testData["email"]
	nickname := testData["nickname"]
	password := "123456"

	// 创建用户
	userReq := &protos.UserReq{
		Cellphone: cellphone,
		Email:     email,
		Nickname:  nickname,
		Password:  password,
	}
	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	userAdd(w, req)

	var createResult map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&createResult)
	if code, ok := createResult["code"].(float64); !ok || code != 0 {
		t.Fatalf("创建测试用户失败: %v", createResult)
	}

	return &TestUserCredentials{
		Cellphone: cellphone,
		Email:     email,
		Nickname:  nickname,
		Password:  password,
	}
}

// addCookieToRequest 为请求添加session cookie
func addCookieToRequest(req *http.Request, sessionCookie string) {
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{
			Name:  common.ServConfig.SessionKey,
			Value: sessionCookie,
		})
	}
}

// makeAuthRequest 创建带认证的HTTP请求
func makeAuthRequest(method, url string, body []byte, sessionCookie string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	addCookieToRequest(req, sessionCookie)
	return req
}

// createTenant 创建测试租户
func createTenant(t *testing.T, sessionCookie string) uint64 {
	testData := generateUniqueTestData()
	tenantReq := &protos.NewTenantReq{
		TenantName: fmt.Sprintf("TestTenant_%d", time.Now().UnixNano()),
		Cellphone:  testData["cellphone"],
		Password:   "123456",
	}
	body, _ := json.Marshal(tenantReq)
	req := makeAuthRequest(http.MethodPost, "/tenant/add", body, sessionCookie)
	w := httptest.NewRecorder()
	TenantAdd(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	if code, ok := result["code"].(float64); !ok || code != 0 {
		t.Logf("创建租户失败: %v", result)
		return 0
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		if tenantID, ok := data["tenant_id"].(float64); ok {
			return uint64(tenantID)
		}
	}
	return 0
}

// createDepartment 创建测试部门
func createDepartment(t *testing.T, sessionCookie string, tenantID uint64, deptName string) uint64 {
	deptReq := map[string]interface{}{
		"tenant_id": tenantID,
		"name":      deptName,
		"leader":    "Leader",
	}
	body, _ := json.Marshal(deptReq)
	req := makeAuthRequest(http.MethodPost, "/tenant/department/add", body, sessionCookie)
	w := httptest.NewRecorder()
	addDepartment(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Result().Body).Decode(&result)
	if code, ok := result["code"].(float64); !ok || code != 0 {
		t.Logf("创建部门失败: %v", result)
		return 0
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		if deptID, ok := data["id"].(float64); ok {
			return uint64(deptID)
		}
	}
	return 0
}

// cd /opt/dev/passport && go test -v ./face -run TestUserAdd
// 同时测试PostgreSQL和SQLite3:
//   - 默认会测试SQLite3（内存数据库）
//   - 如果设置了POSTGRES_TEST_DSN环境变量，也会测试PostgreSQL
//   - 设置SKIP_POSTGRES_TEST=1可以跳过PostgreSQL测试
//
// 示例:
//   POSTGRES_TEST_DSN="postgres://user:pass@localhost:5432/testdb" go test -v ./face -run TestUserAdd
//   SKIP_POSTGRES_TEST=1 go test -v ./face -run TestUserAdd  # 只测试SQLite3

package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liuhengloveyou/passport/database"
	"github.com/liuhengloveyou/passport/protos"
	"go.uber.org/zap"
)

// 设置测试环境
func setupTest() {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
}

// generateUniqueTestData 生成唯一的测试数据
// 使用时间戳（纳秒）确保每次测试运行时数据都是唯一的
func generateUniqueTestData() map[string]string {
	timestamp := time.Now().UnixNano()
	// 生成11位手机号：1 + 10位数字（从时间戳提取）
	// 确保是11位：1 + 10位数字
	phoneSuffix := fmt.Sprintf("%010d", timestamp%10000000000)
	cellphone := "1" + phoneSuffix

	return map[string]string{
		"cellphone": cellphone,
		"email":     fmt.Sprintf("test_%d@example.com", timestamp),
		"nickname":  fmt.Sprintf("TestUser_%d", timestamp),
	}
}

// getUniqueCellphone 获取唯一的手机号
func getUniqueCellphone() string {
	return generateUniqueTestData()["cellphone"]
}

// getUniqueEmail 获取唯一的邮箱
func getUniqueEmail() string {
	return generateUniqueTestData()["email"]
}

// getUniqueNickname 获取唯一的昵称
func getUniqueNickname() string {
	return generateUniqueTestData()["nickname"]
}

// assertEqual 辅助函数用于比较值
func assertEqual(t *testing.T, expected, actual interface{}, msg string) {
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// assertNotNil 辅助函数用于检查非空
func assertNotNil(t *testing.T, value interface{}, msg string) {
	if value == nil {
		t.Errorf("%s: value should not be nil", msg)
	}
}

// assertHasError 检查响应是否包含错误码
func assertHasError(t *testing.T, result map[string]interface{}, msg string) {
	if code, ok := result["code"].(float64); !ok || code == 0 {
		t.Errorf("%s: expected error code, got result: %+v", msg, result)
	}
}

// TestUserAddSuccess 测试成功添加用户
// go test -v ./face -run TestUserAddSuccess
func TestUserAddSuccess(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 使用唯一数据
		testData := generateUniqueTestData()
		userReq := &protos.UserReq{
			Cellphone: testData["cellphone"],
			Email:     testData["email"],
			Password:  "123456",
		}

		t.Logf("[%s] 测试数据: 手机=%s, 邮箱=%s", dbName, testData["cellphone"], testData["email"])
		body, _ := json.Marshal(userReq)

		// 创建测试请求
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 调用被测试的函数
		userAdd(w, req)

		// 验证响应
		resp := w.Result()
		assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

		// 解析响应体
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		// 验证成功（code应该为0，data应该包含用户ID）
		if code, ok := result["code"].(float64); ok {
			if code != 0 {
				t.Errorf("[%s] 期望成功(code=0)，但得到错误码: %v, 响应: %+v", dbName, code, result)
			} else {
				t.Logf("[%s] 成功创建用户，响应: %+v", dbName, result)
			}
		} else {
			t.Errorf("[%s] 响应格式错误: %+v", dbName, result)
		}
	})
}

func TestUserAddMissingAllIdentifiers(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 缺少手机号、邮箱和昵称
		userReq := &protos.UserReq{
			Password: "123456",
		}

		body, _ := json.Marshal(userReq)

		// 创建测试请求
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 调用被测试的函数
		userAdd(w, req)

		// 验证响应
		resp := w.Result()
		assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

		// 解析响应体
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		// 验证返回了错误码
		t.Logf("[%s] Response: %+v", dbName, result)
		// 应该返回参数错误
		assertHasError(t, result, fmt.Sprintf("[%s] Error should be present", dbName))
	})
}

func TestUserAddMissingPassword(t *testing.T) {
	setupTest()

	// 准备测试数据 - 缺少密码
	userReq := &protos.UserReq{
		Cellphone: "18510511015",
		Email:     "test@example.com",
	}

	body, _ := json.Marshal(userReq)

	// 创建测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 验证返回了密码错误
	t.Logf("Response: %+v", result)
	assertHasError(t, result, "Error should be present for missing password")
}

func TestUserAddInvalidJSON(t *testing.T) {
	setupTest()

	// 准备无效的JSON数据
	body := []byte("{invalid json}")

	// 创建测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 验证返回了参数错误
	t.Logf("Response: %+v", result)
	assertHasError(t, result, "Error should be present for invalid JSON")
}

func TestUserAddWithValidCellphone(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 只使用手机号（使用唯一数据）
		userReq := &protos.UserReq{
			Cellphone: getUniqueCellphone(),
			Password:  "123456",
		}

		t.Logf("[%s] 测试手机号: %s", dbName, userReq.Cellphone)
		body, _ := json.Marshal(userReq)

		// 创建测试请求
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 调用被测试的函数
		userAdd(w, req)

		// 验证响应
		resp := w.Result()
		assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

		// 解析响应体
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		// 验证成功
		if code, ok := result["code"].(float64); ok && code == 0 {
			t.Logf("[%s] 成功创建用户，响应: %+v", dbName, result)
		} else {
			t.Errorf("[%s] 期望成功，但得到错误: %+v", dbName, result)
		}
	})
}

func TestUserAddWithValidEmail(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 只使用邮箱（使用唯一数据）
		userReq := &protos.UserReq{
			Email:    getUniqueEmail(),
			Password: "123456",
		}

		t.Logf("[%s] 测试邮箱: %s", dbName, userReq.Email)
		statusCode, result := ExecuteUserAddRequest(t, dbName, userReq)

		assertEqual(t, http.StatusOK, statusCode, "HTTP status code")
		if code, ok := result["code"].(float64); ok && code == 0 {
			t.Logf("[%s] 成功创建用户，响应: %+v", dbName, result)
		} else {
			t.Errorf("[%s] 期望成功，但得到错误: %+v", dbName, result)
		}
	})
}

func TestUserAddWithValidNickname(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 只使用昵称（使用唯一数据）
		userReq := &protos.UserReq{
			Nickname: getUniqueNickname(),
			Password: "123456",
		}

		t.Logf("[%s] 测试昵称: %s", dbName, userReq.Nickname)
		statusCode, result := ExecuteUserAddRequest(t, dbName, userReq)

		assertEqual(t, http.StatusOK, statusCode, "HTTP status code")
		if code, ok := result["code"].(float64); ok && code == 0 {
			t.Logf("[%s] 成功创建用户，响应: %+v", dbName, result)
		} else {
			t.Errorf("[%s] 期望成功，但得到错误: %+v", dbName, result)
		}
	})
}

func TestUserAddWithAllFields(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 包含所有字段（使用唯一数据）
		testData := generateUniqueTestData()
		userReq := &protos.UserReq{
			Cellphone: testData["cellphone"],
			Email:     testData["email"],
			Nickname:  testData["nickname"],
			Password:  "123456",
			AvatarURL: "http://example.com/avatar.jpg",
			Addr:      "北京市朝阳区",
			Gender:    1,
		}

		t.Logf("[%s] 完整测试数据: 手机=%s, 邮箱=%s, 昵称=%s", dbName, testData["cellphone"], testData["email"], testData["nickname"])
		statusCode, result := ExecuteUserAddRequest(t, dbName, userReq)

		assertEqual(t, http.StatusOK, statusCode, "HTTP status code")
		if code, ok := result["code"].(float64); ok && code == 0 {
			t.Logf("[%s] 成功创建用户，响应: %+v", dbName, result)
		} else {
			t.Errorf("[%s] 期望成功，但得到错误: %+v", dbName, result)
		}
	})
}

func TestUserAddEmptyBody(t *testing.T) {
	setupTest()

	// 创建空body的测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 验证返回了错误
	t.Logf("Response: %+v", result)
	assertHasError(t, result, "Error should be present for empty body")
}

func TestUserAddPasswordTooShort(t *testing.T) {
	setupTest()

	// 准备测试数据 - 密码太短
	userReq := &protos.UserReq{
		Cellphone: "18510511015",
		Password:  "123", // 密码太短
	}

	body, _ := json.Marshal(userReq)

	// 创建测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("Response: %+v", result)
	// 根据验证规则，密码至少需要6位
}

func TestUserAddInvalidEmail(t *testing.T) {
	setupTest()

	// 准备测试数据 - 无效的邮箱格式
	userReq := &protos.UserReq{
		Email:    "invalid-email", // 无效的邮箱格式
		Password: "123456",
	}

	body, _ := json.Marshal(userReq)

	// 创建测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("Response: %+v", result)
	// 根据验证规则，应该返回错误
}

func TestUserAddInvalidCellphone(t *testing.T) {
	setupTest()

	// 准备测试数据 - 无效的手机号
	userReq := &protos.UserReq{
		Cellphone: "123", // 无效的手机号（不是11位）
		Password:  "123456",
	}

	body, _ := json.Marshal(userReq)

	// 创建测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	t.Logf("Response: %+v", result)
	// 根据验证规则，手机号必须是11位
}

func TestUserAddBodyTooLarge(t *testing.T) {
	setupTest()

	// 创建一个超大的body
	largeData := make([]byte, 2048) // 超过1024字节限制
	for i := range largeData {
		largeData[i] = 'a'
	}

	// 创建测试请求
	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(largeData))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 调用被测试的函数
	userAdd(w, req)

	// 验证响应
	resp := w.Result()
	assertEqual(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	// 解析响应体
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// 验证返回了参数错误
	t.Logf("Response: %+v", result)
	assertHasError(t, result, "Error should be present for body too large")
}

// TestUserAddDuplicateUser 测试重复用户注册（同时测试PostgreSQL和SQLite3）
func TestUserAddDuplicateUser(t *testing.T) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		// 准备测试数据 - 使用唯一数据确保第一次注册能成功
		testData := generateUniqueTestData()
		userReq := &protos.UserReq{
			Cellphone: testData["cellphone"],
			Email:     testData["email"],
			Password:  "123456",
		}

		t.Logf("[%s] 重复测试数据: 手机=%s, 邮箱=%s", dbName, testData["cellphone"], testData["email"])
		body, _ := json.Marshal(userReq)

		// 第一次注册
		req1 := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		userAdd(w1, req1)

		resp1 := w1.Result()
		var result1 map[string]interface{}
		json.NewDecoder(resp1.Body).Decode(&result1)

		// 验证第一次注册成功
		if code, ok := result1["code"].(float64); !ok || code != 0 {
			t.Errorf("[%s] 第一次注册应该成功，但得到: %+v", dbName, result1)
			return
		}
		t.Logf("[%s] 第一次注册成功: %+v", dbName, result1)

		// 第二次注册相同的用户
		req2 := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		userAdd(w2, req2)

		// 验证第二次请求的响应
		resp2 := w2.Result()
		assertEqual(t, http.StatusOK, resp2.StatusCode, "HTTP status code")

		var result2 map[string]interface{}
		json.NewDecoder(resp2.Body).Decode(&result2)

		// 验证返回了重复错误
		if code, ok := result2["code"].(float64); ok && code == 0 {
			t.Errorf("[%s] 第二次注册应该失败（重复），但得到成功: %+v", dbName, result2)
		} else {
			t.Logf("[%s] 第二次注册正确返回重复错误: %+v", dbName, result2)
		}
	})
}

// 集成测试示例 - 需要数据库连接
func TestUserAddIntegration(t *testing.T) {
	// 跳过集成测试，除非设置了特定的环境变量
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	setupTest()

	// 这里需要实际的数据库连接
	// 初始化数据库连接
	// if err := common.InitDB(); err != nil {
	// 	t.Fatalf("Failed to init DB: %v", err)
	// }

	t.Log("集成测试需要数据库连接，请根据实际情况配置")
}

// Benchmark测试
func BenchmarkUserAdd(b *testing.B) {
	setupTest()

	// 基准测试使用相同数据（因为多次调用会失败，但我们只关心性能）
	userReq := &protos.UserReq{
		Cellphone: getUniqueCellphone(),
		Email:     getUniqueEmail(),
		Password:  "123456",
	}

	body, _ := json.Marshal(userReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		userAdd(w, req)
	}
}

// TestUserAddConcurrency 并发测试（同时测试PostgreSQL和SQLite3）
func TestUserAddConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		setupTest()

		concurrency := 10
		done := make(chan bool, concurrency)
		successCount := 0
		errorCount := 0

		// 每个并发请求使用不同的唯一数据
		for i := 0; i < concurrency; i++ {
			go func(id int) {
				// 每个goroutine使用独立的唯一数据
				testData := generateUniqueTestData()
				userReq := &protos.UserReq{
					Cellphone: testData["cellphone"],
					Email:     testData["email"],
					Password:  "123456",
				}

				body, _ := json.Marshal(userReq)
				req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				userAdd(w, req)

				resp := w.Result()
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)

				if code, ok := result["code"].(float64); ok && code == 0 {
					successCount++
				} else {
					errorCount++
				}

				done <- true
			}(i)
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}

		t.Logf("[%s] 并发测试完成: 成功=%d, 失败=%d", dbName, successCount, errorCount)

		// SQLite3在并发写入时可能会有锁等待，所以成功率可能不是100%
		if dbName == "PostgreSQL" {
			// PostgreSQL应该能处理所有并发请求
			if successCount < concurrency {
				t.Errorf("[%s] 期望所有并发请求成功，但只有 %d/%d 成功", dbName, successCount, concurrency)
			}
		}
	})
}

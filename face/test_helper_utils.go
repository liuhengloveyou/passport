// 测试辅助函数
// 包含创建和登录用户的通用函数
package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liuhengloveyou/passport/protos"
)

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

	// 提取session cookie
	cookies := loginW.Result().Cookies()
	sessionCookie := ""
	for _, cookie := range cookies {
		if cookie.Name == "SESSION_KEY" || cookie.Name == "session" {
			sessionCookie = cookie.Value
			break
		}
	}

	// 从登录结果中获取用户信息
	userData, _ := json.Marshal(createResult["data"])
	var user protos.User
	json.Unmarshal(userData, &user)

	return &user, sessionCookie
}

package face

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/liuhengloveyou/passport/database"
	"github.com/liuhengloveyou/passport/protos"
)

// TestUserAddFunc 测试函数类型
type TestUserAddFunc func(t *testing.T, db database.DB, dbName string, userReq *protos.UserReq)

// RunUserAddTestWithDBs 使用所有数据库运行用户添加测试
func RunUserAddTestWithDBs(t *testing.T, testName string, userReq *protos.UserReq, testFunc TestUserAddFunc) {
	RunWithDBs(t, func(t *testing.T, db database.DB, dbName string) {
		testFunc(t, db, dbName, userReq)
	})
}

// ExecuteUserAddRequest 执行用户添加请求并返回结果
func ExecuteUserAddRequest(t *testing.T, dbName string, userReq *protos.UserReq) (int, map[string]interface{}) {
	body, err := json.Marshal(userReq)
	if err != nil {
		t.Fatalf("[%s] 序列化请求失败: %v", dbName, err)
	}

	req := httptest.NewRequest(http.MethodPost, "/user/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	userAdd(w, req)

	resp := w.Result()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return resp.StatusCode, result
}


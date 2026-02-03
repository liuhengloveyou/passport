// 完整API集成测试脚本
// 按顺序测试所有接口
// go clean -testcache
// 运行方式: cd /opt/dev/passport && go test -v -count=1 ./face -run TestIntegrationAllAPIs

package face

import (
	"testing"
)

// TestIntegrationAllAPIs 完整的API集成测试流程
// 通过调用各个独立测试文件中的测试函数来完成集成测试
func TestIntegrationAllAPIs(t *testing.T) {
	t.Logf("\n========== 开始API集成测试 ==========\n")

	// ==================== 第1阶段: 用户注册 ====================
	t.Log("\n[第1阶段] 测试用户注册API")
	t.Run("用户注册成功", TestUserAddSuccess)
	t.Run("重复手机号注册", TestUserAddDuplicateCellphone)
	t.Run("使用邮箱注册", TestUserAddWithEmail)
	t.Run("无效数据注册", TestUserAddInvalidData)

	// ==================== 第2阶段: 用户登录 ====================
	t.Log("\n[第2阶段] 测试用户登录API")
	t.Run("用户登录成功", TestUserLoginSuccess)
	t.Run("错误密码登录", TestUserLoginInvalidPassword)
	t.Run("使用邮箱登录", TestUserLoginWithEmail)
	t.Run("不存在的用户登录", TestUserLoginNonExistentUser)
	t.Run("空凭证登录", TestUserLoginEmptyCredentials)

	// ==================== 第3阶段: 用户信息 ====================
	t.Log("\n[第3阶段] 测试用户信息API")
	t.Run("获取当前用户信息", TestGetMyInfoSuccess)
	t.Run("根据UID获取用户信息", TestGetInfoByUID)
	t.Run("搜索用户", TestUserSearchByKeyword)
	t.Run("用户认证", TestCheckUserAuth)

	// // ==================== 第4阶段: 用户修改 ====================
	// t.Log("\n[第4阶段] 测试用户修改API")
	// t.Run("修改用户信息", TestUserModifyInfo)
	// t.Run("修改密码", TestModifyUserPassword)

	// // ==================== 第5阶段: 租户管理 ====================
	// t.Log("\n[第5阶段] 测试租户管理API")
	// t.Run("创建租户", TestCreateTenant)
	// t.Run("获取租户角色", TestGetTenantRoles)
	// t.Run("加载租户配置", TestLoadConfiguration)

	// // ==================== 第6阶段: 权限管理 ====================
	// t.Log("\n[第6阶段] 测试权限管理API")
	// t.Run("为用户分配角色", TestAddRoleForUser)
	// t.Run("获取用户角色", TestGetRolesForUser)
	// t.Run("获取当前用户角色", TestGetRolesForMe)
	// t.Run("为角色分配权限", TestAddPolicyToRole)
	// t.Run("获取权限策略", TestGetPolicy)

	// // ==================== 第7阶段: 部门管理 ====================
	// t.Log("\n[第7阶段] 测试部门管理API")
	// t.Run("添加部门", TestAddDepartment)
	// t.Run("列出部门", TestListDepartment)

	// // ==================== 第8阶段: 管理员API ====================
	// t.Log("\n[第8阶段] 测试管理员API")
	// t.Run("管理员列出用户", TestAdminListUsers)
	// t.Run("管理员查询租户", TestAdminQueryTenant)

	// // ==================== 第9阶段: 用户登出 ====================
	// t.Log("\n[第9阶段] 测试用户登出API")
	// t.Run("用户登出", TestUserLogout)

	t.Logf("\n========== API集成测试完成 ==========\n")
}

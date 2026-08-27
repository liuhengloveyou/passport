package accessctl

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	"github.com/casbin/casbin/v3"
	_ "github.com/lib/pq"           // PostgreSQL驱动
	_ "github.com/mattn/go-sqlite3" // SQLite3驱动
	"go.uber.org/zap"

	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/protos"
)

// var policyCache = make(map[string]bool, 10000)

// InitAccessControl 初始化访问控制
// 支持PostgreSQL和SQLite3数据库
// rbacModel: RBAC模型文件路径
// driver: 数据库驱动类型 ("postgres" 或 "sqlite3")
// dsn: 数据库连接字符串
func InitAccessControl(rbacModel, driver, dsn string) (err error) {
	// 根据驱动类型打开数据库连接
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 测试连接
	if err = db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 绝对不能关
	// defer db.Close()

	// 设置连接池参数（SQLite3可能不需要，但设置也无妨）
	if driver == "postgres" {
		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(time.Minute * 10)
	}

	// runtime.SetFinalizer(db, finalizer)

	// 根据驱动类型创建适配器
	// sql-adapter支持多种数据库，驱动名称需要匹配
	adapterDriver := driver
	if driver == "sqlite3" {
		// sql-adapter可能使用不同的名称，尝试使用sqlite3
		adapterDriver = "sqlite3"
	}

	adapter, err := sqladapter.NewAdapter(db, adapterDriver, "casbin_rule")
	if err != nil {
		db.Close()
		return fmt.Errorf("创建casbin适配器失败: %w", err)
	}

	if enforcer, err = casbin.NewSyncedEnforcer(rbacModel, adapter); err != nil {
		return err
	}

	// Load the policy from DB.
	if err = enforcer.LoadPolicy(); err != nil {
		return err
	}

	// enforcer.StartAutoLoadPolicy(10 * time.Minute)

	enforcer.AddFunction("MyMatch", func(args ...any) (any, error) {
		rsub, rdom, _, _ := args[0].(string), args[1].(string), args[2].(string), args[3].(string)
		// fmt.Println("MyMatch: ", rsub, rdom, robj, ract)

		// root账号放行
		roles, err := enforcer.GetRolesForUser(rsub, rdom)
		if err != nil {
			panic(err)
		}
		for i := 0; i < len(roles); i++ {
			if roles[i] == "root" {
				return true, nil
			}
		}

		return false, nil
	})

	// enforcer.EnableLog(true)
	// enforcer.SetLogger(zaplogger.NewLoggerByZap(common.Logger, true))

	return nil
}

func Enforce(uid, tenantID, orgID uint64, obj, act string) (bool, error) {
	if orgID == 0 {
		return false, common.ErrOrgRequired
	}
	return enforce(genUserByUID(uid), Domain(tenantID, orgID), obj, act)
}

func AddRoleForUserInDomain(uid, tenantID, orgID uint64, role string) (err error) {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return addRoleForUserInDomain(genUserByUID(uid), role, Domain(tenantID, orgID))
}

func DeleteRoleForUserInDomain(uid, tenantID, orgID uint64, role string) (err error) {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return deleteRoleForUserInDomain(genUserByUID(uid), role, Domain(tenantID, orgID))
}

func DeleteRolesForUserInDomain(uid, tenantID, orgID uint64) (err error) {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return deleteRolesForUserInDomain(genUserByUID(uid), Domain(tenantID, orgID))
}

func GetRoleForUserInDomain(uid, tenantID, orgID uint64) (roles []string) {
	if orgID == 0 {
		return
	}
	var userInfo *protos.User

	userInfo, err := dao.UserQueryByID(uid)
	if err != nil {
		common.Logger.Sugar().Errorf("GetRoleForUserInDomain UserSelectByID ERR: %v\n", err)
		return
	}
	if userInfo == nil || userInfo.TenantID != tenantID {
		common.Logger.Sugar().Errorf("GetRoleForUserInDomain userInfo ERR: %d %d %v\n", uid, tenantID, userInfo)
		return
	}
	userInfo.Password = ""

	common.Logger.Debug("GetRoleForUserInDomain: ", zap.Uint64("uid", uid), zap.Uint64("tid", tenantID), zap.Uint64("org", orgID), zap.Any("user", userInfo), zap.Error(err))

	return getRoleForUserInDomain(genUserByUID(uid), Domain(tenantID, orgID))
}

func GetUsersForRoleInDomain(role string, tenantID, orgID uint64) (ids []uint64) {
	if orgID == 0 {
		return
	}
	users := getUsersForRoleInDomain(role, Domain(tenantID, orgID))

	ids = make([]uint64, len(users))
	for i := 0; i < len(users); i++ {
		uid, _ := strconv.Atoi(strings.Split(users[i], "-")[1])
		ids[i] = uint64(uid)
	}

	return
}

func AddPolicyToRole(tenantID, orgID uint64, role, obj, act string) (err error) {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return addPolicy(role, Domain(tenantID, orgID), obj, act)
}

func RemovePolicyFromRole(tenantID, orgID uint64, role, obj, act string) (err error) {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return removePolicy(role, Domain(tenantID, orgID), obj, act)
}

func GetFilteredPolicy(tenantID, orgID uint64, roles []string) (lists [][]string) {
	if orgID == 0 {
		return
	}
	policys, err := getFilteredPolicy(Domain(tenantID, orgID))
	common.Logger.Debug("getFilteredPolicy:", zap.Any("policys", policys), zap.Any("roles", roles), zap.Error(err))
	if len(policys) == 0 {
		return
	}

	if len(roles) <= 0 {
		lists = policys
		return
	}

	lists = make([][]string, 0)
	for i := 0; i < len(policys); i++ {
		for j := 0; j < len(roles); j++ {
			if policys[i][0] == roles[j] {
				lists = append(lists, policys[i])
			}
		}
	}

	return
}

// --- 部门角色继承（基于 Casbin g 策略链式继承）---
//
// 原理：
//   g, dep-3, waiter,  domain      部门3 拥有 waiter 角色
//   g, uid-100, dep-3,  domain      用户100 属于部门3
//   => Casbin 自动推导: uid-100 → dep-3 → waiter → policy
//
// 优点：
//   - 部门角色变更，全体成员自动生效，无需逐人更新
//   - 新员工入部门只需加一条 g 策略

func genDepByID(depID uint64) string {
	return fmt.Sprintf("dep-%v", depID)
}

// AddDepRole 给部门添加角色 (g, dep-{id}, role, domain)
func AddDepRole(depID, tenantID, orgID uint64, role string) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return addRoleForUserInDomain(genDepByID(depID), role, Domain(tenantID, orgID))
}

// DeleteDepRole 移除部门的某个角色
func DeleteDepRole(depID, tenantID, orgID uint64, role string) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return deleteRoleForUserInDomain(genDepByID(depID), role, Domain(tenantID, orgID))
}

// GetDepRoles 返回部门的所有角色
func GetDepRoles(depID, tenantID, orgID uint64) []string {
	if orgID == 0 {
		return nil
	}
	return getRoleForUserInDomain(genDepByID(depID), Domain(tenantID, orgID))
}

// DeleteAllDepRoles 清空部门所有角色
func DeleteAllDepRoles(depID, tenantID, orgID uint64) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return deleteRolesForUserInDomain(genDepByID(depID), Domain(tenantID, orgID))
}

// JoinDepForUser 用户加入部门（继承部门角色）：g, uid-{uid}, dep-{depID}, domain
func JoinDepForUser(uid, depID, tenantID, orgID uint64) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return addRoleForUserInDomain(genUserByUID(uid), genDepByID(depID), Domain(tenantID, orgID))
}

// LeaveDepForUser 用户离开部门（移除继承）
func LeaveDepForUser(uid, depID, tenantID, orgID uint64) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	return deleteRoleForUserInDomain(genUserByUID(uid), genDepByID(depID), Domain(tenantID, orgID))
}

// GetImplicitRolesForUser 返回用户所有角色（含从部门继承的，传递闭包）
// 使用 Casbin 原生 GetImplicitRolesForUser，支持任意深度继承链，域隔离
func GetImplicitRolesForUser(uid, tenantID, orgID uint64) []string {
	if orgID == 0 {
		return nil
	}
	roles, err := enforcer.GetImplicitRolesForUser(genUserByUID(uid), Domain(tenantID, orgID))
	if err != nil {
		common.Logger.Sugar().Errorf("GetImplicitRolesForUser ERR: uid=%d err=%v", uid, err)
		return nil
	}
	return roles
}

const depRolePrefix = "dep-"

func isDepRole(role string) bool {
	return strings.HasPrefix(role, depRolePrefix)
}

func depIDFromRole(role string) (uint64, bool) {
	if !isDepRole(role) {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(role, depRolePrefix), 10, 64)
	return id, err == nil && id > 0
}

// DirectRolesForUser 返回用户直接分配的业务角色（不含 dep-* 部门关联伪角色）
func DirectRolesForUser(uid, tenantID, orgID uint64) []string {
	if orgID == 0 {
		return nil
	}
	roles := getRoleForUserInDomain(genUserByUID(uid), Domain(tenantID, orgID))
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		if !isDepRole(role) {
			out = append(out, role)
		}
	}
	return out
}

// IsDepRoleKey 判断 Casbin 角色键是否为部门伪角色 dep-{id}
func IsDepRoleKey(role string) bool {
	return isDepRole(role)
}

// EffectiveRolesForUser 返回用户生效的业务角色（含部门继承，排除 dep-* 伪角色）
func EffectiveRolesForUser(uid, tenantID, orgID uint64) []string {
	roles := GetImplicitRolesForUser(uid, tenantID, orgID)
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if !isDepRole(r) {
			out = append(out, r)
		}
	}
	return out
}

// SyncUserDepsInOrg 同步用户在当前 org 下的部门 Casbin 关联（先离后加，避免换部门残留）
func SyncUserDepsInOrg(uid, tenantID, orgID uint64, wantDepIDs []uint64) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	want := make(map[uint64]struct{}, len(wantDepIDs))
	for _, id := range wantDepIDs {
		if id > 0 {
			want[id] = struct{}{}
		}
	}
	domain := Domain(tenantID, orgID)
	userKey := genUserByUID(uid)
	for _, role := range getRoleForUserInDomain(userKey, domain) {
		depID, ok := depIDFromRole(role)
		if !ok {
			continue
		}
		if _, keep := want[depID]; keep {
			continue
		}
		if err := deleteRoleForUserInDomain(userKey, genDepByID(depID), domain); err != nil {
			return err
		}
	}
	for id := range want {
		if err := addRoleForUserInDomain(userKey, genDepByID(id), domain); err != nil {
			return err
		}
	}
	return nil
}

// CleanupDepCasbinPolicies 删除部门相关的全部 Casbin g 策略（部门角色 + 成员关联）
func CleanupDepCasbinPolicies(depID, tenantID, orgID uint64) error {
	if orgID == 0 {
		return common.ErrOrgRequired
	}
	domain := Domain(tenantID, orgID)
	depKey := genDepByID(depID)
	if err := deleteRolesForUserInDomain(depKey, domain); err != nil {
		return err
	}
	_, err := enforcer.RemoveFilteredGroupingPolicy(1, depKey, domain)
	return err
}

// SetDepRoles 按 diff 更新部门角色，避免先删后加导致的中途空窗
func SetDepRoles(depID, tenantID, orgID uint64, roles []string) error {
	oldSet := stringSet(GetDepRoles(depID, tenantID, orgID))
	newSet := stringSet(normalizeRoleValues(roles))
	for role := range oldSet {
		if _, keep := newSet[role]; keep {
			continue
		}
		if err := DeleteDepRole(depID, tenantID, orgID, role); err != nil {
			return err
		}
	}
	for role := range newSet {
		if _, exists := oldSet[role]; exists {
			continue
		}
		if err := AddDepRole(depID, tenantID, orgID, role); err != nil {
			return err
		}
	}
	return nil
}

func normalizeRoleValues(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role != "" {
			out = append(out, role)
		}
	}
	return out
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

func genUserByUID(uid uint64) string {
	return fmt.Sprintf("uid-%v", uid)
}

func Domain(tenantID, orgID uint64) string {
	return fmt.Sprintf("tenant-%v-org-%v", tenantID, orgID)
}

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

func genUserByUID(uid uint64) string {
	return fmt.Sprintf("uid-%v", uid)
}

func Domain(tenantID, orgID uint64) string {
	return fmt.Sprintf("tenant-%v-org-%v", tenantID, orgID)
}

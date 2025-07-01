package accessctl

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	"github.com/casbin/casbin/v2"
	_ "github.com/lib/pq"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
	"go.uber.org/zap"
)

// var policyCache = make(map[string]bool, 10000)

func finalizer(db *sql.DB) {
	fmt.Println("finalizer: ", db.Stats())
	if db == nil {
		return
	}

	err := db.Close()
	if err != nil {
		panic(err)
	}
}

func InitAccessControl(rbacModel, urn string) (err error) {
	db, err := sql.Open("postgres", urn)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
	// 绝对不能关
	// defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 10)

	// runtime.SetFinalizer(db, finalizer)

	adapter, err := sqladapter.NewAdapter(db, "postgres", "casbin_rule")
	if err != nil {
		panic(err)
	}

	if enforcer, err = casbin.NewSyncedEnforcer(rbacModel, adapter); err != nil {
		return err
	}

	// Load the policy from DB.
	if err = enforcer.LoadPolicy(); err != nil {
		return err
	}

	// enforcer.StartAutoLoadPolicy(10 * time.Minute)

	enforcer.AddFunction("MyMatch", func(args ...interface{}) (interface{}, error) {
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

// sub, domain, obj, act
func Enforce(uid, tenantID uint64, obj, act string) (bool, error) {
	return enforce(genUserByUID(uid), genDomainByTenantID(tenantID), obj, act)
}

func AddRoleForUserInDomain(uid, tenantID uint64, role string) (err error) {
	//var userInfo *protos.User
	//if userInfo, err = dao.UserSelectByID(uid); err != nil {
	//	common.Logger.Sugar().Errorf("AddRoleForUserInDomain UserSelectByID ERR: %v\n", err)
	//	return common.ErrService
	//}
	//if userInfo == nil || userInfo.TenantID != tenantID {
	//	common.Logger.Sugar().Errorf("AddRoleForUserInDomain userInfo ERR: %d %d %v\n", uid, tenantID, userInfo)
	//	return common.ErrNoAuth
	//}

	return addRoleForUserInDomain(genUserByUID(uid), role, genDomainByTenantID(tenantID))
}

func DeleteRoleForUserInDomain(uid, tenantID uint64, role string) (err error) {
	return deleteRoleForUserInDomain(genUserByUID(uid), role, genDomainByTenantID(tenantID))
}

func DeleteRolesForUserInDomain(uid, tenantID uint64) (err error) {
	return deleteRolesForUserInDomain(genUserByUID(uid), genDomainByTenantID(tenantID))
}

func GetRoleForUserInDomain(uid, tenantID uint64) (roles []string) {
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

	common.Logger.Debug("GetRoleForUserInDomain: ", zap.Uint64("uid", uid), zap.Uint64("tid", tenantID), zap.Any("user", userInfo), zap.Error(err))

	return getRoleForUserInDomain(genUserByUID(uid), genDomainByTenantID(tenantID))
}

func GetUsersForRoleInDomain(role string, tenantID uint64) (ids []uint64) {
	users := getUsersForRoleInDomain(role, genDomainByTenantID(tenantID))

	ids = make([]uint64, len(users))
	for i := 0; i < len(users); i++ {
		uid, _ := strconv.Atoi(strings.Split(users[i], "-")[1])
		ids[i] = uint64(uid)
	}

	return
}

func AddPolicyToRole(tenantID uint64, role, obj, act string) (err error) {
	return addPolicy(role, genDomainByTenantID(tenantID), obj, act)

}

func RemovePolicyFromRole(tenantID uint64, role, obj, act string) (err error) {
	return removePolicy(role, genDomainByTenantID(tenantID), obj, act)
}

func GetFilteredPolicy(tenantID uint64, roles []string) (lists [][]string) {
	policys, err := getFilteredPolicy(genDomainByTenantID(tenantID))
	common.Logger.Debug("getFilteredPolicy:", zap.Any("policys", policys), zap.Any("roles", roles), zap.Error(err))
	if len(policys) == 0 {
		return
	}

	if len(roles) <= 0 {
		lists = policys // 不用过滤
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

func genDomainByTenantID(tenantID uint64) string {
	return fmt.Sprintf("tenant-%v", tenantID)
}

package accessctl

import (
	"database/sql"
	"fmt"
	"time"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	"github.com/casbin/casbin/v2"
)

func InitAccessControl(rbacModel, mysqlURN string) (err error) {
	// connect to the database first.
	db, err := sql.Open("mysql", mysqlURN)
	if err != nil {
		return err
	}
	if err = db.Ping();err!=nil{
		return err
	}
	//defer db.Close()

	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 10)

	// Initialize an adapter and use it in a Casbin enforcer:
	// The adapter will use the Sqlite3 table name "casbin_rule_test",
	// the default table name is "casbin_rule".
	// If it doesn't exist, the adapter will create it automatically.
	adapter, err := sqladapter.NewAdapter(db, "mysql", "casbin_rule")
	if err != nil {
		return err
	}

	if enforcer, err = casbin.NewSyncedEnforcer(rbacModel, adapter); err != nil {
		return err
	}

	//enforcer.StartAutoLoadPolicy(time.Minute)

	enforcer.AddFunction("MyMatch", func(args ...interface{}) (interface{}, error) {
		rsub, rdom, robj, ract := args[0].(string), args[1].(string), args[2].(string), args[3].(string)
		fmt.Println("MyMatch: ", rsub, rdom, robj, ract)

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

	return nil
}

// sub, domain, obj, act
func Enforce(uid, tenantID uint64, obj, act string) (bool, error){
	return enforce(genUserByUID(uid), genDomainByTenantID(tenantID), obj, act)
}

func AddRoleForUserInDomain(uid, tenantID uint64, role string) (err error) {
	return addRoleForUserInDomain(genUserByUID(uid), role, genDomainByTenantID(tenantID))
}

func DeleteRoleForUserInDomain(uid, tenantID uint64, role string) (err error) {
	return deleteRoleForUserInDomain(genUserByUID(uid), role, genDomainByTenantID(tenantID))
}

func AddPolicyToUser(uid, tenantID uint64, obj, act string) (err error) {
	return addPolicy(genUserByUID(uid), genDomainByTenantID(tenantID), obj, act)
}

func RemovePolicyFromUser(uid, tenantID uint64, obj, act string) (err error) {
	return removePolicy(genUserByUID(uid), genDomainByTenantID(tenantID), obj, act)
}

func AddPolicyToRole(tenantID uint64, role, obj, act string) (err error) {
	return addPolicy(role, genDomainByTenantID(tenantID), obj, act)
}

func RemovePolicyFromRole(tenantID uint64, role, obj, act string) (err error) {
	return removePolicy(role, genDomainByTenantID(tenantID), obj, act)
}

func genUserByUID(uid uint64) string {
	return fmt.Sprintf("uid-%v", uid)
}

func genDomainByTenantID(tenantID uint64) string {
	return fmt.Sprintf("tenant-%v", tenantID)
}
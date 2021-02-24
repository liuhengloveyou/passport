package accessctl

import (
	"database/sql"
	"time"

	sqladapter "github.com/Blank-Xu/sql-adapter"
	casbin "github.com/casbin/casbin/v2"
	_ "github.com/go-sql-driver/mysql"
)

var (
	enforcer *casbin.SyncedEnforcer
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

	db.SetMaxOpenConns(20)
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

	enforcer, err = casbin.NewSyncedEnforcer(rbacModel, adapter)
	if err != nil {
		return err
	}

	enforcer.StartAutoLoadPolicy(time.Minute)

	// 初始化数据
	enforcer.AddPolicy("admin", "*", "*")

	return nil
}

func enforce(sub, obj, act string) (bool, error){
	return enforcer.Enforce(sub, obj, act)
}

func addPolicy(sub, obj, act string) (err error) {
	if _, err = enforcer.AddPolicy(sub, obj, act); err != nil {
		return
	}

	if err = enforcer.SavePolicy(); err != nil {
		return
	}

	return
}

func removePolicy(sub, obj, act string) (err error) {
	if _, err = enforcer.RemovePolicy(sub, obj, act); err != nil {
		return
	}

	if err = enforcer.SavePolicy(); err != nil {
		return
	}

	return
}

func addRoleForUser(user string, role string) (err error) {
	if _, err = enforcer.AddRoleForUser(user, role); err != nil {
		return
	}

	if err = enforcer.SavePolicy(); err != nil {
		return
	}

	return
}

func deleteRoleForUser(user string, role string) (err error) {
	if _, err = enforcer.DeleteRoleForUser(user, role); err != nil {
		return
	}

	if err = enforcer.SavePolicy(); err != nil {
		return
	}

	return
}

func GetRolesForUser(uid uint64) (roles []string, err error){
	roles, err = enforcer.GetRolesForUser(genUserByUID(uid))

	return
}

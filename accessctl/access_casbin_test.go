package accessctl

import (
	"fmt"
	"testing"
)

func TestAccess(t *testing.T)  {
	err := InitAccessControl("../rbac_model.conf", "root:lhisroot@tcp(127.0.0.1:3306)/passport?charset=utf8&parseTime=true&loc=Local")
	fmt.Println("InitAccessControl: ", err)

	err = AddRoleForUser(123, "admin")
	fmt.Println("InitAccessControl: ", err)

	AddRoleForUser(123, "user")
	roles, err := GetRolesForUser(123)
	fmt.Println("GetRolesForUser: ", roles, err)
}

func TestEnforce(t *testing.T)  {
	err := InitAccessControl("../rbac_model.conf", "root:lhisroot@tcp(127.0.0.1:3306)/passport?charset=utf8&parseTime=true&loc=Local")
	fmt.Println("InitAccessControl: ", err)

	r, e := enforce("admin", "data1", "read")
	fmt.Println("enforce 1: ", r, e)

	enforcer.AddPolicy("admin", "data1", "read")
	r, e = enforce("admin", "data1", "read")
	fmt.Println("enforce 1: ", r, e)

	enforcer.AddRoleForUser("alice", "admin")

	r, e = enforce("alice", "data1", "read")
	fmt.Println("alice enforce: ", r, e)

	r, e = enforce("bob", "data1", "read")
	fmt.Println("bob enforce: ", r, e)
}
package accessctl

import (
	"fmt"
	"testing"
)

func TestAccess(t *testing.T)  {
	err := InitAccessControl("../rbac_with_domains_model.conf", "root:lhisroot@tcp(127.0.0.1:3306)/passport?charset=utf8&parseTime=true&loc=Local")
	fmt.Println("InitAccessControl: ", err)

	AddRoleForUserInDomain(100, 10000, "role-1")
	AddRoleForUserInDomain(101, 10001, "role-1")

	r, e := Enforce(101, 10001, "data-1", "read")
	fmt.Println(">>>>>>>>>>>>>", r, e)

	AddPolicyToUser(100, 10000, "data-1", "read")
	AddPolicyToRole(10001, "role-1", "data-1", "read")

	r, e = Enforce(100, 10000, "data-1", "read")
	fmt.Println(">>>>>>>>>>>>>", r, e)

	r, e = Enforce(101, 10001, "data-1", "read")
	fmt.Println(">>>>>>>>>>>>>", r, e)

	roles, err := enforcer.GetRolesForUser("uid-10000", "tenant-10015")
	fmt.Println("enforcer.GetRolesForUser(): ", roles, err)
	fmt.Println("enforcer.GetAllActions(): ", enforcer.GetAllActions())
	fmt.Println("enforcer.GetAllObjects(): ", enforcer.GetAllObjects())
	fmt.Println("enforcer.GetAllSubjects(): ", enforcer.GetAllSubjects())
	fmt.Println("enforcer.GetPolicy(): ", enforcer.GetPolicy())
	fmt.Println("enforcer.GetAllRoles(): ", enforcer.GetAllRoles())
}
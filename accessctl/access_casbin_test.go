package accessctl

import (
	"fmt"
	"testing"
)

func TestAccess(t *testing.T) {
	err := InitAccessControl("../rbac_with_domains_model.conf", "root:lhisroot@tcp(127.0.0.1:3306)/passport?charset=utf8&parseTime=true&loc=Local")
	fmt.Println("InitAccessControl: ", err)

	AddPolicyToRole(10030, "role-1", "data-1", "read")
	addRoleForUserInDomain("uid-123", "role-1", "tenant-10030")

	r, e := Enforce(123, 10030, "data-1", "read")
	fmt.Println(">>>>>>>>>>>>>", r, e)

	r, e = Enforce(143, 10030, "data-1", "read")
	fmt.Println(">>>>>>>>>>>>>", r, e)

	roles := enforcer.GetRolesForUserInDomain("uid-10000", "tenant-10030")
	fmt.Println("enforcer.GetRolesForUser(): ", roles)
	fmt.Println("enforcer.GetAllActions(): ", enforcer.GetAllNamedActions("p"))
	fmt.Println("enforcer.GetAllObjects(): ", enforcer.GetAllNamedObjects("p"))
	fmt.Println("enforcer.GetAllSubjects(): ", enforcer.GetAllSubjects())
	fmt.Println("enforcer.GetAllRoles(): ", enforcer.GetAllRoles())
	fmt.Println("enforcer.GetPolicy(): ", enforcer.GetFilteredPolicy(1, "tenant-10000"))
}

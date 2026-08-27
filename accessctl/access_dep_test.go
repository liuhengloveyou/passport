package accessctl

import (
	"path/filepath"
	"testing"
)

func TestDepRoleInheritance(t *testing.T) {
	model := filepath.Join("..", "rbac_with_domains_model.conf")
	if err := InitAccessControl(model, "sqlite3", ":memory:"); err != nil {
		t.Fatalf("InitAccessControl: %v", err)
	}

	const tenantID, orgID, depID, uid = 100, 1, 3, 100

	if err := AddDepRole(depID, tenantID, orgID, "waiter"); err != nil {
		t.Fatalf("AddDepRole: %v", err)
	}
	if err := JoinDepForUser(uid, depID, tenantID, orgID); err != nil {
		t.Fatalf("JoinDepForUser: %v", err)
	}
	if err := AddPolicyToRole(tenantID, orgID, "waiter", "/api/foo", "GET"); err != nil {
		t.Fatalf("AddPolicyToRole: %v", err)
	}

	ok, err := Enforce(uid, tenantID, orgID, "/api/foo", "GET")
	if err != nil || !ok {
		t.Fatalf("Enforce via dep inheritance: ok=%v err=%v", ok, err)
	}

	effective := EffectiveRolesForUser(uid, tenantID, orgID)
	if len(effective) != 1 || effective[0] != "waiter" {
		t.Fatalf("EffectiveRolesForUser = %v, want [waiter]", effective)
	}

	direct := DirectRolesForUser(uid, tenantID, orgID)
	if len(direct) != 0 {
		t.Fatalf("DirectRolesForUser = %v, want []", direct)
	}
	if err := SyncUserDepsInOrg(uid, tenantID, orgID, []uint64{5}); err != nil {
		t.Fatalf("SyncUserDepsInOrg: %v", err)
	}
	links := getRoleForUserInDomain(genUserByUID(uid), Domain(tenantID, orgID))
	if len(links) != 1 || links[0] != genDepByID(5) {
		t.Fatalf("after sync dep links = %v, want [%s]", links, genDepByID(5))
	}
	if len(DirectRolesForUser(uid, tenantID, orgID)) != 0 {
		t.Fatalf("DirectRolesForUser after sync should be empty")
	}
	ok, err = Enforce(uid, tenantID, orgID, "/api/foo", "GET")
	if err == nil && ok {
		t.Fatalf("expected enforce fail after leaving dep-3")
	}

	if err := AddRoleForUserInDomain(uid, tenantID, orgID, "cashier"); err != nil {
		t.Fatalf("AddRoleForUserInDomain: %v", err)
	}
	direct = DirectRolesForUser(uid, tenantID, orgID)
	if len(direct) != 1 || direct[0] != "cashier" {
		t.Fatalf("DirectRolesForUser after personal assign = %v, want [cashier]", direct)
	}
	effective = EffectiveRolesForUser(uid, tenantID, orgID)
	if len(effective) != 1 || effective[0] != "cashier" {
		t.Fatalf("EffectiveRolesForUser after personal assign = %v, want [cashier]", effective)
	}

	if err := SetDepRoles(depID, tenantID, orgID, []string{"waiter", "cashier"}); err != nil {
		t.Fatalf("SetDepRoles add: %v", err)
	}
	got := GetDepRoles(depID, tenantID, orgID)
	if len(got) != 2 {
		t.Fatalf("GetDepRoles = %v, want 2 roles", got)
	}
	if err := SetDepRoles(depID, tenantID, orgID, []string{"waiter"}); err != nil {
		t.Fatalf("SetDepRoles remove: %v", err)
	}
	got = GetDepRoles(depID, tenantID, orgID)
	if len(got) != 1 || got[0] != "waiter" {
		t.Fatalf("GetDepRoles after diff = %v, want [waiter]", got)
	}

	if err := CleanupDepCasbinPolicies(depID, tenantID, orgID); err != nil {
		t.Fatalf("CleanupDepCasbinPolicies: %v", err)
	}
	if roles := GetDepRoles(depID, tenantID, orgID); len(roles) != 0 {
		t.Fatalf("dep roles after cleanup = %v", roles)
	}
}

package accessctl

import (
	casbin "github.com/casbin/casbin/v3"
)

var (
	enforcer *casbin.SyncedEnforcer
)

func enforce(sub, domain, obj, act string) (bool, error) {
	return enforcer.Enforce(sub, domain, obj, act)
}

func addPolicy(sub, domain, obj, act string) (err error) {
	if _, err = enforcer.AddPolicy(sub, domain, obj, act); err != nil {
		return
	}

	return
}

func removePolicy(sub, domain, obj, act string) (err error) {
	if _, err = enforcer.RemovePolicy(sub, domain, obj, act); err != nil {
		return
	}

	return
}

func getFilteredPolicy(domain string) ([][]string, error) {
	return enforcer.GetFilteredPolicy(1, domain)
}

func HasPolicy(sub, domain, obj, act string) (bool, error) {
	return enforcer.HasPolicy(sub, domain, obj, act)
}

func addRoleForUserInDomain(user, role, domain string) error {
	_, err := enforcer.AddRoleForUserInDomain(user, role, domain)
	if err != nil {
		return err
	}

	return nil
}

func deleteRolesForUserInDomain(user, domain string) (err error) {
	if _, err = enforcer.DeleteRolesForUserInDomain(user, domain); err != nil {
		return
	}

	return
}

func deleteRoleForUserInDomain(user, role, domain string) (err error) {
	if _, err = enforcer.DeleteRoleForUserInDomain(user, role, domain); err != nil {
		return
	}

	return
}

func getRoleForUserInDomain(user, domain string) []string {
	return enforcer.GetRolesForUserInDomain(user, domain)
}

func getUsersForRoleInDomain(role, domain string) []string {
	return enforcer.GetUsersForRoleInDomain(role, domain)
}

func CopyPolicies(fromDomain, toDomain string) error {
	if enforcer == nil || fromDomain == "" || toDomain == "" || fromDomain == toDomain {
		return nil
	}
	policies, err := enforcer.GetFilteredPolicy(1, fromDomain)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if len(p) < 4 || p[0] == "" || p[2] == "" || p[3] == "" {
			continue
		}
		if _, err = enforcer.AddPolicy(p[0], toDomain, p[2], p[3]); err != nil {
			return err
		}
	}
	return nil
}

func CopyGrouping(fromDomain, toDomain string) error {
	if enforcer == nil || fromDomain == "" || toDomain == "" || fromDomain == toDomain {
		return nil
	}
	gs, err := enforcer.GetFilteredGroupingPolicy(2, fromDomain)
	if err != nil {
		return err
	}
	for _, g := range gs {
		if len(g) < 2 || g[0] == "" || g[1] == "" {
			continue
		}
		if _, err = enforcer.AddRoleForUserInDomain(g[0], g[1], toDomain); err != nil {
			return err
		}
	}
	return nil
}

func RemoveDomain(domain string) error {
	if enforcer == nil || domain == "" {
		return nil
	}
	if _, err := enforcer.RemoveFilteredPolicy(1, domain); err != nil {
		return err
	}
	if _, err := enforcer.RemoveFilteredGroupingPolicy(2, domain); err != nil {
		return err
	}
	return nil
}

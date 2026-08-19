package service

import (
	"strings"

	"github.com/liuhengloveyou/passport/v4/accessctl"
	"github.com/liuhengloveyou/passport/v4/cache"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/protos"
)

func OrgCreate(tenantID uint64, name string) (uint64, error) {
	name = strings.TrimSpace(name)
	if tenantID == 0 || name == "" {
		return 0, common.ErrParam
	}
	tenant, err := TenantGetByIDService(tenantID)
	if err != nil {
		return 0, err
	}

	if got, err := dao.OrgGetByTenantName(tenantID, name); err != nil {
		common.Logger.Sugar().Errorf("OrgCreate get ERR: %v", err)
		return 0, common.ErrService
	} else if got != nil && got.ID > 0 {
		return 0, common.ErrOrgNameDup
	}

	existing, err := dao.OrgListByTenant(tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgCreate list ERR: %v", err)
		return 0, common.ErrService
	}

	id, err := dao.OrgInsert(&protos.Organization{TenantID: tenantID, Name: name})
	if err != nil || id == 0 {
		common.Logger.Sugar().Errorf("OrgCreate insert ERR: %v %v", id, err)
		return 0, common.MapPostgresOrgInsertError(err)
	}

	if tenant.UID > 0 {
		if err = dao.OrgMemberInsert(id, tenant.UID, tenantID); err != nil {
			common.Logger.Sugar().Errorf("OrgCreate member ERR: %v", err)
			return 0, common.ErrService
		}
		cache.DelOrgMemberCache(id, tenant.UID)
		if err = accessctl.AddRoleForUserInDomain(tenant.UID, tenantID, id, "root"); err != nil {
			common.Logger.Sugar().Errorf("OrgCreate root ERR: %v", err)
			return 0, common.ErrService
		}
	}

	fromDomain := accessctl.LegacyTenantDomain(tenantID)
	for i := range existing {
		if existing[i].ID > 0 {
			fromDomain = accessctl.Domain(tenantID, existing[i].ID)
			break
		}
	}
	if err = accessctl.CopyPolicies(fromDomain, accessctl.Domain(tenantID, id)); err != nil {
		common.Logger.Sugar().Warnf("OrgCreate copy policy ERR: %v", err)
	}
	return id, nil
}

func OrgGet(orgID uint64) (*protos.Organization, error) {
	if orgID == 0 {
		return nil, common.ErrOrgRequired
	}
	if org := cache.GetOrgCache(orgID); org != nil {
		return org, nil
	}
	org, err := dao.OrgGetByID(orgID)
	if err != nil {
		return nil, common.ErrService
	}
	if org == nil || org.ID == 0 {
		return nil, common.ErrOrgNotFound
	}
	cache.SetOrgCache(org)
	return org, nil
}

func RequireOrg(tenantID, orgID uint64) (*protos.Organization, error) {
	if tenantID == 0 || orgID == 0 {
		return nil, common.ErrOrgRequired
	}
	org, err := OrgGet(orgID)
	if err != nil {
		return nil, err
	}
	if org.TenantID != tenantID {
		return nil, common.ErrNoAuth
	}
	return org, nil
}

// UserInOrg 校验用户是该租户下该组织的成员（org / 成员关系走内存缓存）。
func UserInOrg(uid, tenantID, orgID uint64) error {
	if uid == 0 {
		return common.ErrNoAuth
	}
	if _, err := RequireOrg(tenantID, orgID); err != nil {
		return err
	}
	if in, hit := cache.GetOrgMemberCache(orgID, uid); hit {
		if in {
			return nil
		}
		return common.ErrNoAuth
	}
	ok, err := dao.OrgMemberExists(orgID, uid)
	if err != nil {
		return common.ErrService
	}
	if !ok {
		return common.ErrNoAuth
	}
	cache.SetOrgMemberCache(orgID, uid, true)
	return nil
}

func OrgGetByTenantName(tenantID uint64, name string) (*protos.Organization, error) {
	name = strings.TrimSpace(name)
	if tenantID == 0 || name == "" {
		return nil, common.ErrParam
	}
	org, err := dao.OrgGetByTenantName(tenantID, name)
	if err != nil {
		return nil, common.ErrService
	}
	return org, nil
}

func OrgListByTenant(tenantID uint64) ([]protos.Organization, error) {
	rr, err := dao.OrgListByTenant(tenantID)
	if err != nil {
		return nil, common.ErrService
	}
	return rr, nil
}

func OrgListByUser(uid, tenantID uint64) ([]protos.Organization, error) {
	if uid == 0 || tenantID == 0 {
		return nil, common.ErrParam
	}
	rr, err := dao.OrgListByUser(uid, tenantID)
	if err != nil {
		return nil, common.ErrService
	}
	return rr, nil
}

func OrgAddMember(orgID, uid, tenantID uint64) error {
	if orgID == 0 || uid == 0 || tenantID == 0 {
		return common.ErrParam
	}
	if _, err := RequireOrg(tenantID, orgID); err != nil {
		return err
	}
	if err := dao.OrgMemberInsert(orgID, uid, tenantID); err != nil {
		return common.ErrService
	}
	cache.DelOrgMemberCache(orgID, uid)
	return nil
}

func OrgRemoveMember(orgID, uid, tenantID uint64) error {
	if orgID == 0 || uid == 0 || tenantID == 0 {
		return common.ErrParam
	}
	if _, err := RequireOrg(tenantID, orgID); err != nil {
		return err
	}
	if err := accessctl.DeleteRolesForUserInDomain(uid, tenantID, orgID); err != nil {
		common.Logger.Sugar().Errorf("OrgRemoveMember roles ERR: %v", err)
		return common.ErrService
	}
	if err := dao.OrgMemberDelete(orgID, uid); err != nil {
		return common.ErrService
	}
	cache.DelOrgMemberCache(orgID, uid)
	return nil
}

func OrgDelete(tenantID, orgID uint64) error {
	if _, err := RequireOrg(tenantID, orgID); err != nil {
		return err
	}
	uids, err := dao.OrgMemberUIDs(orgID)
	if err != nil {
		return common.ErrService
	}
	if err = accessctl.RemoveDomain(accessctl.Domain(tenantID, orgID)); err != nil {
		common.Logger.Sugar().Errorf("OrgDelete domain ERR: %v", err)
		return common.ErrService
	}
	if err = dao.DepartmentDeleteByOrg(tenantID, orgID); err != nil {
		return common.ErrService
	}
	if err = dao.OrgMemberDeleteByOrg(orgID); err != nil {
		return common.ErrService
	}
	if err = dao.OrgDelete(orgID, tenantID); err != nil {
		return common.ErrService
	}
	cache.DelOrgCache(orgID)
	for _, uid := range uids {
		cache.DelOrgMemberCache(orgID, uid)
	}
	return nil
}

func UserHasRoleInTenant(uid, tenantID uint64, role string) bool {
	if uid == 0 || tenantID == 0 || role == "" {
		return false
	}
	orgs, err := dao.OrgListByUser(uid, tenantID)
	if err != nil {
		return false
	}
	for i := range orgs {
		roles := accessctl.GetRoleForUserInDomain(uid, tenantID, orgs[i].ID)
		for _, r := range roles {
			if r == role {
				return true
			}
		}
	}
	return false
}

func AddRoleForUserInAllOrgs(uid, tenantID uint64, role string) error {
	orgs, err := dao.OrgListByTenant(tenantID)
	if err != nil {
		return common.ErrService
	}
	for i := range orgs {
		if err = dao.OrgMemberInsert(orgs[i].ID, uid, tenantID); err != nil {
			return common.ErrService
		}
		cache.DelOrgMemberCache(orgs[i].ID, uid)
		if err = accessctl.AddRoleForUserInDomain(uid, tenantID, orgs[i].ID, role); err != nil {
			return common.ErrService
		}
	}
	return nil
}

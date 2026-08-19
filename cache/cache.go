package cache

import (
	"fmt"

	"github.com/liuhengloveyou/passport/v4/protos"
)

const (
	tenantCache    = "tenant-%d"
	orgCache       = "org-%d"
	orgMemberCache = "org-member-%d-%d"
)

var defaultCache = NewExpiredMap()

// SetTenantCache 将租户信息写入内存缓存。
func SetTenantCache(m *protos.Tenant) {
	defaultCache.Set(tenantCacheKey(m.ID), m, 3600)
}

// GetTenantCache 按租户ID读取缓存中的租户信息。
func GetTenantCache(id uint64) *protos.Tenant {
	if ok, v := defaultCache.Get(tenantCacheKey(id)); ok {
		return v.(*protos.Tenant)
	}
	return nil
}

// DelTenantCache 删除指定租户ID的缓存。
func DelTenantCache(id uint64) {
	defaultCache.Delete(tenantCacheKey(id))
}

func tenantCacheKey(id uint64) string {
	return fmt.Sprintf(tenantCache, id)
}

func SetOrgCache(m *protos.Organization) {
	if m == nil || m.ID == 0 {
		return
	}
	defaultCache.Set(orgCacheKey(m.ID), m, 3600)
}

func GetOrgCache(id uint64) *protos.Organization {
	if ok, v := defaultCache.Get(orgCacheKey(id)); ok {
		return v.(*protos.Organization)
	}
	return nil
}

func DelOrgCache(id uint64) {
	defaultCache.Delete(orgCacheKey(id))
}

func orgCacheKey(id uint64) string {
	return fmt.Sprintf(orgCache, id)
}

func SetOrgMemberCache(orgID, uid uint64, in bool) {
	if orgID == 0 || uid == 0 || !in {
		return
	}
	defaultCache.Set(orgMemberCacheKey(orgID, uid), true, 300)
}

func GetOrgMemberCache(orgID, uid uint64) (in bool, hit bool) {
	ok, v := defaultCache.Get(orgMemberCacheKey(orgID, uid))
	if !ok {
		return false, false
	}
	return v.(bool), true
}

func DelOrgMemberCache(orgID, uid uint64) {
	defaultCache.Delete(orgMemberCacheKey(orgID, uid))
}

func orgMemberCacheKey(orgID, uid uint64) string {
	return fmt.Sprintf(orgMemberCache, orgID, uid)
}

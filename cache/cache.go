package cache

import (
	"fmt"

	"github.com/liuhengloveyou/passport/protos"
)

const (
	// 单个tenant信息缓存
	tenantCache = "tenant-%d"
)

var defaultCache = NewExpiredMap()

func SetTenantCache(m *protos.Tenant) {
	defaultCache.Set(tenantCacheKey(m.ID), m, 3600)
}

func GetTenantCache(id uint64) *protos.Tenant {
	if ok, v := defaultCache.Get(tenantCacheKey(id)); ok {
		return v.(*protos.Tenant)
	}
	return nil
}

func DelTenantCache(id uint64) {
	defaultCache.Delete(tenantCacheKey(id))
}

func tenantCacheKey(id uint64) string {
	return fmt.Sprintf(tenantCache, id)
}

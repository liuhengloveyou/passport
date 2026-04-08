package cache

import (
	"fmt"

	"github.com/liuhengloveyou/passport/v3/protos"
)

const (
	// 单个tenant信息缓存
	tenantCache = "tenant-%d"
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

// tenantCacheKey 生成租户缓存键。
func tenantCacheKey(id uint64) string {
	return fmt.Sprintf(tenantCache, id)
}

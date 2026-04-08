package cache

import (
	"sync"
	"time"
)

type data struct {
	val       any
	expiredAt int64 // 过期时间戳,秒级时间戳
}

type ExpiredMap struct {
	m   map[string]*data // 键值映射
	lck sync.RWMutex
}

// NewExpiredMap 创建一个带过期能力的内存键值映射。
func NewExpiredMap() *ExpiredMap {
	return &ExpiredMap{
		m: make(map[string]*data),
	}
}

// Set 写入缓存值并设置过期时间，支持相对TTL秒数或绝对过期时间戳。
func (e *ExpiredMap) Set(key string, value any, aliveSecond int64) {
	if aliveSecond <= 0 {
		return
	}

	now := time.Now().Unix()
	expiredAt := aliveSecond
	// 兼容两种传参：
	// 1) 相对TTL（秒）: 3600
	// 2) 绝对过期时间戳: time.Now().Unix()+3600
	if aliveSecond <= now {
		expiredAt = now + aliveSecond
	}

	e.lck.Lock()
	e.m[key] = &data{
		val:       value,
		expiredAt: expiredAt,
	}
	e.lck.Unlock()
}

// Get 读取缓存值；若已过期会惰性删除并返回未命中。
func (e *ExpiredMap) Get(key string) (found bool, value any) {
	e.lck.RLock()
	val, found := e.m[key]
	e.lck.RUnlock()
	if !found {
		return
	}
	if val.expiredAt <= time.Now().Unix() {
		e.Delete(key)
		return false, nil
	}

	return true, val.val
}

// Delete 删除指定key的缓存项。
func (e *ExpiredMap) Delete(key string) {
	e.lck.Lock()
	delete(e.m, key)
	e.lck.Unlock()
}

// Length 返回当前缓存项数量（包含尚未访问到的过期项）。
func (e *ExpiredMap) Length() int {
	e.lck.RLock()
	defer e.lck.RUnlock()

	return len(e.m)
}

// TTL 返回指定key剩余秒数；不存在或已过期时返回-1。
func (e *ExpiredMap) TTL(key string) int64 {
	e.lck.RLock()
	val, found := e.m[key]
	e.lck.RUnlock()
	if !found {
		return -1
	}
	ttl := val.expiredAt - time.Now().Unix()
	if ttl < 0 {
		e.Delete(key)
		return -1
	}

	return ttl
}

// Clear 清空全部缓存项。
func (e *ExpiredMap) Clear() {
	e.lck.Lock()
	defer e.lck.Unlock()

	e.m = make(map[string]*data)
}

// DoForEach 遍历缓存项，对过期项执行惰性删除。
func (e *ExpiredMap) DoForEach(handler func(string, any)) {
	e.lck.Lock()
	defer e.lck.Unlock()
	for k, v := range e.m {
		if v.expiredAt <= time.Now().Unix() {
			delete(e.m, k)
			continue
		}
		handler(k, v.val)
	}
}

// DoForEachWithBreak 遍历缓存项，handler返回true时中断。
func (e *ExpiredMap) DoForEachWithBreak(handler func(string, any) bool) {
	e.lck.Lock()
	defer e.lck.Unlock()

	for k, v := range e.m {
		if v.expiredAt <= time.Now().Unix() {
			delete(e.m, k)
			continue
		}
		if handler(k, v.val) {
			break
		}
	}
}

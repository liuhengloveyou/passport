package service

import "github.com/liuhengloveyou/passport/v4/cache"

var deparmentCache *cache.ExpiredMap = nil

func init() {
	// gob.Register(protos.MiniAppSessionInfo{})
	deparmentCache = cache.NewExpiredMap()
}

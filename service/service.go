package service

import (
	"encoding/gob"
	"github.com/liuhengloveyou/passport/common"

	"github.com/liuhengloveyou/passport/protos"
)

var (
	MiniAppService *miniAppService
)

func init() {
	gob.Register(protos.MiniAppSessionInfo{})

	MiniAppService = &miniAppService{
		AppID:     common.ServConfig.WxMiniApp.AppID,
		AppSecret: common.ServConfig.WxMiniApp.AppSecret,
	}
}

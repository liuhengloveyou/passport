package service

import (
	"encoding/gob"
	"github.com/liuhengloveyou/passport/common"
	"go.uber.org/zap"
)

var logger       *zap.SugaredLogger

func init() {
	gob.Register(MiniAppUserInfo{})

	logger = common.Logger.Sugar()
}

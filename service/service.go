package service

import (
	"encoding/gob"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	gob.Register(MiniAppUserInfo{})
}

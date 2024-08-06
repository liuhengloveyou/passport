package service

import (
	"encoding/gob"

	"github.com/liuhengloveyou/passport/protos"
)

func init() {
	gob.Register(protos.MiniAppSessionInfo{})

}

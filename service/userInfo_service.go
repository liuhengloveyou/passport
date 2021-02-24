package service

import (
	"fmt"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	"github.com/liuhengloveyou/passport/dao"
)

func GetUserInfoService(uid uint64) (r protos.User, e error) {
	if uid <= 0 {
		e = fmt.Errorf("uid nil")
		return
	}

	model := &protos.UserReq{
		UID: uid,
	}

	var rr []protos.User
	if rr, e = dao.UserSelect(model, 1, 1); e != nil {
		common.Logger.Sugar().Errorf("GetUserInfoService DB ERR: %v\n", e)
		return
	}

	if rr != nil && len(rr) == 1 {
		rr[0].Password = ""
		return rr[0], nil
	}
	
	return
}

func GetBusinessUserInfoService(uid uint64, models interface{}) (e error) {
	if uid <= 0 {
		e = fmt.Errorf("uid nil")
		return
	}

	model := &protos.UserReq{
		UID: uid,
	}

	e = dao.BusinessSelect(model, models, 1, 1)

	return
}

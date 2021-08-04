package service

import (
	"fmt"
	"github.com/liuhengloveyou/passport/protos"

	. "github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
)

func AuthPWDService(uid uint64, pwd string) (auth bool, e error) {
	if uid <= 0 {
		return false, fmt.Errorf("用户错误")
	}

	if pwd == "" {
		return false, fmt.Errorf("用户密码为空")
	}

	var rr []protos.User
	if rr, e = dao.UserSelect(&protos.UserReq{UID: uid}, 1, 1); e != nil {
		return false, e
	}

	if rr == nil || len(rr) != 1 {
		return false, nil
	}

	if rr[0].Password != EncryPWD(pwd) {
		return false, nil
	}

	return true, nil
}

package service

import (
	"fmt"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	. "github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"

	validator "github.com/go-playground/validator/v10"
)

func UserLogin(user *protos.UserReq) (one *protos.User, e error) {
	if user == nil {
		return nil, fmt.Errorf("请求参数错误")
	}
	if user.Password == "" {
		return nil, fmt.Errorf("密码不能为空")
	}

	if user.Cellphone == "" && user.Nickname == "" {
		return nil, fmt.Errorf("请求参数错误")
	}

	userPreTreat(user)

	if err := Validate.Struct(user); err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			switch errs[0].Field() {
			case "Cellphone":
				err = fmt.Errorf("手机号格式错误")
			case "Email":
				err = fmt.Errorf("邮箱格式错误")
			case "Nickname":
				err = fmt.Errorf("昵称格式错误")
			case "Password":
				err = fmt.Errorf("密码长度必须大于6")
			}
		}

		return nil, err
	}

	if user.Cellphone != "" {
		one, e = loginByCellphone(user)
	} else if user.Email != "" {
		one, e = loginByEmail(user)
	} else if user.Nickname != "" {
		one, e = loginByNickname(user)
	}

	if e != nil {
		common.Logger.Sugar().Errorf("db err: %v\n", e.Error())
		e = common.ErrService
	}

	if one == nil {
		return nil, common.ErrLogin
	}

	if EncryPWD(user.Password) != one.Password {
		return nil, common.ErrPWD
	}
	one.Password = ""
	one.AddTime = nil
	one.UpdateTime = nil

	// tenant
	if common.ServConfig.IsTenant && one.TenantID > 0 {
		if one.Tenant, e = dao.TenantGetByID(one.TenantID); e != nil {
			common.Logger.Sugar().Errorf("TenantGetByID ERR: ", e)
		}
		one.Tenant.Configuration = nil
		one.Tenant.Info = nil
		one.Tenant.AddTime = nil
		one.Tenant.UpdateTime = nil
	}

	return
}

func loginByCellphone(p *protos.UserReq) (one *protos.User, e error) {
	rr, e := dao.UserSelect(p, 1, 1)
	if e != nil {
		return
	}

	if len(rr) == 0 {
		return // 不存在用户
	}

	return &rr[0], nil
}

func loginByEmail(p *protos.UserReq) (one *protos.User, e error) {
	return nil, nil
}

func loginByNickname(p *protos.UserReq) (one *protos.User, e error) {
	rr, e := dao.UserSelect(p, 1, 1)
	if e != nil {
		return
	}

	if len(rr) == 0 {
		return // 不存在用户
	}

	return &rr[0], nil
}

package service

import (
	"fmt"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
	"github.com/liuhengloveyou/passport/sms"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4/zero"

	validator "github.com/go-playground/validator/v10"
)

/*
微信公众号oauth2登录
*/
func UserLoginByWeixin(req *protos.UserReq) (one *protos.User, e error) {
	if req == nil || len(req.WxOpenId) == 0 {
		return nil, common.ErrParam
	}

	userPreTreat(req)

	one, e = dao.UserQueryOne(req)
	if e != nil {
		common.Logger.Error("db err: ", zap.Error(e), zap.Any("req", req))
		e = common.ErrService
	}

	if one == nil {
		common.Logger.Error("login user nil: ", zap.Any("req", req))
		return nil, common.ErrLogin
	}

	disabled, ok := one.Ext["disabled"].(float64)
	if ok && int8(disabled) == 1 {
		common.Logger.Sugar().Errorf("login Disabled ERR: [%v] \n", one.Ext)
		return nil, common.ErrDisable
	}

	now := time.Now()
	one.LoginTime = &now

	rows, err := dao.UserUpdateLoginTime(one.UID, one.LoginTime)
	if err != nil || rows != 1 {
		common.Logger.Error("UserUpdateLoginTime ERR: ", zap.Int64("row", rows), zap.Error(e), zap.Any("req", req))
		e = common.ErrService
		return
	}

	one.Password = ""
	one.Ext = nil
	one.Roles = nil
	one.Departments = nil

	// tenant
	if one.TenantID > 0 {
		if one.Tenant, e = dao.TenantGetByID(one.TenantID); e != nil {
			common.Logger.Sugar().Errorf("TenantGetByID ERR: ", e)
			one.TenantID, e = 0, nil // 没有租户也可以登录成功
		}
		if one.Tenant != nil {
			one.Tenant.Configuration = nil
			one.Tenant.Info = nil
			one.Tenant.CreateTime = nil
			one.Tenant.UpdateTime = nil
		}
	}
	common.Logger.Sugar().Infof("UserLoginByWeixin: %#v\n", one)

	return
}

func UserLogin(user *protos.UserReq) (one *protos.User, e error) {
	if user == nil ||
		(len(user.Password) == 0 && len(user.SmsCode) == 0) ||
		(len(user.Cellphone) == 0 && len(user.Nickname) == 0 && len(user.Email) == 0) {
		return nil, common.ErrParam
	}

	userPreTreat(user)

	if err := common.Validate.Struct(user); err != nil {
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

	if len(user.Cellphone) > 0 && len(user.SmsCode) > 0 {
		one, e = loginBySmsCode(user)
	} else if user.Cellphone != "" {
		one, e = loginByCellphone(user)
	} else if user.Nickname != "" {
		one, e = loginByNickname(user)
	} else if user.Email != "" {
		one, e = loginByEmail(user)
	}

	if e != nil {
		common.Logger.Sugar().Errorf("db err: %v\n", e.Error())
		e = common.ErrService
	}

	if one == nil {
		common.Logger.Sugar().Errorf("login user nil: %v\n", user)
		return nil, common.ErrLogin
	}

	disabled, ok := one.Ext["disabled"].(float64)
	if ok && int8(disabled) == 1 {
		common.Logger.Sugar().Errorf("login Disabled ERR: [%v] \n", one.Ext)
		return nil, common.ErrDisable
	}

	if len(user.Password) > 0 && (common.EncryPWD(user.Password) != one.Password) {
		common.Logger.Sugar().Errorf("login pwd ERR: [%v] [%v] [%v]\n", user.Password, common.EncryPWD(user.Password), one.Password)
		return nil, common.ErrPWD
	}

	now := time.Now()
	one.LoginTime = &now

	rows, err := dao.UserUpdateLoginTime(one.UID, one.LoginTime)
	if err != nil || rows != 1 {
		common.Logger.Sugar().Errorf("UserUpdateLoginTime db err: %v %v\n", e, rows)
		e = common.ErrService
		return
	}

	one.Password = ""
	one.Ext = nil
	one.Roles = nil
	one.Departments = nil
	if one.Tenant != nil {
		one.Tenant.Info = nil
	}

	// tenant
	if one.TenantID > 0 {
		if one.Tenant, e = dao.TenantGetByID(one.TenantID); e != nil {
			common.Logger.Sugar().Errorf("TenantGetByID ERR: ", e)
			one.TenantID, e = 0, nil // 没有租户也可以登录成功
		}
		if one.Tenant != nil {
			one.Tenant.Configuration = nil
			one.Tenant.Info = nil
			one.Tenant.CreateTime = nil
			one.Tenant.UpdateTime = nil
		}
	}
	common.Logger.Sugar().Errorf("UserLogin: %#v\n", one)

	return
}

func loginBySmsCode(p *protos.UserReq) (one *protos.User, e error) {
	if len(p.Cellphone) == 0 || len(p.SmsCode) == 0 {
		return nil, common.ErrParam
	}

	e = sms.CheckSmsCode(p.Cellphone, p.SmsCode)
	if e != nil && e != sms.ErrSmsNotInit {
		return nil, e
	}

	one, e = dao.UserQueryOne(p)

	// 短信登录，用户不存在则自动注册
	if one == nil {
		id, e := dao.UserInsert(p, nil)
		if e != nil {
			common.Logger.Error("loginBySmsCode.UserInsert ERR: ", zap.Error(e))
			return nil, e
		}
		cellphone := zero.StringFrom(p.Cellphone)
		one = &protos.User{
			UID:       uint64(id),
			Cellphone: &cellphone,
		}
	}

	return
}

func loginByCellphone(p *protos.UserReq) (one *protos.User, e error) {
	one, e = dao.UserQueryOne(p)

	return
}

func loginByEmail(p *protos.UserReq) (one *protos.User, e error) {
	one, e = dao.UserQueryOne(p)

	return
}

func loginByNickname(p *protos.UserReq) (one *protos.User, e error) {
	one, e = dao.UserQueryOne(p)

	return
}

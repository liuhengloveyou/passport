package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/sms"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4/zero"

	validator "github.com/go-playground/validator/v10"
)

/*
UserLoginByOpenID 微信 / 支付宝等渠道 OAuth 登录。
身份键写入并查询 users.wx_openid（历史列名；支付宝 openId 也存此字段）。
新用户昵称固定为 wx_{openid} / ali_{openid}，与 openid 一一对应，避免撞库重试。
*/
func UserLoginByOpenID(req *protos.UserReq) (one *protos.User, e error) {
	if req == nil || len(req.WxOpenId) == 0 {
		return nil, common.ErrParam
	}

	userPreTreat(req)
	kind := channelKindFromExt(req.Ext)

	// 只按 openid 查
	one, e = dao.UserQueryOne(&protos.UserReq{WxOpenId: req.WxOpenId})
	if e != nil {
		common.Logger.Error("UserLoginByOpenID db err", zap.Error(e), zap.String("kind", kind), zap.Any("req", req))
		return nil, common.ErrService
	}

	if one == nil {
		ins := &protos.UserReq{
			WxOpenId:  req.WxOpenId,
			Nickname:  channelAccountNick(kind, req.WxOpenId),
			Password:  common.EncryPWD(req.WxOpenId),
			AvatarURL: req.AvatarURL,
			Gender:    req.Gender,
			Ext:       req.Ext,
		}
		if ins.Ext == nil {
			ins.Ext = protos.MapStruct{}
		}
		applyChannelExt(ins.Ext, kind)

		id, ierr := dao.UserInsert(ins, nil)
		if ierr != nil {
			common.Logger.Error("UserLoginByOpenID auto register ERR",
				zap.Error(ierr), zap.String("kind", kind), zap.Any("req", req))
			return nil, common.ErrService
		}
		one, e = dao.UserQueryByID(uint64(id))
		if e != nil || one == nil {
			common.Logger.Error("UserLoginByOpenID query after insert ERR", zap.Error(e), zap.String("kind", kind))
			return nil, common.ErrService
		}
		common.Logger.Sugar().Infof("UserLoginByOpenID registered uid=%d kind=%s openid=%s nick=%s\n",
			one.UID, kind, req.WxOpenId, ins.Nickname)
	} else {
		if one.Ext == nil {
			one.Ext = protos.MapStruct{}
		}
		applyChannelExt(one.Ext, kind)
		if req.AvatarURL != "" && (one.AvatarURL == nil || one.AvatarURL.String == "") {
			a := zero.StringFrom(req.AvatarURL)
			one.AvatarURL = &a
		}
	}

	disabled, ok := one.Ext["disabled"].(float64)
	if ok && protos.UserDisableStatus(int8(disabled)) == protos.UserDisabled {
		common.Logger.Sugar().Errorf("login Disabled ERR: [%v] \n", one.Ext)
		return nil, common.ErrDisable
	}

	now := time.Now()
	one.LoginTime = &now

	rows, err := dao.UserUpdateLoginTime(one.UID, one.LoginTime)
	if err != nil || rows != 1 {
		common.Logger.Error("UserUpdateLoginTime ERR: ", zap.Int64("row", rows), zap.Error(err), zap.Any("req", req))
		e = common.ErrService
		return
	}

	one.Password = ""
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
	common.Logger.Sugar().Infof("UserLoginByOpenID ok uid=%d kind=%s openid=%s\n", one.UID, kind, req.WxOpenId)

	return
}

func channelKindFromExt(ext protos.MapStruct) string {
	if ext == nil {
		return "wechat"
	}
	if k, _ := ext["kind"].(string); k == "alipay" || k == "wechat" {
		return k
	}
	if v, ok := ext["alipay"]; ok && truthyExt(v) {
		return "alipay"
	}
	if v, ok := ext["wechat"]; ok && truthyExt(v) {
		return "wechat"
	}
	return "wechat"
}

func truthyExt(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}

func applyChannelExt(ext protos.MapStruct, kind string) {
	if ext == nil {
		return
	}
	ext["kind"] = kind
	switch kind {
	case "alipay":
		ext["alipay"] = 1
		ext["wechat"] = 0
	default:
		ext["wechat"] = 1
		ext["alipay"] = 0
	}
}

func channelAccountNick(kind, openID string) string {
	prefix := "wx_"
	if kind == "alipay" {
		prefix = "ali_"
	}
	nick := prefix + openID
	// users.nickname 最长 64
	if len(nick) > 64 {
		nick = nick[:64]
	}
	return nick
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
	if ok && protos.UserDisableStatus(int8(disabled)) == protos.UserDisabled {
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

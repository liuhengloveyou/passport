package sms

import (
	"strings"
	"time"

	cache "github.com/liuhengloveyou/passport/cache"

	"github.com/liuhengloveyou/go-errors"
)

var (
	ErrSmsDriver   = errors.NewError(-4000, "不存在的短信驱动")
	ErrSmsNotInit  = errors.NewError(-4001, "末启用短信功能")
	ErrSmsExist    = errors.NewError(-4002, "短信已发送")
	ErrSmsCheckErr = errors.NewError(-4003, "短信验证码错误")
)

type factoryFun func(config map[string]interface{}) Sms

// sms接口
type Sms interface {
	// 发送用户注册/添加验证码
	// 返回验证码
	SendUserAddSms(phoneNumber string, aliveSecond int64) (code string, err error)

	// 发送用户登录验证码
	// 返回验证码
	SendUserLoginSms(phoneNumber string, aliveSecond int64) (code string, err error)

	// 发送用户找回密码验证码
	// 返回验证码
	SendGetBackPwdSms(phoneNumber string, aliveSecond int64) (code string, err error)

	// 发送微信公众号验证后绑定手机号验证码
	// 返回验证码
	SendWxBindSms(phoneNumber string, aliveSecond int64) (code string, err error)
}

var smsFactoryByName = make(map[string]factoryFun)

var (
	defaultSms Sms               = nil
	codeCache  *cache.ExpiredMap = nil
)

func Register(name string, f factoryFun) {
	smsFactoryByName[name] = f
}

func Init(name string, config map[string]interface{}) error {
	if f, ok := smsFactoryByName[name]; ok {
		defaultSms = f(config)
	} else {
		return ErrSmsExist
	}

	if defaultSms != nil {
		codeCache = cache.NewExpiredMap()
	}

	return nil
}

func CheckSmsCode(phoneNumber, code string) error {
	if defaultSms == nil {
		return ErrSmsNotInit
	}

	found, value := codeCache.Get(phoneNumber)
	if !found {
		return ErrSmsCheckErr
	}

	if strings.Compare(code, value.(string)) != 0 {
		return ErrSmsCheckErr
	}

	return nil
}

func SendUserAddSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	if defaultSms == nil {
		return "", ErrSmsNotInit
	}

	if codeCache.TTL(phoneNumber) >= 0 {
		return "", ErrSmsExist
	}
	if aliveSecond == 0 {
		aliveSecond = 60
	}

	code, err = defaultSms.SendUserAddSms(phoneNumber, aliveSecond)
	codeCache.Set(phoneNumber, code, time.Now().Unix()+aliveSecond)

	return
}

func SendUserLoginSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	if defaultSms == nil {
		return "", ErrSmsNotInit
	}

	if codeCache.TTL(phoneNumber) >= 0 {
		return "", ErrSmsExist
	}
	if aliveSecond == 0 {
		aliveSecond = 60
	}

	code, err = defaultSms.SendUserAddSms(phoneNumber, aliveSecond)
	if err != nil {
		return
	} else {
		codeCache.Set(phoneNumber, code, time.Now().Unix()+aliveSecond)
	}

	return
}

func SendGetBackPwdSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	if defaultSms == nil {
		return "", ErrSmsNotInit
	}

	if codeCache.TTL(phoneNumber) >= 0 {
		return "", ErrSmsExist
	}
	if aliveSecond == 0 {
		aliveSecond = 60
	}

	code, err = defaultSms.SendGetBackPwdSms(phoneNumber, aliveSecond)
	if err != nil {
		return
	} else {
		codeCache.Set(phoneNumber, code, time.Now().Unix()+aliveSecond)
	}

	return
}

func SendWxBindSms(phoneNumber string, aliveSecond int64) (code string, err error) {
	if defaultSms == nil {
		return "", ErrSmsNotInit
	}

	if codeCache.TTL(phoneNumber) >= 0 {
		return "", ErrSmsExist
	}
	if aliveSecond == 0 {
		aliveSecond = 60
	}

	code, err = defaultSms.SendWxBindSms(phoneNumber, aliveSecond)
	if err != nil {
		return
	} else {
		codeCache.Set(phoneNumber, code, time.Now().Unix()+aliveSecond)
	}

	return
}

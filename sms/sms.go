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
}

var smsFactoryByName = make(map[string]factoryFun)

var (
	defaultSms Sms = nil

	codeCache *cache.ExpiredMap = nil
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
		return nil
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

	code, err = defaultSms.SendUserAddSms(phoneNumber, aliveSecond)
	codeCache.Set(phoneNumber, code, time.Now().Unix()+aliveSecond)

	return
}

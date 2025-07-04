package common

import (
	"github.com/liuhengloveyou/go-errors"
)

var (
	ErrOK           = &errors.Error{Code: 0, Message: "OK"}
	ErrNull         = &errors.Error{Code: 1, Message: "数据为空"}
	ErrParam        = errors.NewError(-1000, "请求参数错误")
	ErrService      = errors.NewError(-1001, "服务错误")
	ErrSession      = errors.NewError(-1002, "会话错误")
	ErrNoLogin      = errors.NewError(-1003, "请登录")
	ErrNoAuth       = errors.NewError(-1004, "没有权限")
	ErrPgDupKey     = errors.NewError(-1005, "重复记录")
	ErrLogin        = errors.NewError(-1006, "登录失败")
	ErrPWD          = errors.NewError(-1007, "密码不正确")
	ErrDisable      = errors.NewError(-1008, "账号已停用")
	ErrUserNmae     = errors.NewError(-1009, "账号为空")
	ErrPWDNil       = errors.NewError(-1010, "密码为空")
	ErrPhoneDup     = errors.NewError(-1011, "手机号码重复")
	ErrEmailDup     = errors.NewError(-1012, "邮箱重复")
	ErrNickDup      = errors.NewError(-1013, "昵称重复")
	ErrWxOpenidDup  = errors.NewError(-1014, "账号已经存在")
	ErrModify       = errors.NewError(-1015, "更新用户信息失败") //
	ErrUserNotFound = errors.NewError(-1016, "用户不存在")

	ErrTenantNotFound           = errors.NewError(-2000, "租户不存在")
	ErrTenantNameNull           = errors.NewError(-2001, "租户名字为空")
	ErrTenantTypeNull           = errors.NewError(-2002, "租户类型为空")
	ErrTenantLimit              = errors.NewError(-2003, "账号只能属于一个租户")
	ErrTenantAddERR             = errors.NewError(-2004, "添加租户失败")
	ErrTenantAdminCellphoneNull = errors.NewError(-2005, "管理员手机号为空")
	ErrTenantAdminPasswordNull  = errors.NewError(-2006, "管理员密码为空")

	ErrWxService = errors.NewError(-3000, "微信接口返回错误")

	ErrTenantSetParent   = errors.NewError(-104000, "设置租户父级失败")
	ErrTenantCircularRef = errors.NewError(-104001, "循环设置租户父级")
	ErrTenantRoot        = errors.NewError(-104002, "不能给Root租户设置父级")
	ErrTenantSame        = errors.NewError(-104003, "不能设置相同账号为父级")
)

package common

import (
	"github.com/liuhengloveyou/go-errors"
)

var (
	ErrOK        = errors.Error{Code: 0, Message: "OK"}
	ErrParam     = errors.NewError(-1000, "请求参数错误")
	ErrService   = errors.NewError(-1001, "服务错误")
	ErrSession   = errors.NewError(-1002, "会话错误")
	ErrNoLogin   = errors.NewError(-1003, "请登录")
	ErrNoAuth    = errors.NewError(-1004, "没有权限")
	ErrMysql1062 = errors.NewError(-1005, "重复记录")

	ErrTenantNameNull = errors.NewError(-2000, "名字为空")
	ErrTenantTypeNull = errors.NewError(-2001, "类型为空")
)

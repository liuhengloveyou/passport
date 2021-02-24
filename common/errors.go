package common

import (
	"github.com/liuhengloveyou/go-errors"
)

var (
	ErrOK      = errors.Error{Code: 0, Message: "OK"}
	ErrParam   = errors.NewError(-1000, "请求参数错误")
	ErrService = errors.NewError(-1001, "服务错误")
	ErrNoLogin = errors.NewError(-1002, "请登录")
	ErrNoAuth  = errors.NewError(-1003, "没有权限")
)

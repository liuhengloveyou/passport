package common

import ()

type TaskQueue interface {
	Push() // 添加一个任务
	Pop()  // 取一个任务
}

package main

import (
	"github.com/gin-gonic/gin"
	passport "github.com/liuhengloveyou/passport/face"
	passportprotos "github.com/liuhengloveyou/passport/protos"
)

var engine *gin.Engine

func main() {
	if err := InitAdnRun(":8080"); err != nil {
		panic(err.Error())
	}
}

/*
curl -v -X PUT -H "X-API: register" -d \
'{
    "cellphone": "17688396387",
    "password": "123456"
}' "http://127.0.0.1:8080/user"
*/
func InitAdnRun(addr string) error {
	engine = gin.Default()

	// curl http://127.0.0.1:8080/ping
	engine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	options := &passportprotos.OptionStruct{
		LogDir:    "./logs", // 日志目录
		LogLevel:  "debug",  // 日志级别
		MysqlURN:  "root:lhisroot@tcp(127.0.0.1:3306)/kuge?charset=utf8mb4&parseTime=true&loc=Local",
	}
	engine.Any("/user", gin.WrapH(passport.InitAndRunHttpApi(options)))

	if err := engine.Run(addr); err != nil {
		return err
	}

	return nil
}

package main

import (
	"net/http"
	"time"

	passport "github.com/liuhengloveyou/passport/face"
	passportprotos "github.com/liuhengloveyou/passport/protos"
)

func main() {
	if err := InitHttpApi(":8080"); err != nil {
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
func InitHttpApi(addr string) error {
	options := &passportprotos.OptionStruct{
		LogDir:    "./logs", // 日志目录
		LogLevel:  "debug",  // 日志级别
		MysqlURN:  "root:lhisroot@tcp(127.0.0.1:3306)/kuge?charset=utf8mb4&parseTime=true&loc=Local",
	}
	http.Handle("/user", passport.InitAndRunHttpApi(options))
	// 业务可以挂在这里
	http.Handle("/", &HttpServer{})

	s := &http.Server{
		Addr:           addr,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

type HttpServer struct{}

// curl http://127.0.0.1:8080
func (p *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello passport"))
}

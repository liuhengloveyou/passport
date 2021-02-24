[toc]

## 用户中心(Passport)

用户基础信息维护、登录、权限验证、会话管理。



## 开始使用
### 作为一个单独的服务

可以启动为一个独立的用户中心服务，提供HTTP服务。接口说明见[下文](#接口)，服务配置示例如下：

#### 配置

```yaml
addr: ":9999" # 服务监听地址
log_dir: "./logs" # 日志打印路径
log_level: "debug" # 日志级别
redis: "127.0.0.1:19738" # redis缓存的地址
mysql: "root:lhisroot@tcp(127.0.0.1:3306)/xxx?charset=utf8&parseTime=true" # mysql数据库URN

# 会话配置
session:
  store_type: "cookie"
  expire: 86400

access_control: true #是否启用权限控制模块  
```


### 作为代码模块整合到HTTP路由

只需要像下面代码一样添加一个路由

```go
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

func InitHttpApi(addr string) error {
	options := &passportprotos.OptionStruct{
		LogDir:    "./logs", // 日志目录
		LogLevel:  "debug",  // 日志级别
		MysqlURN:  "root:lhisroot@tcp(127.0.0.1:3306)/xxx?charset=utf8mb4&parseTime=true&loc=Local",
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

func (p *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello passport"))
}
```


### 作为代码模块整合到gin路由

如果用gin框架，像下面一样添加路由
```go
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

func InitAdnRun(addr string) error {
	engine = gin.Default()

	engine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	options := &passportprotos.OptionStruct{
		LogDir:    "./logs", // 日志目录
		LogLevel:  "debug",  // 日志级别
		MysqlURN:  "root:lhisroot@tcp(127.0.0.1:3306)/xxx?charset=utf8mb4&parseTime=true&loc=Local",
	}
	engine.Any("/user", gin.WrapH(passport.InitAndRunHttpApi(options)))

	if err := engine.Run(addr); err != nil {
		return err
	}

	return nil
}
```



## 接口

GET请求使用标准的URL参数，POST用JSON格式的body。

### 注册/添加用户

| 参数字段  | 解释     | 是否必须 |
| --------- | -------- | -------- |
| password  | 密码     | 是       |
| cellphone | 手机号   | 否       |
| email     | 邮箱地址 | 否       |
| nickname  | 昵称     | 否       |

示例：

```
	curl -v -X PUT -H "X-API: register" -d \
	'{
	    "cellphone": "17688396387",
	    "password": "123456"
	}' "http://127.0.0.1:9999/user"
	
	成功: 
		statCode: 200
	  body: {
	    code: 0,
	    data: "uid"
		}
	失败: 
		statCode: 200
	  body: {code: -1, errmsg: "错误信息"}
```


### 登入

| 参数字段  | 解释     | 是否必须 |
| --------- | -------- | -------- |
| password  | 密码     | 是       |
| cellphone | 手机号   | 否       |
| email     | 邮箱地址 | 否       |
| nickname  | 昵称     | 否       |

```
curl -v -X POST -H "X-API: login" -d \
'{
	"cellphone": "17688396387",
  "password": "123456"
  }' "http://127.0.0.1:8080/user"
	  
	成功返回:
	{
		code: 0,
	  data: "uid"
	}
	
	失败返回:
	{
		code: -1,
	  errmsg:"错误信息"
	}
```

### 登出

```
GET /user
Header: {
    cookie: gsessionid=xxxxxx
}

成功: 200 {code: 0, data: "sucess"}
失败: 200 {code: -1, errmsg:"错误信息"}
```

### 签权

```
GET /user
Header: {
    cookie: gsessionid=xxxxxx
}

成功: 200 {code: 0, data: {
	"cellphone":"18510511015", 
	"email":"liuhengloveyou@gmail.com",
	"nickname":"恒"
	}}
失败: 200 {code: -1, errmsg:"错误信息"}
```

### 更新信息

```
POST /user
Header: {
    cookie: gsessionid=xxxxxx
}
Body: {
	"cellphone":"18510511015", 
	"email":"liuhengloveyou@gmail.com",
	"nickname":"恒恒",
	"password":"123456"
}
```

### 更新密码

| 参数字段 | 解释   | 是否必须 |
| -------- | ------ | -------- |
| n        | 新密码 | 是       |
| o        | 旧密码 | 是       |


示例：
```
POST /user
Header: {
    cookie: gsessionid=xxxxxx
}
Body: {
	"n":"18510511015", 
	"o":"liuhengloveyou@gmail.com",
}
```

### 更新头像

```
POST /user
Header: {
    cookie: gsessionid=xxxxxx
}
Body: {
}
```

### 查询自己的账号详情

```
curl -v -X GET -H "X-API: info" "http://127.0.0.1:8080/user"
```



## 访问控制(支持域/租户的RBAC)相关接口

可以用RBAC模型做功能和数据的访问权限控制。

### 为用户添加角色

```shell
curl -v -X POST -H "X-API: role/add" -d \
'{
	"uid": 123,
  "role": "role1"
}' "http://127.0.0.1:8080/user"
```

### 从用户删除角色

```shell
curl -v -X POST -H "X-API: role/del" -d \
'{
   "uid": 123,
   "role": "role1"
}' "http://127.0.0.1:8080/user"
```

### 为主体添加权限

```shell
curl -v -X POST -H "X-API: policy/add" -d \
'{
	"uid": 123,
  "sub": "data1",
  "act": "read"
}' "http://127.0.0.1:8080/user"
```

### 从主体删除权限

```shell
curl -v -X POST -H "X-API: policy/del" -d \
'{
	"uid": 123,
  "sub": "data1",
  "act": "read"
}' "http://127.0.0.1:8080/user"
```



## 应答格式说明

应答格式为JSON。正常情况：

```json
{
	code: 0,
  data:"xxx"
}
```

出错误情况：

```json
{
	code: -1,
  message:"错误信息"
}
```



## 错误信息说明

```json
{Code: 0, Message: "OK"}
{Code: -1000, Message: "请求参数错误"}
{Code: -1001, Message: "服务错误"}
{Code: -1002, Message: "请登录"}
{Code: -1003, Message: "您没有权限"}
```




## 数据库表结构

使用mysql数据库。可以创建单独的数据库， 也可以在业务库里添加users表， 表结构至少包含如下字段：

```sql
CREATE SCHEMA `passport` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin ;

CREATE TABLE `passport`.`users` (
  `uid` bigint(64) NOT NULL AUTO_INCREMENT,
  `cellphone` varchar(11) COLLATE utf8_bin DEFAULT NULL COMMENT '手机号',
  `email` varchar(255) COLLATE utf8_bin DEFAULT NULL COMMENT '邮件是址',
  `nickname` varchar(255) COLLATE utf8_bin DEFAULT NULL COMMENT '昵称',
  `password` varchar(255) COLLATE utf8_bin NOT NULL,
  `avatar_url` varchar(255) COLLATE utf8_bin DEFAULT NULL COMMENT '头像URL',
  `gender` int(11) DEFAULT NULL COMMENT '性别；1=男，2=女',
  `addr` varchar(100) COLLATE utf8_bin DEFAULT NULL COMMENT '通讯地址',
  `add_time` datetime NOT NULL,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`uid`),
  UNIQUE KEY `cellphone_UNIQUE` (`cellphone`),
  UNIQUE KEY `email_UNIQUE` (`email`),
  UNIQUE KEY `nickname_UNIQUE` (`nickname`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
```


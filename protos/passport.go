package protos

type OptionStruct struct {
	ServID    string `yaml:"serv_id"`
	PidFile   string `yaml:"pid_file"`
	Face      string `yaml:"face"`
	Addr      string `yaml:"addr"`      // 启动http的端口
	LogDir    string `yaml:"log_dir"`   // 日志目录
	LogLevel  string `yaml:"log_level"` // 日志级别
	RedisAddr string `yaml:"redis"`
	MysqlURN  string `yaml:"mysql"`
	AvatarDir string `yaml:"avatar_dir"`

	AdminTenantID uint64 `yaml:"admin_tenant_id"` // admin接口只有一个租户有权限

	SessionKey       string `yaml:"session_key"`
	SessionStoreType string `yaml:"session_store_type"` // 会话存储类型；"cookie/mem/reids"
	SessionExpire    int    `yaml:"session_expire"`

	SmsDriveer string                 `yaml:"sms"`
	SmsConf    map[string]interface{} `yaml:"sms_conf"`

	AppID     string `yaml:"wx_appid"`
	AppSecret string `yaml:"wx_secret"`

	ApiConf map[string]ApiConfStruct `yaml:"api_conf"`
}

type ApiConfStruct struct {
	NeedAccess bool `yaml:"need_access"`
}

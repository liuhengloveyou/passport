package protos

type OptionStruct struct {
	ServID    string `yaml:"serv_id"`
	PidFile   string `yaml:"pid_file"`
	Addr      string `yaml:"addr"`      // 启动http的端口
	LogDir    string `yaml:"log_dir"`   // 日志目录
	LogLevel  string `yaml:"log_level"` // 日志级别
	AvatarDir string `yaml:"avatar_dir"`

	RedisAddr string `yaml:"redis"`

	// 数据库配置（新）
	DBDriver string `yaml:"db_driver"` // "postgres" 或 "sqlite3"
	DBDSN    string `yaml:"db_dsn"`    // 数据库连接字符串

	RootUserID   uint64 `yaml:"root_user_id"`   // -init 时写入的超级管理员 uid，默认 10000
	RootTenantID uint64 `yaml:"root_tenant_id"` // admin 接口有权限的根租户；-init 默认 10000

	Domain           string `json:"domain"`
	SessionKey       string `yaml:"session_key"`
	SessionStoreType string `yaml:"session_store_type"` // 会话存储类型；"cookie/mem/reids"
	SessionExpire    int    `yaml:"session_expire"`

	SmsDriveer string                 `yaml:"sms"`
	SmsConf    map[string]interface{} `yaml:"sms_conf"`

	// 微信开放平台
	AppID     string `yaml:"wx_appid"`
	AppSecret string `yaml:"wx_secret"`

	// 微信小程序
	MiniAppID     string `yaml:"wx_mini_appid"`
	MiniAppSecret string `yaml:"wx_mini_secret"`

	// 支付宝网页授权（H5）
	AlipayAppID      string `yaml:"alipay_app_id"`
	AlipayPrivateKey string `yaml:"alipay_private_key"`
	AlipayPublicKey  string `yaml:"alipay_public_key"`
	AlipayEncryptKey string `yaml:"alipay_encrypt_key"`

	ApiConf map[string]ApiConfStruct `yaml:"api_conf"`
}

type ApiConfStruct struct {
	NeedAccess bool `yaml:"need_access"`
}

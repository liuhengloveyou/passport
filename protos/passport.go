package protos

type OptionStruct struct {
	ServID           string `yaml:"serv_id"`
	PidFile          string `yaml:"pid_file"`
	Face             string `yaml:"face"`
	Addr             string `yaml:"addr"`               // 启动http的端口
	LogDir           string `yaml:"log_dir"`            // 日志目录
	LogLevel         string `yaml:"log_level"`          // 日志级别
	SessionStoreType string `yaml:"session_store_type"` // 会话存储类型；"cookie/mem/reids"
	SessionExpire    int    `yaml:"session_expire"`

	RedisAddr      string `yaml:"redis"`
	MysqlURN       string `yaml:"mysql"`
	MysqlTableName string `yaml:"mysql_table_name"`
	AvatarDir      string `yaml:"avatar_dir"`

	AccessControl bool   `yaml:"access_control"`  // 是否启用权限控制模块
	IsTenant      bool   `yaml:"is_tenant"`       // 是否启用多租户
	AdminTenantID uint64 `yaml:"admin_tenant_id"` // admin接口只有一个租户有权限
}

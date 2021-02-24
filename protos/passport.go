package protos

type OptionStruct struct {
	ServID   string `yaml:"serv_id"`
	Face     string `yaml:"face"`
	Addr     string `yaml:"addr"` // 启动http的端口
	LogDir   string `yaml:"log_dir"` // 日志目录
	LogLevel string `yaml:"log_level"`// 日志级别

	RedisAddr string `yaml:"redis"`
	MysqlURN string `yaml:"mysql"`
	AvatarDir string `yaml:"avatar_dir"`

	AccessControl bool `yaml:"access_control"` // 是否启用权限控制模块
}

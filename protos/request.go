package protos

type UserReq struct {
	UID       uint64 `json:"uid" validate:"-"`
	TenantID  uint64 `json:"tenant_id" validate:"-"`
	Cellphone string `json:"cellphone" validate:"omitempty,phone,len=11"`
	Email     string `json:"email" validate:"omitempty,email,max=64"`
	Nickname  string `json:"nickname" validate:"omitempty,min=2,max=32"`
	Password  string `json:"password" validate:"omitempty,min=6,max=64"`
	AvatarURL string `json:"avatarUrl" validate:"omitempty,max=100"`
	Addr      string `json:"addr" validate:"omitempty,min=1,max=100"`
	Gender    int32  `json:"gender" validate:"omitempty,min=1,max=2"`
	SmsCode   string `json:"sms" validate:"omitempty,min=1,max=6"`

	// 微信
	WxOpenId string `json:"wxopenid,omitempty" validate:"-"`

	Roles   []string  `json:"roles" validate:"-"`
	DepIds  []uint64  `json:"depIds" validate:"-"`
	Ext     MapStruct `json:"ext" validate:"-"` // 记录用户的扩展信息
	Disable int8      `json:"disable" validate:"-"`

	PageNo   uint64
	PageSize uint64
}

type ModifyPwdReq struct {
	OldPwd string `json:"o" validate:"required,min=6,max=64"`
	NewPwd string `json:"n" validate:"required,min=6,max=64"`
}

type GetbackPwdReq struct {
	Cellphone string `json:"cellphone" validate:"required,phone,len=11"`
	SmsCode   string `json:"sms" validate:"required,len=6"`
	NewPwd    string `json:"n" validate:"required,min=6,max=64"`
}

type PolicyReq struct {
	Role string `json:"role" validate:"required,max=100"`
	Obj  string `json:"obj" validate:"required,min=1,max=100"`
	Act  string `json:"act" validate:"required,min=1,max=10"`
}

type RoleReq struct {
	RoleValue    string `json:"value" validate:"max=10"`
	NewRoleValue string `json:"newValue" validate:"max=10"`

	UID uint64 `json:"uid" validate:"-"`
}

type DisableUserReq struct {
	UID     uint64 `json:"uid" validate:"required,min=1"`
	Disable int8   `json:"disable" validate:"min=0,max=1"`
}

type SetDepartmentReq struct {
	UID    uint64   `json:"uid" validate:"required,min=1"`
	DepIds []uint64 `json:"depIds" validate:"-"`
}

type KvReq struct {
	ID       uint64      `json:"id" validate:"required,min=1"`
	TenantID uint64      `json:"tenant_id" validate:"-"`
	K        string      `json:"k" validate:"required,max=10"`
	V        interface{} `json:"v" validate:"-"`
}

type SmsReq struct {
	Cellphone string `json:"cellphone" validate:"phone,len=11"`
	AliveSec  int64  `json:"aliveSec" validate:"min=0,max=100"`
}

type NewTenantReq struct {
	TenantName string `json:"tenantName"  validate:"omitempty,min=2,max=64"`
	TenantType string `json:"tenantType"  validate:"omitempty,min=2,max=64"`
	Cellphone  string `json:"cellphone" validate:"required,phone,len=11"`
	Password   string `json:"password" validate:"required,min=6,max=64"`
}

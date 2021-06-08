package protos

type UserReq struct {
	UID       uint64 `json:"uid" validate:"-"`
	TenantID  uint64 `json:"tenant_id" validate:"-"`
	Cellphone string `json:"cellphone" validate:"omitempty,phone,len=11"`
	Email     string `json:"email" validate:"omitempty,email,max=64"`
	Nickname  string `json:"nickname" validate:"omitempty,min=2,max=32"`
	Password  string `json:"password" validate:"omitempty,min=6,max=64"`
	AvatarURL string `json:"avatarUrl" validate:"omitempty,url,max=100"`
	Addr      string `json:"addr" validate:"omitempty,min=1,max=100"`
	Gender    int32  `json:"gender" validate:"omitempty,min=1,max=2"`

	Roles   []string `json:"roles" validate:"omitempty"`
	Disable int8     `json:"disable" validate:"-"`
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

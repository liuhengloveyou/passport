package protos

import (
	"time"

	null "gopkg.in/guregu/null.v3/zero"
)

type UserReq struct {
	UID       uint64 `json:"uid,omitempty" validate:"-"`
	Cellphone string `json:"cellphone,omitempty" validate:"omitempty,phone"`
	Email     string `json:"email,omitempty" validate:"omitempty,email"`
	Nickname  string `json:"nickname,omitempty" validate:"omitempty,min=2,max=64"`
	Password  string `json:"password,omitempty" validate:"omitempty,min=6,max=256"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Addr      string `json:"addr,omitempty"`
	Gender    int32  `json:"gender,omitempty"`
}

type User struct {
	UID        uint64       `json:"uid,omitempty" validate:"-" db:"uid"`
	Password   string       `json:"password,omitempty" validate:"required,min=6,max=256" db:"password"`
	Cellphone  *null.String `json:"cellphone,omitempty" validate:"omitempty,phone" db:"cellphone"`
	Email      *null.String `json:"email,omitempty" validate:"omitempty,email" db:"email"`
	Nickname   *null.String `json:"nickname,omitempty" validate:"omitempty,min=2,max=64" db:"nickname"`
	AvatarURL  *null.String `json:"avatarUrl,omitempty" db:"avatar_url"`
	Gender     *null.Int    `json:"gender,omitempty" db:"gender"`
	Addr       *null.String `json:"addr,omitempty" db:"addr"`
	AddTime    *time.Time   `json:"addTime,omitempty" validate:"-" db:"add_time"`
	UpdateTime *time.Time   `json:"updateTime,omitempty" validate:"-" db:"update_time"`
}

type PolicyReq struct {
	UID uint64 `json:"uid,omitempty" validate:"-"`
	Sub string `json:"sub" validate:"required,min=2,max=32"`
	Act string `json:"act" validate:"required,min=2,max=32"`
}

type RoleReq struct {
	UID uint64 `json:"uid,omitempty" validate:"-"`
	Role string `json:"role" validate:"required,min=2,max=32"`
}

package protos

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	null "gopkg.in/guregu/null.v3/zero"
)

type User struct {
	UID        uint64        `json:"uid,omitempty" validate:"-" db:"uid"`
	TenantID   uint64        `json:"tenant_id,omitempty" validate:"-" db:"tenant_id"`
	Password   string        `json:"password,omitempty" validate:"required,min=6,max=256" db:"password"`
	Cellphone  *null.String  `json:"cellphone,omitempty" validate:"omitempty,phone" db:"cellphone"`
	Email      *null.String  `json:"email,omitempty" validate:"omitempty,email" db:"email"`
	Nickname   *null.String  `json:"nickname,omitempty" validate:"omitempty,min=2,max=64" db:"nickname"`
	AvatarURL  *null.String  `json:"avatarUrl" db:"avatar_url"`
	Addr       *null.String  `json:"addr" db:"addr"`
	Gender     *null.Int     `json:"gender" db:"gender"`
	AddTime    *time.Time    `json:"addTime,omitempty" validate:"-" db:"add_time"`
	UpdateTime *time.Time    `json:"updateTime,omitempty" validate:"-" db:"update_time"`
	DeleteTime *time.Time    `json:"deleteTime,omitempty" validate:"-" db:"delete_time"`
	LoginTime  *time.Time    `json:"loginTime,omitempty" validate:"-" db:"login_time"`
	Tenant     *Tenant       `json:"tenant,omitempty" validate:"-" db:"tenant"`

	/*
	{
		"disabled": [1 | 0]

		"TOKEN": "xxx"
	}
	*/
	Ext        MapStruct `json:"ext,omitempty" validate:"-" db:"ext"`
}

// 租户
type Tenant struct {
	ID            uint64               `json:"id" validate:"-" db:"id"`
	UID           uint64               `json:"uid,omitempty" validate:"-" db:"uid"`
	TenantName    string               `json:"tenant_name" db:"tenant_name" validate:"omitempty,min=2,max=64"`
	TenantType    string               `json:"tenant_type" db:"tenant_type" validate:"omitempty,min=2,max=10"`
	AddTime       *time.Time           `json:"addTime,omitempty" validate:"-" db:"add_time"`
	UpdateTime    *time.Time           `json:"updateTime,omitempty" validate:"-" db:"update_time"`
	Info          MapStruct            `json:"info,omitempty" db:"info"`
	Configuration *TenantConfiguration `json:"configuration,omitempty" db:"configuration"`
}

// 租户配置字段
type TenantConfiguration struct {
	Roles []RoleStruct `json:"roles"` // 用户角色字典列表
	More  MapStruct    `json:"more"`
}

func (t *TenantConfiguration) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	if len(src.([]byte)) <= 2 {
		return nil
	}

	b, _ := src.([]byte)
	return json.Unmarshal(b, t)
}
func (t TenantConfiguration) Value() (driver.Value, error) {
	return json.Marshal(t)
}

type RoleStruct struct {
	RoleTitle string `json:"title" validate:"max=64"`
	RoleValue string `json:"value" validate:"max=64"`

	UID uint64 `json:"uid,omitempty" validate:"-"`
}

type PageResponse struct {
	Total uint64      `json:"total,omitempty"`
	List  interface{} `json:"list"`
}

type MapStruct map[string]interface{}

func (t *MapStruct) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	if len(src.([]byte)) <= 2 {
		return nil
	}

	b, _ := src.([]byte)
	return json.Unmarshal(b, t)
}
func (t MapStruct) Value() (driver.Value, error) {
	return json.Marshal(t)
}

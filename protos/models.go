package protos

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	null "gopkg.in/guregu/null.v4/zero"
)

const (
	DepartmentExtKey = "deps"

	WX_MP_AUTH_UID = 100
	ALI_AUTH_UID   = 101
)

type PageResponse struct {
	Total uint64      `json:"total,omitempty"`
	List  interface{} `json:"list"`
}

type User struct {
	UID       uint64       `json:"uid,omitempty" validate:"-" db:"uid"` // 正常要从10000开始往上自增，10000以下保留内部使用
	TenantID  uint64       `json:"tenant_id,omitempty" validate:"-" db:"tenant_id"`
	Password  string       `json:"password,omitempty" validate:"required,min=6,max=256" db:"password"`
	Cellphone *null.String `json:"cellphone,omitempty" validate:"omitempty,len=11" db:"cellphone"`
	Email     *null.String `json:"email,omitempty" validate:"omitempty,email" db:"email"`
	Nickname  *null.String `json:"nickname,omitempty" validate:"omitempty,min=2,max=64" db:"nickname"`
	AvatarURL *null.String `json:"avatarUrl,omitempty" db:"avatar_url"`
	Addr      *null.String `json:"addr,omitempty" db:"addr"`
	Gender    *null.Int    `json:"gender,omitempty" db:"gender"`

	Province *null.String `json:"province,omitempty" db:"province"`
	City     *null.String `json:"city,omitempty" db:"city"`

	CreateTime *time.Time `json:"createTime,omitempty" validate:"-" db:"create_time"`
	UpdateTime *time.Time `json:"updateTime,omitempty" validate:"-" db:"update_time"`
	DeleteTime *time.Time `json:"deleteTime,omitempty" validate:"-" db:"delete_time"`
	LoginTime  *time.Time `json:"loginTime,omitempty" validate:"-" db:"login_time"`

	Tenant      *Tenant      `json:"tenant,omitempty" validate:"-" db:"tenant"`
	Roles       []RoleStruct `json:"roles,omitempty" validate:"-"`
	Departments []Department `json:"departments,omitempty" validate:"-"`

	WxOpenId *null.String `json:"wxopenid,omitempty" validate:"omitempty,min=2,max=64" db:"wx_openid"` // 微信

	/*
		{
			"disabled": [1 | 0]
			"deps": [1,2,3]
			"TOKEN": "xxx"
		}
	*/
	Ext MapStruct `json:"ext,omitempty" validate:"-" db:"ext"` // 记录用户的扩展信息

	CacheTime int64 `json:"-" validate:"-" db:"-"` // 缓存到内存的时间
}

func (u *User) SetExt(k string, v interface{}) {
	if u.Ext == nil {
		u.Ext = make(map[string]interface{})
	}

	u.Ext[k] = v
}

type UserLite struct {
	UID       uint64       `json:"uid,omitempty" validate:"-" db:"uid"` // 正常要从10000开始往上自增，100以下保留内部使用
	TenantID  uint64       `json:"tenant_id,omitempty" validate:"-" db:"tenant_id"`
	Nickname  *null.String `json:"nickname,omitempty" validate:"omitempty,min=2,max=64" db:"nickname"`
	Cellphone *null.String `json:"cellphone,omitempty" validate:"omitempty,len=11" db:"cellphone"`
	AvatarURL *null.String `json:"avatarUrl" db:"avatar_url"`
	WxOpenId  *null.String `json:"wxopenid,omitempty" validate:"omitempty,min=2,max=64" db:"wx_openid"` // 微信
	Ext       MapStruct    `json:"ext,omitempty" validate:"-" db:"ext"`                                 // 记录用户的扩展信息
}

type UserLiteArr []UserLite

func (t *UserLiteArr) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	if len(src.([]byte)) <= 2 {
		return nil
	}

	b, _ := src.([]byte)
	return json.Unmarshal(b, t)
}
func (t UserLiteArr) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// 租户
type Tenant struct {
	ID            uint64               `json:"id" validate:"-" db:"id"`
	UID           uint64               `json:"uid,omitempty" validate:"-" db:"uid"`
	TenantName    string               `json:"tenantName" db:"tenant_name" validate:"omitempty,min=2,max=64"`
	TenantType    string               `json:"tenantType" db:"tenant_type" validate:"omitempty,min=1,max=64"`
	CreateTime    *time.Time           `json:"createTime,omitempty" validate:"-" db:"create_time"`
	UpdateTime    *time.Time           `json:"updateTime,omitempty" validate:"-" db:"update_time"`
	Info          *TenantInfo          `json:"info,omitempty" db:"info"`
	Configuration *TenantConfiguration `json:"configuration,omitempty" db:"configuration"`

	Depth int `json:"depth,omitempty" validate:"-" db:"-"`
}

// 租户配置字段
type TenantConfiguration struct {
	Roles []RoleStruct `json:"roles"` // 用户角色字典列表
	More  MapStruct    `json:"more"`
}

func (t TenantConfiguration) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TenantConfiguration) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		if len(v) <= 2 {
			return nil
		}
		b = v
	case string:
		if len(v) <= 2 {
			return nil
		}
		b = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into TenantConfiguration", src)
	}

	return json.Unmarshal(b, t)
}

// t_tenant_closure 租户闭包表
type TenantClosure struct {
	ID           uint64 `json:"id" db:"id"`
	AncestorID   uint64 `json:"ancestorId" db:"ancestor_id"`     // 主先
	DescendantID uint64 `json:"descendantId" db:"descendant_id"` // 后代
	Distance     int    `json:"distance" db:"distance"`          // 级数
}

// 租户配置字段
type TenantInfo struct {
	AdminCellphone string `json:"adminCellphone"` // 管理员手机号列表
}

func (t TenantInfo) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TenantInfo) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v) // 将 string 转换为 []byte
	default:
		return fmt.Errorf("cannot scan %T into TenantInfo", value)
	}

	return json.Unmarshal(bytes, t)
}

// 角色
type RoleStruct struct {
	RoleTitle string `json:"title" validate:"max=64"`
	RoleValue string `json:"value" validate:"required,max=64"`

	UID uint64 `json:"uid,omitempty" validate:"-"`
}

// 权限条目
type PermissionStruct struct {
	ID         uint64     `json:"id,omitempty" validate:"-" db:"id"`
	TenantID   uint64     `json:"tenantId,omitempty" validate:"-" db:"tenant_id"`
	Domain     string     `json:"domain,omitempty" validate:"required,min=2,max=128" db:"domain"`
	Title      string     `json:"title" validate:"required,min=2,max=128" db:"title"`
	Value      string     `json:"value" validate:"required,min=2,max=256" db:"value"`
	CreateTime *time.Time `json:"createTime,omitempty" validate:"-" db:"create_time"`
	UpdateTime *time.Time `json:"updateTime,omitempty" validate:"-" db:"update_time"`
}

// 权限策略
type Policy struct {
	Role string `json:"role"`
	Obj  string `json:"obj"`
	Act  string `json:"act"`
}

// 部门
type Department struct {
	Id         uint64     `json:"id" validate:"omitempty,min=1" db:"id" gorm:"column:id;type:INT;primaryKey;autoIncrement"`
	ParentID   uint64     `json:"parentId" validate:"-" db:"parent_id"`
	UserId     uint64     `json:"uid" validate:"omitempty,min=1" db:"uid" gorm:"column:uid;type:INT;not null;"`
	TenantID   uint64     `json:"tenantId,omitempty" validate:"-" db:"tenant_id"`
	CreateTime *time.Time `json:"createTime,omitempty" validate:"-" db:"create_time"` // 创建时间
	UpdateTime *time.Time `json:"updateTime" validate:"-" db:"update_time"`           // 最后更新时间
	Name       string     `json:"name" validate:"required,max=10" db:"name"`
	Config     MapStruct  `json:"config,omitempty" validate:"-" db:"config"` // 记录用户的扩展信息
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

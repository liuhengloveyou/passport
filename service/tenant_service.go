package service

import (
	"github.com/liuhengloveyou/go-errors"
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func TenantAdd(m *protos.Tenant) (tenantID int64, e error) {
	if m.TenantName == "" {
		return -1, common.ErrTenantNameNull
	}
	if m.TenantType == "" {
		return -1, common.ErrTenantTypeNull
	}

	if m.Configuration == nil {
		m.Configuration = &protos.TenantConfiguration{}
	}
	if len(m.Configuration.More) > 100 {
		return -1, common.ErrService
	}

	// 默认添加超级管理员角色
	m.Configuration.Roles = []protos.RoleStruct{{
		RoleTitle: "超级管理员",
		RoleValue: "root",
	}}

	tx := common.DB.MustBegin()
	defer func() {
		if e != nil {
			tx.Rollback()
		}
	}()

	tenantID, e = dao.TenantInsert(tx, m)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantAdd TenantInsert ERR: %v %v %v\n", m.UID, tenantID, e)
		if _, ok := e.(errors.Error); !ok {
			e = common.ErrService
		}
		return
	}
	if tenantID <= 0 {
		common.Logger.Sugar().Errorf("TenantAdd AddRoleForUserInDomain ERR: %v %v %v\n", m.UID, tenantID, e)
		return -1, common.ErrTenantAddERR
	}

	// 当前用户设置为超级管理员角色
	if e = accessctl.AddRoleForUserInDomain(m.UID, uint64(tenantID), "root"); e != nil {
		common.Logger.Sugar().Errorf("TenantAdd AddRoleForUserInDomain ERR: %v %v %v\n", m.UID, tenantID, e)
		return -1, common.ErrService
	}

	e = tx.Commit()

	return
}

func TenantUserDel(uid, currTenantID uint64) (r int64, e error) {
	if r, e = dao.UserUpdateTenantID(uid, 0, currTenantID); e != nil {
		common.Logger.Sugar().Errorf("TenantUserDel ERR: ", e)
		return 0, common.ErrService
	}

	return
}

func TenantAddRole(tenantId uint64, role protos.RoleStruct) error {
	tenant, err := dao.TenantGetByID(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantAddRole db ERR: ", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)
	if len(tenant.Configuration.Roles) > 100 {
		return common.ErrService
	}
	for i := 0; i < len(tenant.Configuration.Roles); i++ {
		if tenant.Configuration.Roles[i].RoleTitle == role.RoleTitle || tenant.Configuration.Roles[i].RoleValue == role.RoleValue {
			return common.ErrMysql1062
		}
	}

	tenant.Configuration.Roles = append(tenant.Configuration.Roles, role)

	return dao.TenantUpdateConfiguration(tenant)
}

func TenantGetRole(tenantId uint64) (roles []protos.RoleStruct) {
	tenant, err := dao.TenantGetByID(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantAddRole db ERR: ", err)
		return
	}
	if nil == tenant {
		return
	}

	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)

	return tenant.Configuration.Roles
}

func TenantUpdateConfiguration(tenantId uint64, k string, v interface{}) error {
	tenant, err := dao.TenantGetByID(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantUpdateConfiguration db ERR: ", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}
	if tenant.Configuration.More == nil {
		tenant.Configuration.More = make(protos.MapStruct, 1)
	}

	if v != nil {
		if len(tenant.Configuration.More) > 100 {
			return common.ErrService
		}
		tenant.Configuration.More[k] = v
	} else {
		delete(tenant.Configuration.More, k)
	}

	return dao.TenantUpdateConfiguration(tenant)
}

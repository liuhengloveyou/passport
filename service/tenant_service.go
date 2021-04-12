package service

import (
	"github.com/liuhengloveyou/go-errors"
	"github.com/liuhengloveyou/passport/accessctl"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func TenantAdd(m *protos.Tenant) (id int64, e error) {
	if m.TenantName == "" {
		return -1, common.ErrTenantNameNull
	}
	if m.TenantType == "" {
		return -1, common.ErrTenantTypeNull
	}

	// 默认添加超级管理员角色
	m.Configuration.Roles = []string{"root"}

	tx := common.DB.MustBegin()
	defer func() {
		if e != nil {
			tx.Rollback()
		}
	}()

	id, e = dao.TenantInsert(tx, m)
	if e != nil {
		common.Logger.Sugar().Errorf("TenantAdd TenantInsert ERR: %v %v %v\n", m.UID, id, e)
		if _, ok := e.(errors.Error); !ok {
			e = common.ErrService
		}
		return
	}

	// 当前用户设置为超级管理员角色
	if e = accessctl.AddRoleForUserInDomain(m.UID, uint64(id), "root"); e != nil {
		common.Logger.Sugar().Errorf("TenantAdd AddRoleForUserInDomain ERR: %v %v %v\n", m.UID, id, e)
		return -1, common.ErrService
	}

	e = tx.Commit()

	return
}


func TenantAddRole(tenantId uint64, role string) error {
	tenant, err := dao.TenantGetByID(tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("TenantAddRole db ERR: ", err)
		return common.ErrService
	}
	if nil == tenant {
		return common.ErrTenantNotFound
	}

	common.Logger.Sugar().Debugf("tenant: %v\n", tenant)

	for i := 0; i < len(tenant.Configuration.Roles); i++ {
		if tenant.Configuration.Roles[i] == role {
			return common.ErrMysql1062
		}
	}

	tenant.Configuration.Roles = append(tenant.Configuration.Roles, role)

	return dao.TenantUpdateConfiguration(tenant)
}


func TenantGetRole(tenantId uint64) (roles []string) {
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
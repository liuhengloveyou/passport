package service

import (
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

	id, e = dao.TenantInsert(m)

	return
}

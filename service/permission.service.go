package service

import (
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)


func PermissionCreate(model *protos.PermissionStruct) (int64, error) {
	return dao.PermissionCreate(common.DB, model)
}

func PermissionDelete(id, tenantID uint64) error {
	if id <= 0 {
		logger.Errorf("PermissionDelete id ERR: %v\n", id)
		return common.ErrParam
	}
	row, err := dao.PermissionDelete(common.DB, id, tenantID)
	if err != nil {
		logger.Errorf("dao.PermissionDelete ERR: %v\n", err)
		return common.ErrService
	}
	if row != 1 {
		logger.Warnf("PermissionDelete row: %v\n", row)
	}

	return nil
}

func PermissionList(tenantID uint64, domain string) (rr []protos.PermissionStruct, err error) {
	if rr, err = dao.PermissionList(common.DB, tenantID, domain); err != nil {
		logger.Errorf("dao.PermissionList ERR: %v\n", err)
		return nil, common.ErrService
	}

	return
}
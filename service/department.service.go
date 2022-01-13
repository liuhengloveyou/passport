package service

import (
	"github.com/go-sql-driver/mysql"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func DepartmentCreate(m *protos.Department) (lastInsertId int64, e error) {
	lastInsertId, err := dao.DepartmentCreate(common.DB, m)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentCreate ERR: %v %v\n", lastInsertId, err)
		if merr, ok := err.(*mysql.MySQLError); ok {
			if merr.Number == 1062 {
				return -1, common.ErrMysql1062
			}
		}

		return 0, common.ErrService
	}

	return lastInsertId, nil
}

func DepartmentDelete(id, tenantID uint64) (e error) {
	rr, err := dao.DepartmentDelete(common.DB, id, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentDelete ERR: %v\n", err)
		e = common.ErrService
		return
	}
	common.Logger.Sugar().Infof("DepartmentDelete: %v", rr)

	return
}

func DepartmentFind(id, tenantId uint64) (rr []protos.Department, e error) {
	rr, err := dao.DepartmentFind(common.DB, id, tenantId)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentFind ERR: %v\n", err)
		e = common.ErrService
		return
	}

	common.Logger.Sugar().Infof("DepartmentFind: %v", rr)
	return
}

func DepartmentUpdate(model *protos.Department) (e error) {
	if model.Id <= 0 || model.TenantID <= 0 || model.UserId <= 0 {
		common.Logger.Sugar().Errorf("DepartmentUpdate id nil: %#v", model)
		return common.ErrParam
	}
	model.UserId = 0
	model.ParentID = 0

	row, err := dao.DepartmentUpdate(common.DB, model)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentUpdate ERR: %v\n", err)
		e = common.ErrService
		return
	}

	if row != 1 {
		common.Logger.Sugar().Warn("DepartmentUpdate row: ", row, model.Id, model.TenantID, model)
	}

	return
}

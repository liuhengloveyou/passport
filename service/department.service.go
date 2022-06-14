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

func DepartmentUpdateConfig(id, currUid, currTenantID uint64, k string, v interface{}) error {
	if currUid <= 0 || id <= 0 || currTenantID <= 0 {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig ERR: %d %d %d %v %v", id, currUid, currTenantID, k, v)
		return common.ErrParam
	}
	if k == "" {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig ERR: %d %d %d %v %v", id, currUid, currTenantID, k, v)
		return common.ErrParam
	}

	rr, e := dao.DepartmentFind(common.DB, id, currTenantID)
	if e != nil {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig db ERR: %v", e)
		return common.ErrService
	}
	if len(rr) != 1 {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig db ERR: %v", e)
		return common.ErrNull
	}

	if rr[0].TenantID != currTenantID || rr[0].Id != id {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig tenant ERR: %d %d %d %v", id, currUid, currTenantID, rr[0])
		return common.ErrNoAuth
	}

	common.Logger.Sugar().Infof("DepartmentUpdateConfig: %d %d %d %v %v %v", id, currUid, currTenantID, k, v, rr[0])
	if v == nil {
		common.Logger.Sugar().Warnf("DepartmentUpdateConfig delete: %v", k)
		delete(rr[0].Config, k)
	} else {
		rr[0].Config[k] = v
	}

	rows, e := dao.DepartmentUpdateConfig(common.DB, &rr[0])
	if e != nil {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig ERR: %v", e)
		return common.ErrService
	}
	if rows < 1 {
		common.Logger.Sugar().Warnf("DepartmentUpdateConfig RowsAffected 0")
	}

	return nil
}

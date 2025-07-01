package service

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
	"github.com/liuhengloveyou/passport/protos"
)

func DepartmentCreate(m *protos.Department) (lastInsertId int64, e error) {
	lastInsertId, err := dao.DepartmentCreate(m)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentCreate ERR: %v %v\n", lastInsertId, err)
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" { // 唯一约束冲突
			return -1, common.ErrPgDupKey
		}

		return 0, common.ErrService
	}

	return lastInsertId, nil
}

func DepartmentDelete(id, tenantID uint64) (e error) {
	defer func() {
		deparmentCache.Delete(fmt.Sprintf("%d/%d", tenantID, id))
	}()

	rr, err := dao.DepartmentDelete(id, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentDelete ERR: %v\n", err)
		e = common.ErrService
		return
	}
	common.Logger.Sugar().Infof("DepartmentDelete: %v", rr)

	return
}

func DepartmentFind(id, tenantId, page, pageSize uint64) (rr []protos.Department, e error) {
	if tenantId <= 0 {
		return nil, common.ErrParam
	}

	rr, err := dao.DepartmentFind(id, tenantId, page, pageSize)
	if err != nil {
		common.Logger.Sugar().Errorf("DepartmentFind ERR: %v\n", err)
		e = common.ErrService
		return
	}

	common.Logger.Sugar().Infof("DepartmentFind: %v", rr)

	if id > 0 && len(rr) == 1 {
		deparmentCache.Set(fmt.Sprintf("%d/%d", tenantId, id), rr[0], 600)
	}

	return
}

func DepartmentUpdate(model *protos.Department) (e error) {
	if model.Id <= 0 || model.TenantID <= 0 || model.UserId <= 0 {
		common.Logger.Sugar().Errorf("DepartmentUpdate id nil: %#v", model)
		return common.ErrParam
	}
	model.UserId = 0
	model.ParentID = 0

	defer func() {
		deparmentCache.Delete(fmt.Sprintf("%d/%d", model.TenantID, model.Id))
	}()

	row, err := dao.DepartmentUpdate(model)
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

	defer func() {
		deparmentCache.Delete(fmt.Sprintf("%d/%d", currTenantID, currUid))
	}()

	rr, e := dao.DepartmentFind(id, currTenantID, 0, 0)
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

	rows, e := dao.DepartmentUpdateConfig(&rr[0])
	if e != nil {
		common.Logger.Sugar().Errorf("DepartmentUpdateConfig ERR: %v", e)
		return common.ErrService
	}
	if rows < 1 {
		common.Logger.Sugar().Warnf("DepartmentUpdateConfig RowsAffected 0")
	}

	return nil
}

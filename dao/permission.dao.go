package dao

import (
	"context"
	"time"

	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/protos"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
)

func PermissionCreate(m *protos.PermissionStruct) (id int64, e error) {
	table := "permission"

	// 构建插入数据
	data := map[string]interface{}{
		"tenant_id":   m.TenantID,
		"domain":      m.Domain,
		"title":       m.Title,
		"value":       m.Value,
		"create_time": time.Now(),
	}

	// 使用squirrel构建SQL
	sql, args, err := sq.Insert(table).SetMap(data).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return -1, err
	}

	// 执行插入
	rst, err := common.DBPool.Exec(context.Background(), sql, args...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		// 处理重复记录错误
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" { // 唯一约束冲突
			return -1, common.ErrPgDupKey
		}
		return -1, err
	}

	id = rst.RowsAffected()
	return id, nil
}

func PermissionDelete(id, tenantID uint64) (int64, error) {
	table := "permission"

	// 构建 WHERE 条件
	where := sq.Eq{
		"id":        id,
		"tenant_id": tenantID,
	}

	// 使用squirrel构建SQL
	sql, args, err := sq.Delete(table).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return 0, err
	}

	// 执行删除
	rst, err := common.DBPool.Exec(context.Background(), sql, args...)
	common.Logger.Sugar().Debugf("db.exec: %v %v\n", rst, err)
	if err != nil {
		return 0, err
	}

	return rst.RowsAffected(), nil
}

func PermissionList(tenantID uint64, domain string) (rr []protos.PermissionStruct, err error) {
	table := "permission"

	// 构建 WHERE 条件
	where := sq.Eq{
		"tenant_id": tenantID,
		"domain":    domain,
	}

	// 使用squirrel构建SQL
	sql, args, err := sq.Select("*").From(table).Where(where).PlaceholderFormat(sq.Dollar).ToSql()
	common.Logger.Sugar().Debugf("%v %v %v\n", sql, args, err)
	if err != nil {
		return nil, err
	}

	// 执行查询
	rr = []protos.PermissionStruct{}
	rows, err := common.DBPool.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("db.Query error: %v\n", err)
		return nil, err
	}
	defer rows.Close()

	// 手动扫描结果到结构体切片
	for rows.Next() {
		var p protos.PermissionStruct
		err = rows.Scan(
			&p.ID,
			&p.TenantID,
			&p.Domain,
			&p.Title,
			&p.Value,
			&p.CreateTime,
			&p.UpdateTime,
		)
		if err != nil {
			common.Logger.Sugar().Errorf("rows.Scan error: %v\n", err)
			return nil, err
		}
		rr = append(rr, p)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		common.Logger.Sugar().Errorf("rows iteration error: %v\n", err)
		return nil, err
	}

	return rr, nil
}

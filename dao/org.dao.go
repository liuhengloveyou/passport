package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/database"
	"github.com/liuhengloveyou/passport/v4/protos"

	sq "github.com/Masterminds/squirrel"
)

func OrgInsert(m *protos.Organization) (id uint64, err error) {
	if m == nil || m.TenantID == 0 || strings.TrimSpace(m.Name) == "" {
		return 0, common.ErrParam
	}
	now := time.Now()
	err = common.DB.QueryRow(context.Background(),
		`INSERT INTO organizations (tenant_id, name, create_time, update_time) VALUES ($1, $2, $3, $4) RETURNING id`,
		m.TenantID, strings.TrimSpace(m.Name), now, now,
	).Scan(&id)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgInsert ERR: %v", err)
		return 0, err
	}
	return id, nil
}

func OrgGetByTenantName(tenantID uint64, name string) (*protos.Organization, error) {
	name = strings.TrimSpace(name)
	if tenantID == 0 || name == "" {
		return nil, nil
	}
	row := common.DB.QueryRow(context.Background(),
		`SELECT id, tenant_id, name, create_time, update_time FROM organizations WHERE tenant_id = $1 AND name = $2 LIMIT 1`,
		tenantID, name)
	var org protos.Organization
	if err := row.Scan(&org.ID, &org.TenantID, &org.Name, &org.CreateTime, &org.UpdateTime); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		common.Logger.Sugar().Errorf("OrgGetByTenantName ERR: %v", err)
		return nil, err
	}
	return &org, nil
}

func OrgGetByID(id uint64) (*protos.Organization, error) {
	if id == 0 {
		return nil, nil
	}
	row := common.DB.QueryRow(context.Background(),
		`SELECT id, tenant_id, name, create_time, update_time FROM organizations WHERE id = $1`, id)
	var org protos.Organization
	if err := row.Scan(&org.ID, &org.TenantID, &org.Name, &org.CreateTime, &org.UpdateTime); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		common.Logger.Sugar().Errorf("OrgGetByID ERR: %v", err)
		return nil, err
	}
	return &org, nil
}

func OrgListByTenant(tenantID uint64) ([]protos.Organization, error) {
	if tenantID == 0 {
		return nil, common.ErrParam
	}
	rows, err := common.DB.Query(context.Background(),
		`SELECT id, tenant_id, name, create_time, update_time FROM organizations WHERE tenant_id = $1 ORDER BY id`, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgListByTenant ERR: %v", err)
		return nil, err
	}
	defer rows.Close()
	out := make([]protos.Organization, 0)
	for rows.Next() {
		var org protos.Organization
		if err = rows.Scan(&org.ID, &org.TenantID, &org.Name, &org.CreateTime, &org.UpdateTime); err != nil {
			return nil, err
		}
		out = append(out, org)
	}
	return out, rows.Err()
}

func OrgListByUser(uid, tenantID uint64) ([]protos.Organization, error) {
	if uid == 0 || tenantID == 0 {
		return nil, common.ErrParam
	}
	rows, err := common.DB.Query(context.Background(),
		`SELECT o.id, o.tenant_id, o.name, o.create_time, o.update_time
		 FROM organizations o
		 INNER JOIN org_members m ON m.org_id = o.id
		 WHERE m.uid = $1 AND o.tenant_id = $2
		 ORDER BY o.id`, uid, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgListByUser ERR: %v", err)
		return nil, err
	}
	defer rows.Close()
	out := make([]protos.Organization, 0)
	for rows.Next() {
		var org protos.Organization
		if err = rows.Scan(&org.ID, &org.TenantID, &org.Name, &org.CreateTime, &org.UpdateTime); err != nil {
			return nil, err
		}
		out = append(out, org)
	}
	return out, rows.Err()
}

func OrgMemberInsert(orgID, uid, tenantID uint64) error {
	if orgID == 0 || uid == 0 || tenantID == 0 {
		return common.ErrParam
	}
	_, err := common.DB.Exec(context.Background(),
		`INSERT INTO org_members (org_id, uid, tenant_id, create_time) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (org_id, uid) DO NOTHING`,
		orgID, uid, tenantID, time.Now())
	if err != nil {
		common.Logger.Sugar().Errorf("OrgMemberInsert ERR: %v", err)
		return err
	}
	return nil
}

func OrgMemberDelete(orgID, uid uint64) error {
	if orgID == 0 || uid == 0 {
		return common.ErrParam
	}
	_, err := common.DB.Exec(context.Background(),
		`DELETE FROM org_members WHERE org_id = $1 AND uid = $2`, orgID, uid)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgMemberDelete ERR: %v", err)
		return err
	}
	return nil
}

func OrgMemberDeleteByOrg(orgID uint64) error {
	if orgID == 0 {
		return common.ErrParam
	}
	_, err := common.DB.Exec(context.Background(), `DELETE FROM org_members WHERE org_id = $1`, orgID)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgMemberDeleteByOrg ERR: %v", err)
		return err
	}
	return nil
}

func OrgDelete(id, tenantID uint64) error {
	if id == 0 || tenantID == 0 {
		return common.ErrParam
	}
	_, err := common.DB.Exec(context.Background(),
		`DELETE FROM organizations WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		common.Logger.Sugar().Errorf("OrgDelete ERR: %v", err)
		return err
	}
	return nil
}

func OrgMemberExists(orgID, uid uint64) (bool, error) {
	if orgID == 0 || uid == 0 {
		return false, nil
	}
	var n int
	err := common.DB.QueryRow(context.Background(),
		`SELECT 1 FROM org_members WHERE org_id = $1 AND uid = $2 LIMIT 1`, orgID, uid).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func OrgMemberCountByUser(uid, tenantID uint64) (int64, error) {
	var n int64
	err := common.DB.QueryRow(context.Background(),
		`SELECT COUNT(1) FROM org_members WHERE uid = $1 AND tenant_id = $2`, uid, tenantID).Scan(&n)
	return n, err
}

func OrgMemberUIDs(orgID uint64) ([]uint64, error) {
	if orgID == 0 {
		return nil, common.ErrParam
	}
	rows, err := common.DB.Query(context.Background(),
		`SELECT uid FROM org_members WHERE org_id = $1`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]uint64, 0)
	for rows.Next() {
		var uid uint64
		if err = rows.Scan(&uid); err != nil {
			return nil, err
		}
		if uid > 0 {
			out = append(out, uid)
		}
	}
	return out, rows.Err()
}

func UserQueryByOrg(tenantID, orgID, page, pageSize uint64, nickname string, uids []uint64) ([]protos.User, error) {
	if tenantID == 0 || orgID == 0 {
		return nil, common.ErrParam
	}
	act := sq.Select("u.uid", "u.tenant_id", "u.cellphone", "u.email", "u.nickname", "u.avatar_url", "u.gender", "u.addr", "u.ext", "u.create_time").
		From("users u").
		Join("org_members m ON m.uid = u.uid AND m.org_id = ?", orgID).
		PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).
		Where(sq.Eq{"u.tenant_id": tenantID})
	if nickname != "" {
		act = act.Where(sq.Like{"u.nickname": "%" + nickname + "%"})
	} else if len(uids) > 0 {
		ors := make(sq.Or, len(uids))
		for i := 0; i < len(uids); i++ {
			ors[i] = sq.Eq{"u.uid": uids[i]}
		}
		act = act.Where(ors)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 100
	}
	sql, args, err := act.Offset((page - 1) * pageSize).Limit(pageSize).ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := common.DB.Query(context.Background(), sql, args...)
	if err != nil {
		common.Logger.Sugar().Errorf("UserQueryByOrg ERR: %v", err)
		return nil, err
	}
	defer rows.Close()
	rr := []protos.User{}
	for rows.Next() {
		var user protos.User
		if err := rows.Scan(&user.UID, &user.TenantID, &user.Cellphone, &user.Email, &user.Nickname, &user.AvatarURL, &user.Gender, &user.Addr, &user.Ext, &user.CreateTime); err != nil {
			return nil, err
		}
		rr = append(rr, user)
	}
	return rr, rows.Err()
}

func UserCountByOrg(tenantID, orgID uint64, nickname string, uids []uint64) (uint64, error) {
	if tenantID == 0 || orgID == 0 {
		return 0, common.ErrParam
	}
	act := sq.Select("count(u.uid) as count").
		From("users u").
		Join("org_members m ON m.uid = u.uid AND m.org_id = ?", orgID).
		PlaceholderFormat(database.GetPlaceholderFormat(common.DB.DriverType())).
		Where(sq.Eq{"u.tenant_id": tenantID})
	if nickname != "" {
		act = act.Where(sq.Like{"u.nickname": "%" + nickname + "%"})
	} else if len(uids) > 0 {
		ors := make(sq.Or, len(uids))
		for i := 0; i < len(uids); i++ {
			ors[i] = sq.Eq{"u.uid": uids[i]}
		}
		act = act.Where(ors)
	}
	sql, args, err := act.ToSql()
	if err != nil {
		return 0, err
	}
	var count int64
	if err = common.DB.QueryRow(context.Background(), sql, args...).Scan(&count); err != nil {
		return 0, err
	}
	if count < 0 {
		return 0, nil
	}
	return uint64(count), nil
}

func DepartmentAssignOrphan(tenantID, orgID uint64) error {
	if tenantID == 0 || orgID == 0 {
		return common.ErrParam
	}
	_, err := common.DB.Exec(context.Background(),
		`UPDATE departments SET org_id = $1 WHERE tenant_id = $2 AND org_id = 0`, orgID, tenantID)
	return err
}

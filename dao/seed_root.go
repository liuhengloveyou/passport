package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
)

// 默认 root 账号与 README 注册/登录示例对齐（cellphone + password）。
const (
	DefaultRootUID        uint64 = 10000
	DefaultRootTenantID   uint64 = 10000
	DefaultRootNickname          = "admin"
	DefaultRootPassword          = "123456"
	DefaultRootCellphone         = "15360651247" // VARCHAR(11)
	DefaultRootEmail             = "35221199@qq.com"
	DefaultRootTenantName        = "系统超级管理员"
)

// SeedRootOptions 初始化超级管理员与根租户的可选项；零值字段使用默认值。
type SeedRootOptions struct {
	UID        uint64
	TenantID   uint64
	Nickname   string
	Password   string // 明文；入库前会 EncryPWD
	Cellphone  string
	Email      string
	TenantName string
}

func (o *SeedRootOptions) normalize() SeedRootOptions {
	out := SeedRootOptions{
		UID:        DefaultRootUID,
		TenantID:   DefaultRootTenantID,
		Nickname:   DefaultRootNickname,
		Password:   DefaultRootPassword,
		Cellphone:  DefaultRootCellphone,
		Email:      DefaultRootEmail,
		TenantName: DefaultRootTenantName,
	}
	if o == nil {
		return out
	}
	if o.UID > 0 {
		out.UID = o.UID
	}
	if o.TenantID > 0 {
		out.TenantID = o.TenantID
	}
	if o.Nickname != "" {
		out.Nickname = o.Nickname
	}
	if o.Password != "" {
		out.Password = o.Password
	}
	if o.Cellphone != "" {
		out.Cellphone = o.Cellphone
	}
	if o.Email != "" {
		out.Email = o.Email
	}
	if o.TenantName != "" {
		out.TenantName = o.TenantName
	}
	return out
}

// SeedRoot 按 README 表结构写入超级管理员与根租户（幂等 upsert），并维护 closure 自引用。
//
// users / tenants / user_closure / tenant_closure 字段与 README「PostgreSQL 建表」一致：
//   - users: uid, tenant_id, nickname, cellphone, email, password, ext, create_time, update_time
//   - tenants: id, uid, tenant_name, tenant_type, info, configuration, create_time, update_time
//   - *_closure: (ancestor_id, descendant_id) PK，自引用 depth=0
func SeedRoot(opt *SeedRootOptions) error {
	db := common.DB
	if db == nil {
		return fmt.Errorf("passport DB 未初始化（请先 InitDBWithDriver）")
	}

	cfg := opt.normalize()
	if len(cfg.Cellphone) != 11 {
		return fmt.Errorf("root cellphone 必须为 11 位（users.cellphone VARCHAR(11)），当前: %q", cfg.Cellphone)
	}

	passwordHash := common.EncryPWD(cfg.Password)

	// ext / info / configuration 与业务模型 JSON tag 对齐，避免手写 JSON 字段名偏差
	extJSON, err := json.Marshal(protos.MapStruct{"disabled": 0})
	if err != nil {
		return fmt.Errorf("marshal user.ext: %w", err)
	}
	infoJSON, err := json.Marshal(protos.MapStruct{"adminCellphone": cfg.Cellphone})
	if err != nil {
		return fmt.Errorf("marshal tenant.info: %w", err)
	}
	confJSON, err := json.Marshal(protos.TenantConfiguration{
		Roles: []protos.RoleStruct{{
			RoleTitle: "超级管理员",
			RoleValue: "root",
		}},
		More: protos.MapStruct{},
	})
	if err != nil {
		return fmt.Errorf("marshal tenant.configuration: %w", err)
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := upsertRootUser(ctx, tx, db.DriverType(), cfg, passwordHash, extJSON); err != nil {
		return err
	}
	if err := upsertRootTenant(ctx, tx, db.DriverType(), cfg, infoJSON, confJSON); err != nil {
		return err
	}
	if err := upsertRootClosures(ctx, tx, db.DriverType(), cfg.UID, cfg.TenantID); err != nil {
		return err
	}
	if err := calibrateRootSequences(ctx, tx, db.DriverType(), cfg.UID, cfg.TenantID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

func upsertRootUser(ctx context.Context, tx database.Tx, driver database.DriverType, cfg SeedRootOptions, passwordHash string, extJSON []byte) error {
	var sql string
	var args []interface{}

	switch driver {
	case database.DriverPostgreSQL:
		// 与 README：ext JSONB，时间 TIMESTAMPTZ（NOW()）
		sql = `
INSERT INTO users (
  uid, tenant_id, nickname, cellphone, email, password, ext, create_time, update_time
) VALUES (
  $1, $2, $3, $4, $5, $6, $7::jsonb, NOW(), NOW()
)
ON CONFLICT (uid) DO UPDATE SET
  tenant_id   = EXCLUDED.tenant_id,
  nickname    = EXCLUDED.nickname,
  cellphone   = EXCLUDED.cellphone,
  email       = EXCLUDED.email,
  password    = EXCLUDED.password,
  ext         = COALESCE(EXCLUDED.ext, '{}'::jsonb),
  update_time = NOW();`
		args = []interface{}{cfg.UID, cfg.TenantID, cfg.Nickname, cfg.Cellphone, cfg.Email, passwordHash, string(extJSON)}
	default:
		// SQLite：JSON 以 TEXT 存储（dao.Init JSONType）
		sql = `
INSERT INTO users (
  uid, tenant_id, nickname, cellphone, email, password, ext, create_time, update_time
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT (uid) DO UPDATE SET
  tenant_id   = excluded.tenant_id,
  nickname    = excluded.nickname,
  cellphone   = excluded.cellphone,
  email       = excluded.email,
  password    = excluded.password,
  ext         = COALESCE(excluded.ext, '{}'),
  update_time = CURRENT_TIMESTAMP;`
		args = []interface{}{cfg.UID, cfg.TenantID, cfg.Nickname, cfg.Cellphone, cfg.Email, passwordHash, string(extJSON)}
	}

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("写入 users: %w", err)
	}
	return nil
}

func upsertRootTenant(ctx context.Context, tx database.Tx, driver database.DriverType, cfg SeedRootOptions, infoJSON, confJSON []byte) error {
	var sql string
	var args []interface{}

	switch driver {
	case database.DriverPostgreSQL:
		sql = `
INSERT INTO tenants (
  id, uid, tenant_name, tenant_type, info, configuration, create_time, update_time
) VALUES (
  $1, $2, $3, 'admin', $4::jsonb, $5::jsonb, NOW(), NOW()
)
ON CONFLICT (id) DO UPDATE SET
  uid           = EXCLUDED.uid,
  tenant_name   = EXCLUDED.tenant_name,
  tenant_type   = EXCLUDED.tenant_type,
  info          = EXCLUDED.info,
  configuration = EXCLUDED.configuration,
  update_time   = NOW();`
		args = []interface{}{cfg.TenantID, cfg.UID, cfg.TenantName, string(infoJSON), string(confJSON)}
	default:
		sql = `
INSERT INTO tenants (
  id, uid, tenant_name, tenant_type, info, configuration, create_time, update_time
) VALUES (
  ?, ?, ?, 'admin', ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT (id) DO UPDATE SET
  uid           = excluded.uid,
  tenant_name   = excluded.tenant_name,
  tenant_type   = excluded.tenant_type,
  info          = excluded.info,
  configuration = excluded.configuration,
  update_time   = CURRENT_TIMESTAMP;`
		args = []interface{}{cfg.TenantID, cfg.UID, cfg.TenantName, string(infoJSON), string(confJSON)}
	}

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("写入 tenants: %w", err)
	}
	return nil
}

func upsertRootClosures(ctx context.Context, tx database.Tx, driver database.DriverType, uid, tenantID uint64) error {
	// 与 TenantClosureInsert / README 闭包表一致：自引用 depth=0
	var userSQL, tenantSQL string
	var userArgs, tenantArgs []interface{}

	switch driver {
	case database.DriverPostgreSQL:
		userSQL = `
INSERT INTO user_closure (ancestor_id, descendant_id, depth)
VALUES ($1, $2, 0)
ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;`
		tenantSQL = `
INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
VALUES ($1, $2, 0)
ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;`
		userArgs = []interface{}{uid, uid}
		tenantArgs = []interface{}{tenantID, tenantID}
	default:
		userSQL = `
INSERT INTO user_closure (ancestor_id, descendant_id, depth)
VALUES (?, ?, 0)
ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;`
		tenantSQL = `
INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
VALUES (?, ?, 0)
ON CONFLICT (ancestor_id, descendant_id) DO NOTHING;`
		userArgs = []interface{}{uid, uid}
		tenantArgs = []interface{}{tenantID, tenantID}
	}

	if _, err := tx.Exec(ctx, userSQL, userArgs...); err != nil {
		return fmt.Errorf("写入 user_closure: %w", err)
	}
	if _, err := tx.Exec(ctx, tenantSQL, tenantArgs...); err != nil {
		return fmt.Errorf("写入 tenant_closure: %w", err)
	}
	return nil
}

func calibrateRootSequences(ctx context.Context, tx database.Tx, driver database.DriverType, uid, tenantID uint64) error {
	if driver != database.DriverPostgreSQL {
		// SQLite AUTOINCREMENT 在显式写入较大 ROWID 后会抬高 sqlite_sequence
		return nil
	}
	// README 空库用 ALTER SEQUENCE ... RESTART WITH 10000；
	// 显式插入 uid/id=10000 后需把序列至少推到已用最大值，避免 nextval 冲突。
	stmts := []string{
		fmt.Sprintf(`SELECT setval('users_uid_seq', GREATEST((SELECT COALESCE(MAX(uid), %d) FROM users), %d), true);`, uid, uid),
		fmt.Sprintf(`SELECT setval('tenants_id_seq', GREATEST((SELECT COALESCE(MAX(id), %d) FROM tenants), %d), true);`, tenantID, tenantID),
	}
	for _, sql := range stmts {
		if _, err := tx.Exec(ctx, sql); err != nil {
			return fmt.Errorf("校准序列: %w", err)
		}
	}
	return nil
}

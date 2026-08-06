package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liuhengloveyou/passport/v3/database"
)

// EnsureDatabase 确保 PostgreSQL 目标库已创建，并初始化表结构（幂等）。
// 从 dsn 解析 dbname，连到 postgres 系统库执行 CREATE DATABASE，再连目标库建表；sqlite3 无需建库。
func EnsureDatabase(driver, dsn string) error {
	if dsn == "" {
		return fmt.Errorf("db_dsn 为空")
	}
	driverType := database.DriverType(driver)
	if driver != "" && driverType != database.DriverPostgreSQL {
		return nil
	}

	dbName := dsnParam(dsn, "dbname")
	if dbName == "" {
		return fmt.Errorf("db_dsn 缺少 dbname")
	}

	if dbName != "postgres" {
		if err := ensurePostgresDatabase(dsn, dbName); err != nil {
			return err
		}
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("连接目标数据库失败: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("连接目标数据库失败: %w", err)
	}
	if err := InitDBTable(pool); err != nil {
		return fmt.Errorf("初始化表结构失败: %w", err)
	}
	fmt.Println("passport 表结构初始化完成")
	return nil
}

func ensurePostgresDatabase(dsn, dbName string) error {
	adminDSN := replaceDSNParam(dsn, "dbname", "postgres")
	pool, err := pgxpool.New(context.Background(), adminDSN)
	if err != nil {
		return fmt.Errorf("连接 postgres 系统库失败: %w", err)
	}
	defer pool.Close()

	ctx := context.Background()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("连接 postgres 系统库失败: %w", err)
	}

	var exists bool
	if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists); err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}
	if exists {
		fmt.Printf("数据库 %s 已存在，跳过创建\n", dbName)
		return nil
	}

	// CREATE DATABASE 不能参数化；库名来自配置，转义双引号
	safeName := strings.ReplaceAll(dbName, `"`, `""`)
	if _, err := pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, safeName)); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			fmt.Printf("数据库 %s 已存在，跳过创建\n", dbName)
			return nil
		}
		return fmt.Errorf("创建数据库 %s 失败: %w", dbName, err)
	}
	fmt.Printf("数据库 %s 已创建\n", dbName)
	return nil
}

// InitDBTable 初始化 PostgreSQL 业务表结构（幂等）。
func InitDBTable(db *pgxpool.Pool) error {
	ctx := context.Background()

	_, err := db.Exec(ctx, `
		-- 用户表
		CREATE TABLE IF NOT EXISTS users 
		(
		uid BIGSERIAL PRIMARY KEY,
		tenant_id BIGINT NOT NULL DEFAULT 0,
		nickname VARCHAR(64) UNIQUE,
		cellphone VARCHAR(11) UNIQUE,
		email VARCHAR(255) UNIQUE,
		wx_openid VARCHAR(64) UNIQUE,
		password VARCHAR(512) NOT NULL,
		avatar_url VARCHAR(255),
		gender SMALLINT,
		addr VARCHAR(1024),
		province VARCHAR(64),
		city VARCHAR(64),
		ext JSONB,
		create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		login_time TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id) WITH (deduplicate_items=True);
		DO $$
		BEGIN
			IF (SELECT last_value FROM users_uid_seq) < 10000 THEN
				ALTER SEQUENCE users_uid_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建用户表失败: %w", err)
	}

	_, err = db.Exec(ctx, `
		-- 租户表
		CREATE TABLE IF NOT EXISTS tenants (
			id BIGSERIAL PRIMARY KEY,
			uid BIGINT NOT NULL DEFAULT 0,
			tenant_name VARCHAR(255) NOT NULL UNIQUE,
			tenant_type VARCHAR(45) NOT NULL DEFAULT '',
			info JSONB,
			configuration JSONB,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_tenants_tenant_name ON tenants(tenant_name);
		DO $$
		BEGIN
			IF (SELECT last_value FROM tenants_id_seq) < 10000 THEN
				ALTER SEQUENCE tenants_id_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建租户表失败: %w", err)
	}

	_, err = db.Exec(ctx, `
		-- 权限表
		CREATE TABLE IF NOT EXISTS permission (
			id BIGSERIAL PRIMARY KEY,
			tenant_id BIGINT NOT NULL,
			domain VARCHAR(128) NOT NULL,
			title VARCHAR(128) NOT NULL,
			value VARCHAR(256) NOT NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (tenant_id, domain, title),
			UNIQUE (value, domain, tenant_id)
		);
		CREATE INDEX IF NOT EXISTS idx_permission_tenant_id ON permission(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_permission_domain ON permission(domain);
		DO $$
		BEGIN
			IF (SELECT last_value FROM permission_id_seq) < 10000 THEN
				ALTER SEQUENCE permission_id_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建权限表失败: %w", err)
	}

	_, err = db.Exec(ctx, `
		-- 部门表
		CREATE TABLE IF NOT EXISTS departments (
			id BIGSERIAL PRIMARY KEY,
			parent_id BIGINT NOT NULL DEFAULT 0,
			uid BIGINT NOT NULL,
			tenant_id BIGINT NOT NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			name VARCHAR(16) NOT NULL,
			config JSONB,
			UNIQUE (tenant_id, name)
		);
		CREATE INDEX IF NOT EXISTS idx_departments_tenant_id ON departments(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id);
		CREATE INDEX IF NOT EXISTS idx_departments_uid ON departments(uid);
		DO $$
		BEGIN
			IF (SELECT last_value FROM departments_id_seq) < 10000 THEN
				ALTER SEQUENCE departments_id_seq RESTART WITH 10000;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("创建部门表失败: %w", err)
	}

	_, err = db.Exec(ctx, `
		-- 用户闭包表
		CREATE TABLE IF NOT EXISTS user_closure (
			ancestor_id BIGINT NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
			descendant_id BIGINT NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
			depth INT NOT NULL CHECK (depth >= 0),
			PRIMARY KEY (ancestor_id, descendant_id)
		);
		CREATE INDEX IF NOT EXISTS idx_user_closure_ancestor ON user_closure(ancestor_id);
		CREATE INDEX IF NOT EXISTS idx_user_closure_descendant ON user_closure(descendant_id);
	`)
	if err != nil {
		return fmt.Errorf("创建用户闭包表失败: %w", err)
	}

	_, err = db.Exec(ctx, `
		-- 租户闭包表
		CREATE TABLE IF NOT EXISTS tenant_closure (
			ancestor_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			descendant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			depth INT NOT NULL CHECK (depth >= 0),
			PRIMARY KEY (ancestor_id, descendant_id)
		);
		CREATE INDEX IF NOT EXISTS idx_tenant_closure_tenant_id ON tenant_closure(ancestor_id);
		CREATE INDEX IF NOT EXISTS idx_tenant_closure_ancestor_id ON tenant_closure(descendant_id);
	`)
	if err != nil {
		return fmt.Errorf("创建租户闭包表失败: %w", err)
	}

	return nil
}

func dsnParam(dsn, key string) string {
	prefix := key + "="
	for _, part := range strings.Fields(dsn) {
		if strings.HasPrefix(part, prefix) {
			return strings.TrimPrefix(part, prefix)
		}
	}
	return ""
}

func replaceDSNParam(dsn, key, value string) string {
	prefix := key + "="
	parts := strings.Fields(dsn)
	out := make([]string, 0, len(parts))
	found := false
	for _, part := range parts {
		if strings.HasPrefix(part, prefix) {
			out = append(out, prefix+value)
			found = true
			continue
		}
		out = append(out, part)
	}
	if !found {
		out = append(out, prefix+value)
	}
	return strings.Join(out, " ")
}

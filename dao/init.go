package dao

import (
	"context"
	"fmt"

	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/database"
	"github.com/liuhengloveyou/passport/v3/protos"
	"go.uber.org/zap"
)

// Init 根据配置文件初始化数据库结构
// 支持PostgreSQL和SQLite3
func Init(options *protos.OptionStruct) error {
	if options == nil {
		return fmt.Errorf("配置选项不能为空")
	}

	// 确定数据库驱动和DSN
	var driver database.DriverType
	var dsn string

	if options.DBDriver != "" && options.DBDSN != "" {
		// 使用新配置
		driver = database.DriverType(options.DBDriver)
		dsn = options.DBDSN
	} else {
		return fmt.Errorf("未配置数据库连接信息（需要设置db_driver和db_dsn）")
	}

	// 创建数据库连接
	db, err := database.NewDB(driver, dsn)
	if err != nil {
		return fmt.Errorf("创建数据库连接失败: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 测试连接
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 根据数据库类型初始化表结构
	dialect := database.NewDialect(driver)
	if err := initTables(ctx, db, dialect); err != nil {
		return fmt.Errorf("初始化数据库表失败: %w", err)
	}

	if common.Logger != nil {
		common.Logger.Info("数据库表结构初始化成功",
			zap.String("driver", string(driver)),
			zap.String("dsn", maskDSN(dsn)))
	}

	return nil
}

// initTables 初始化所有表结构
func initTables(ctx context.Context, db database.DB, dialect database.Dialect) error {
	// 创建用户表
	if err := createUsersTable(ctx, db, dialect); err != nil {
		return fmt.Errorf("创建用户表失败: %w", err)
	} else {
		fmt.Println("创建用户表成功")
	}

	// 创建租户表
	if err := createTenantsTable(ctx, db, dialect); err != nil {
		fmt.Println("创建租户表失败: %w", err)
	} else {
		fmt.Println("创建租户表成功")
	}

	// 创建权限表
	if err := createPermissionTable(ctx, db, dialect); err != nil {
		fmt.Println("创建权限表失败: %w", err)
	} else {
		fmt.Println("创建权限表成功")
	}

	// 创建部门表
	if err := createDepartmentsTable(ctx, db, dialect); err != nil {
		fmt.Println("创建部门表失败: %w", err)
	} else {
		fmt.Println("创建部门表成功")
	}

	// 创建用户闭包表
	if err := createUserClosureTable(ctx, db, dialect); err != nil {
		fmt.Println("创建用户闭包表失败: %w", err)
	} else {
		fmt.Println("创建用户闭包表成功")
	}

	// 创建租户闭包表
	if err := createTenantClosureTable(ctx, db, dialect); err != nil {
		fmt.Println("创建租户闭包表失败: %w", err)
	} else {
		fmt.Println("创建租户闭包表成功")
	}

	return nil
}

// createUsersTable 创建用户表
func createUsersTable(ctx context.Context, db database.DB, dialect database.Dialect) error {
	jsonType := dialect.JSONType()
	autoIncrement := dialect.AutoIncrement()
	timestampType := getTimestampType(dialect)

	// SQLite3的AutoIncrement已经包含PRIMARY KEY，PostgreSQL的BIGSERIAL需要单独指定
	primaryKey := ""
	if db.DriverType() == database.DriverPostgreSQL {
		primaryKey = "PRIMARY KEY"
	}

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS users (
			uid %s %s,
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
			ext %s,
			create_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			login_time %s
		)`, autoIncrement, primaryKey, jsonType, timestampType, timestampType, timestampType)

	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}

	// 创建索引
	indexSQL := "CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id)"
	if db.DriverType() == database.DriverPostgreSQL {
		indexSQL += " WITH (deduplicate_items=True)"
	}
	if _, err := db.Exec(ctx, indexSQL); err != nil {
		return err
	}

	// PostgreSQL需要设置序列起始值
	if db.DriverType() == database.DriverPostgreSQL {
		seqSQL := `
			DO $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_sequences WHERE sequencename = 'users_uid_seq') THEN
					IF (SELECT last_value FROM users_uid_seq) < 10000 THEN
						ALTER SEQUENCE users_uid_seq RESTART WITH 10000;
					END IF;
				END IF;
			END $$;
		`
		if _, err := db.Exec(ctx, seqSQL); err != nil {
			return err
		}
	}

	return nil
}

// createTenantsTable 创建租户表
func createTenantsTable(ctx context.Context, db database.DB, dialect database.Dialect) error {
	jsonType := dialect.JSONType()
	autoIncrement := dialect.AutoIncrement()
	timestampType := getTimestampType(dialect)

	// SQLite3的AutoIncrement已经包含PRIMARY KEY，PostgreSQL的BIGSERIAL需要单独指定
	primaryKey := ""
	if db.DriverType() == database.DriverPostgreSQL {
		primaryKey = "PRIMARY KEY"
	}

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS tenants (
			id %s %s,
			uid BIGINT NOT NULL DEFAULT 0,
			tenant_name VARCHAR(255) NOT NULL UNIQUE,
			tenant_type VARCHAR(45) NOT NULL DEFAULT '',
			info %s,
			configuration %s,
			create_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`, autoIncrement, primaryKey, jsonType, jsonType, timestampType, timestampType)

	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tenants_tenant_name ON tenants(tenant_name)",
	}
	for _, idxSQL := range indexes {
		if _, err := db.Exec(ctx, idxSQL); err != nil {
			return err
		}
	}

	// PostgreSQL需要设置序列起始值
	if db.DriverType() == database.DriverPostgreSQL {
		seqSQL := `
			DO $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_sequences WHERE sequencename = 'tenants_id_seq') THEN
					IF (SELECT last_value FROM tenants_id_seq) < 10000 THEN
						ALTER SEQUENCE tenants_id_seq RESTART WITH 10000;
					END IF;
				END IF;
			END $$;
		`
		if _, err := db.Exec(ctx, seqSQL); err != nil {
			return err
		}
	}

	return nil
}

// createPermissionTable 创建权限表
func createPermissionTable(ctx context.Context, db database.DB, dialect database.Dialect) error {
	autoIncrement := dialect.AutoIncrement()
	timestampType := getTimestampType(dialect)

	// SQLite3的AutoIncrement已经包含PRIMARY KEY，PostgreSQL的BIGSERIAL需要单独指定
	primaryKey := ""
	if db.DriverType() == database.DriverPostgreSQL {
		primaryKey = "PRIMARY KEY"
	}

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS permission (
			id %s %s,
			tenant_id BIGINT NOT NULL,
			domain VARCHAR(128) NOT NULL,
			title VARCHAR(128) NOT NULL,
			value VARCHAR(256) NOT NULL,
			create_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (tenant_id, domain, title),
			UNIQUE (value, domain, tenant_id)
		)`, autoIncrement, primaryKey, timestampType, timestampType)

	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_permission_tenant_id ON permission(tenant_id)",
		"CREATE INDEX IF NOT EXISTS idx_permission_domain ON permission(domain)",
	}
	for _, idxSQL := range indexes {
		if _, err := db.Exec(ctx, idxSQL); err != nil {
			return err
		}
	}

	// PostgreSQL需要设置序列起始值
	if db.DriverType() == database.DriverPostgreSQL {
		seqSQL := `
			DO $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_sequences WHERE sequencename = 'permission_id_seq') THEN
					IF (SELECT last_value FROM permission_id_seq) < 10000 THEN
						ALTER SEQUENCE permission_id_seq RESTART WITH 10000;
					END IF;
				END IF;
			END $$;
		`
		if _, err := db.Exec(ctx, seqSQL); err != nil {
			return err
		}
	}

	return nil
}

// createDepartmentsTable 创建部门表
func createDepartmentsTable(ctx context.Context, db database.DB, dialect database.Dialect) error {
	jsonType := dialect.JSONType()
	autoIncrement := dialect.AutoIncrement()
	timestampType := getTimestampType(dialect)

	// SQLite3的AutoIncrement已经包含PRIMARY KEY，PostgreSQL的BIGSERIAL需要单独指定
	primaryKey := ""
	if db.DriverType() == database.DriverPostgreSQL {
		primaryKey = "PRIMARY KEY"
	}

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS departments (
			id %s %s,
			parent_id BIGINT NOT NULL DEFAULT 0,
			uid BIGINT NOT NULL,
			tenant_id BIGINT NOT NULL,
			create_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time %s NOT NULL DEFAULT CURRENT_TIMESTAMP,
			name VARCHAR(16) NOT NULL,
			config %s,
			UNIQUE (tenant_id, name)
		)`, autoIncrement, primaryKey, timestampType, timestampType, jsonType)

	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_departments_tenant_id ON departments(tenant_id)",
		"CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id)",
		"CREATE INDEX IF NOT EXISTS idx_departments_uid ON departments(uid)",
	}
	for _, idxSQL := range indexes {
		if _, err := db.Exec(ctx, idxSQL); err != nil {
			return err
		}
	}

	// PostgreSQL需要设置序列起始值
	if db.DriverType() == database.DriverPostgreSQL {
		seqSQL := `
			DO $$
			BEGIN
				IF EXISTS (SELECT 1 FROM pg_sequences WHERE sequencename = 'departments_id_seq') THEN
					IF (SELECT last_value FROM departments_id_seq) < 10000 THEN
						ALTER SEQUENCE departments_id_seq RESTART WITH 10000;
					END IF;
				END IF;
			END $$;
		`
		if _, err := db.Exec(ctx, seqSQL); err != nil {
			return err
		}
	}

	return nil
}

// createUserClosureTable 创建用户闭包表
func createUserClosureTable(ctx context.Context, db database.DB, dialect database.Dialect) error {
	// SQLite3不支持外键约束（除非显式启用），所以需要处理
	foreignKeySQL := ""
	if db.DriverType() == database.DriverPostgreSQL {
		foreignKeySQL = `
			ancestor_id BIGINT NOT NULL REFERENCES users(uid) ON DELETE CASCADE,
			descendant_id BIGINT NOT NULL REFERENCES users(uid) ON DELETE CASCADE,`
	} else {
		foreignKeySQL = `
			ancestor_id BIGINT NOT NULL,
			descendant_id BIGINT NOT NULL,`
	}

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS user_closure (
			%s
			depth INT NOT NULL CHECK (depth >= 0),
			PRIMARY KEY (ancestor_id, descendant_id)
		)`, foreignKeySQL)

	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_user_closure_ancestor ON user_closure(ancestor_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_closure_descendant ON user_closure(descendant_id)",
	}
	for _, idxSQL := range indexes {
		if _, err := db.Exec(ctx, idxSQL); err != nil {
			return err
		}
	}

	return nil
}

// createTenantClosureTable 创建租户闭包表
func createTenantClosureTable(ctx context.Context, db database.DB, dialect database.Dialect) error {
	// SQLite3不支持外键约束（除非显式启用），所以需要处理
	foreignKeySQL := ""
	if db.DriverType() == database.DriverPostgreSQL {
		foreignKeySQL = `
			ancestor_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			descendant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,`
	} else {
		foreignKeySQL = `
			ancestor_id BIGINT NOT NULL,
			descendant_id BIGINT NOT NULL,`
	}

	sql := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS tenant_closure (
			%s
			depth INT NOT NULL CHECK (depth >= 0),
			PRIMARY KEY (ancestor_id, descendant_id)
		)`, foreignKeySQL)

	if _, err := db.Exec(ctx, sql); err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tenant_closure_ancestor ON tenant_closure(ancestor_id)",
		"CREATE INDEX IF NOT EXISTS idx_tenant_closure_descendant ON tenant_closure(descendant_id)",
	}
	for _, idxSQL := range indexes {
		if _, err := db.Exec(ctx, idxSQL); err != nil {
			return err
		}
	}

	return nil
}

// getTimestampType 获取时间戳类型
func getTimestampType(dialect database.Dialect) string {
	// 根据数据库类型返回时间戳类型
	switch dialect.(type) {
	case *database.PostgresDialect:
		return "TIMESTAMPTZ"
	case *database.SQLite3Dialect:
		return "TIMESTAMP"
	default:
		return "TIMESTAMP"
	}
}

// maskDSN 隐藏DSN中的敏感信息（用于日志）
func maskDSN(dsn string) string {
	// 简单实现：隐藏密码部分
	if len(dsn) > 50 {
		return dsn[:20] + "..."
	}
	return dsn
}

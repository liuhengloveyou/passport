package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MigrationConfig struct {
	MySQLDSN    string
	PostgresDSN string
	BatchSize   int
}

type Migrator struct {
	mysqlDB    *sql.DB
	postgresDB *pgxpool.Pool
	config     *MigrationConfig
}

func NewMigrator(config *MigrationConfig) (*Migrator, error) {
	// 连接MySQL
	mysqlDB, err := sql.Open("mysql", config.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("连接MySQL失败: %w", err)
	}

	if err := mysqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL连接测试失败: %w", err)
	}

	// 连接PostgreSQL
	postgresDB, err := pgxpool.New(context.Background(), config.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	if err := postgresDB.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("PostgreSQL连接测试失败: %w", err)
	}

	return &Migrator{
		mysqlDB:    mysqlDB,
		postgresDB: postgresDB,
		config:     config,
	}, nil
}

func (m *Migrator) Close() {
	if m.mysqlDB != nil {
		m.mysqlDB.Close()
	}
	if m.postgresDB != nil {
		m.postgresDB.Close()
	}
}

// 迁移用户表
func (m *Migrator) MigrateUsers() error {
	log.Println("开始迁移用户表...")

	// 查询MySQL用户数据
	rows, err := m.mysqlDB.Query(`
		SELECT uid, tenant_id, cellphone, email, nickname, password, 
		       avatar_url, gender, addr, ext, add_time, update_time, 
		       login_time, wx_openid 
		FROM users ORDER BY uid
	`)
	if err != nil {
		return fmt.Errorf("查询MySQL用户数据失败: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var (
			uid        int64
			tenantID   int64
			cellphone  sql.NullString
			email      sql.NullString
			nickname   sql.NullString
			password   string
			avatarURL  sql.NullString
			gender     sql.NullInt32
			addr       sql.NullString
			ext        sql.NullString
			addTime    time.Time
			updateTime time.Time
			loginTime  sql.NullTime
			wxOpenid   sql.NullString
		)

		err := rows.Scan(&uid, &tenantID, &cellphone, &email, &nickname,
			&password, &avatarURL, &gender, &addr, &ext, &addTime,
			&updateTime, &loginTime, &wxOpenid)
		if err != nil {
			return fmt.Errorf("扫描用户数据失败: %w", err)
		}

		// 处理JSON字段
		var extJSON interface{}
		if ext.Valid && ext.String != "" {
			if err := json.Unmarshal([]byte(ext.String), &extJSON); err != nil {
				log.Printf("警告: 用户%d的ext字段JSON解析失败: %v", uid, err)
				extJSON = nil
			}
		}

		// 插入PostgreSQL - 修复字段名称
		_, err = m.postgresDB.Exec(context.Background(), `
			INSERT INTO users (uid, tenant_id, cellphone, email, nickname, password, 
			                  avatar_url, gender, addr, ext, create_time, update_time, 
			                  login_time, wx_openid) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (uid) DO UPDATE SET
				tenant_id = EXCLUDED.tenant_id,
				cellphone = EXCLUDED.cellphone,
				email = EXCLUDED.email,
				nickname = EXCLUDED.nickname,
				password = EXCLUDED.password,
				avatar_url = EXCLUDED.avatar_url,
				gender = EXCLUDED.gender,
				addr = EXCLUDED.addr,
				ext = EXCLUDED.ext,
				update_time = EXCLUDED.update_time,
				login_time = EXCLUDED.login_time,
				wx_openid = EXCLUDED.wx_openid
		`, uid, tenantID, cellphone, email, nickname, password,
			avatarURL, gender, addr, extJSON, addTime, updateTime,
			loginTime, wxOpenid)

		if err != nil {
			return fmt.Errorf("插入用户数据失败 (uid=%d): %w", uid, err)
		}

		count++
		if count%100 == 0 {
			log.Printf("已迁移 %d 个用户", count)
		}
	}

	log.Printf("用户表迁移完成，共迁移 %d 条记录", count)
	return nil
}

// 迁移租户表
func (m *Migrator) MigrateTenants() error {
	log.Println("开始迁移租户表...")

	rows, err := m.mysqlDB.Query(`
		SELECT id, uid, tenant_name, tenant_type, info, configuration, 
		       add_time, update_time 
		FROM tenant ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("查询MySQL租户数据失败: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var (
			id            int64
			uid           int64
			tenantName    string
			tenantType    string
			info          sql.NullString
			configuration sql.NullString
			addTime       time.Time
			updateTime    time.Time
		)

		err := rows.Scan(&id, &uid, &tenantName, &tenantType, &info,
			&configuration, &addTime, &updateTime)
		if err != nil {
			return fmt.Errorf("扫描租户数据失败: %w", err)
		}

		// 处理JSON字段
		var infoJSON, configJSON interface{}
		if info.Valid && info.String != "" {
			if err := json.Unmarshal([]byte(info.String), &infoJSON); err != nil {
				log.Printf("警告: 租户%d的info字段JSON解析失败: %v", id, err)
			}
		}
		if configuration.Valid && configuration.String != "" {
			if err := json.Unmarshal([]byte(configuration.String), &configJSON); err != nil {
				log.Printf("警告: 租户%d的configuration字段JSON解析失败: %v", id, err)
			}
		}

		// 插入PostgreSQL
		_, err = m.postgresDB.Exec(context.Background(), `
			INSERT INTO tenants (id, uid, tenant_name, tenant_type, info, 
			                    configuration, create_time, update_time) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				uid = EXCLUDED.uid,
				tenant_name = EXCLUDED.tenant_name,
				tenant_type = EXCLUDED.tenant_type,
				info = EXCLUDED.info,
				configuration = EXCLUDED.configuration,
				update_time = EXCLUDED.update_time
		`, id, uid, tenantName, tenantType, infoJSON, configJSON,
			addTime, updateTime)

		if err != nil {
			return fmt.Errorf("插入租户数据失败 (id=%d): %w", id, err)
		}

		count++
	}

	log.Printf("租户表迁移完成，共迁移 %d 条记录", count)
	return nil
}

// 迁移权限表
func (m *Migrator) MigratePermissions() error {
	log.Println("开始迁移权限表...")

	rows, err := m.mysqlDB.Query(`
		SELECT id, tenant_id, domain, title, value, add_time, update_time 
		FROM permission ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("查询MySQL权限数据失败: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var (
			id         int64
			tenantID   int64
			domain     string
			title      string
			value      string
			addTime    time.Time
			updateTime time.Time
		)

		err := rows.Scan(&id, &tenantID, &domain, &title, &value,
			&addTime, &updateTime)
		if err != nil {
			return fmt.Errorf("扫描权限数据失败: %w", err)
		}

		// 插入PostgreSQL
		_, err = m.postgresDB.Exec(context.Background(), `
			INSERT INTO permission (id, tenant_id, domain, title, value, create_time, update_time) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				tenant_id = EXCLUDED.tenant_id,
				domain = EXCLUDED.domain,
				title = EXCLUDED.title,
				value = EXCLUDED.value,
				update_time = EXCLUDED.update_time
		`, id, tenantID, domain, title, value, addTime, updateTime)

		if err != nil {
			return fmt.Errorf("插入权限数据失败 (id=%d): %w", id, err)
		}

		count++
	}

	log.Printf("权限表迁移完成，共迁移 %d 条记录", count)
	return nil
}

// 迁移部门表
func (m *Migrator) MigrateDepartments() error {
	log.Println("开始迁移部门表...")

	rows, err := m.mysqlDB.Query(`
		SELECT id, parent_id, uid, tenant_id, add_time, update_time, name, config 
		FROM departments ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("查询MySQL部门数据失败: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var (
			id         int64
			parentID   int64
			uid        int64
			tenantID   int64
			addTime    time.Time
			updateTime time.Time
			name       string
			config     sql.NullString
		)

		err := rows.Scan(&id, &parentID, &uid, &tenantID, &addTime,
			&updateTime, &name, &config)
		if err != nil {
			return fmt.Errorf("扫描部门数据失败: %w", err)
		}

		// 处理JSON字段
		var configJSON interface{}
		if config.Valid && config.String != "" {
			if err := json.Unmarshal([]byte(config.String), &configJSON); err != nil {
				log.Printf("警告: 部门%d的config字段JSON解析失败: %v", id, err)
			}
		}

		// 插入PostgreSQL
		_, err = m.postgresDB.Exec(context.Background(), `
			INSERT INTO departments (id, parent_id, uid, tenant_id, 
			                        create_time, update_time, name, config) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				parent_id = EXCLUDED.parent_id,
				uid = EXCLUDED.uid,
				tenant_id = EXCLUDED.tenant_id,
				update_time = EXCLUDED.update_time,
				name = EXCLUDED.name,
				config = EXCLUDED.config
		`, id, parentID, uid, tenantID, addTime, updateTime, name, configJSON)

		if err != nil {
			return fmt.Errorf("插入部门数据失败 (id=%d): %w", id, err)
		}

		count++
	}

	log.Printf("部门表迁移完成，共迁移 %d 条记录", count)
	return nil
}

// 执行完整迁移
func (m *Migrator) MigrateAll() error {
	log.Println("开始数据库迁移...")

	// 按依赖顺序迁移表
	if err := m.MigrateTenants(); err != nil {
		return err
	}

	if err := m.MigrateUsers(); err != nil {
		return err
	}

	if err := m.MigratePermissions(); err != nil {
		return err
	}

	if err := m.MigrateDepartments(); err != nil {
		return err
	}

	log.Println("数据库迁移完成！")
	return nil
}

func main() {
	config := &MigrationConfig{
		// MySQL连接字符串
		MySQLDSN: "root:lhisroot@tcp(localhost:3306)/passport?charset=utf8mb4&parseTime=True&loc=Local",
		// PostgreSQL连接字符串
		PostgresDSN: "host=localhost user=club password=club16888 dbname=club port=5432 sslmode=disable TimeZone=Asia/Shanghai",
		BatchSize:   1000,
	}

	migrator, err := NewMigrator(config)
	if err != nil {
		log.Fatalf("创建迁移器失败: %v", err)
	}
	defer migrator.Close()

	if err := migrator.MigrateAll(); err != nil {
		log.Fatalf("迁移失败: %v", err)
	}
}

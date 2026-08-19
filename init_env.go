package passport

import (
	"fmt"
	"os"

	gocommon "github.com/liuhengloveyou/go-common"

	"github.com/liuhengloveyou/passport/v4/accessctl"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/protos"
	"github.com/liuhengloveyou/passport/v4/service"
)

const defaultRBACModel = "rbac_with_domains_model.conf"

// InitDatabaseEnv 供 CLI -init 使用：确保配置与日志就绪后执行 InitSystemEnv。
func InitDatabaseEnv() error {
	if common.ServConfig.DBDriver == "" || common.ServConfig.DBDSN == "" {
		configPath := os.Getenv("PASSPORT_CONFIG")
		if configPath == "" {
			configPath = "./passport.conf.yaml"
		}

		fmt.Printf("尝试从配置文件加载: %s\n", configPath)
		if err := gocommon.LoadYamlConfig(configPath, &common.ServConfig); err != nil {
			return fmt.Errorf("加载配置文件失败: %v\n请使用 -passport 参数指定配置文件路径，或设置 PASSPORT_CONFIG 环境变量", err)
		}
	}

	if common.ServConfig.DBDriver == "" || common.ServConfig.DBDSN == "" {
		return fmt.Errorf("未找到数据库配置，请检查配置文件")
	}

	if common.Logger == nil {
		logDir := common.ServConfig.LogDir
		if logDir == "" {
			logDir = "./logs"
		}
		logLevel := common.ServConfig.LogLevel
		if logLevel == "" {
			logLevel = "info"
		}
		if err := common.InitLog(logDir, logLevel); err != nil {
			fmt.Printf("警告: 初始化日志失败: %v\n", err)
		}
	}

	fmt.Printf("开始初始化 passport... driver: %s, dsn: %s\n", common.ServConfig.DBDriver, common.ServConfig.DBDSN)
	if err := InitSystemEnv(&common.ServConfig); err != nil {
		return err
	}
	fmt.Println("✓ passport 初始化成功！")
	return nil
}

// InitSystemEnv 初始化 passport 库表结构、写入 root 管理员/租户，并绑定 root 角色。
// 供 passport 自身 -init 以及宿主项目（如 struct-ocr）复用。
func InitSystemEnv(options *protos.OptionStruct) error {
	return InitSystemEnvWithRBAC(options, defaultRBACModel)
}

// InitSystemEnvWithRBAC 同 InitSystemEnv，可指定 casbin 模型文件路径。
func InitSystemEnvWithRBAC(options *protos.OptionStruct, rbacModel string) error {
	if options == nil {
		return fmt.Errorf("配置选项不能为空")
	}
	if options.DBDriver == "" || options.DBDSN == "" {
		return fmt.Errorf("未配置数据库连接信息（需要设置 db_driver 和 db_dsn）")
	}
	if rbacModel == "" {
		rbacModel = defaultRBACModel
	}

	// 先创建目标库（如 club-passport），再连库建表
	if err := common.EnsureDatabase(options.DBDriver, options.DBDSN); err != nil {
		return fmt.Errorf("确保 passport 数据库存在失败: %w", err)
	}

	if common.DB == nil {
		if err := common.InitDBWithDriver(options.DBDriver, options.DBDSN); err != nil {
			return fmt.Errorf("初始化 passport DB 失败: %w", err)
		}
	}

	// PostgreSQL 表已在 EnsureDatabase/InitDBTable 中创建；sqlite 等其它驱动走 dao.Init
	if options.DBDriver != "postgres" {
		if err := dao.Init(options); err != nil {
			return fmt.Errorf("初始化数据库表结构失败: %w", err)
		}
	}

	seed := &dao.SeedRootOptions{
		UID:      dao.DefaultRootUID,
		TenantID: dao.DefaultRootTenantID,
	}
	if options.RootUserID > 0 {
		seed.UID = options.RootUserID
	}
	if options.RootTenantID > 0 {
		seed.TenantID = options.RootTenantID
	}

	if err := dao.SeedRoot(seed); err != nil {
		return fmt.Errorf("初始化 root 管理员/租户失败: %w", err)
	}

	if err := accessctl.InitAccessControl(rbacModel, options.DBDriver, options.DBDSN); err != nil {
		return fmt.Errorf("初始化 accessctl 失败: %w", err)
	}
	orgs, err := dao.OrgListByTenant(seed.TenantID)
	if err != nil {
		return fmt.Errorf("查询 root 组织失败: %w", err)
	}
	if len(orgs) == 0 {
		if _, err := service.OrgCreate(seed.TenantID, "root"); err != nil {
			return fmt.Errorf("创建 root 组织失败: %w", err)
		}
	} else if err := accessctl.AddRoleForUserInDomain(seed.UID, seed.TenantID, orgs[0].ID, "root"); err != nil {
		return fmt.Errorf("绑定 root 角色失败: %w", err)
	}

	fmt.Printf("passport 初始化完成: nickname=%s password=%s uid=%d tenant_id=%d\n",
		dao.DefaultRootNickname, dao.DefaultRootPassword, seed.UID, seed.TenantID)
	return nil
}

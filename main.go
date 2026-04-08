package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	gocommon "github.com/liuhengloveyou/go-common"

	"github.com/liuhengloveyou/passport/v3/common"
	"github.com/liuhengloveyou/passport/v3/dao"
	facehttp "github.com/liuhengloveyou/passport/v3/face/http"
)

var (
	BuildTime string
	CommitID  string
	GitTag    string

	showVer    = flag.Bool("version", false, "show version")
	initEnv    = flag.Bool("init", false, "根据配置文件初始化数据库结构")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
)

func main() {
	// 先解析命令行参数，获取配置文件路径
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	if *showVer {
		fmt.Printf("%s\t%s\t%s\n", GitTag, CommitID, BuildTime)
		os.Exit(0)
	}

	// 初始化数据库结构
	if *initEnv {
		if err := initDatabaseEnv(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	gocommon.SingleInstane(common.ServConfig.PidFile)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			panic(err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic(err)
		}
		f.Close()
	}

	facehttp.InitAndRunHttpApi(&common.ServConfig)

}

// initDatabaseEnv 初始化数据库环境
// 根据配置文件初始化PostgreSQL或SQLite3数据库结构
func initDatabaseEnv() error {
	// 如果配置未加载，尝试重新加载（可能配置文件路径不对）
	if common.ServConfig.DBDriver == "" || common.ServConfig.DBDSN == "" {
		// 尝试从环境变量或默认路径加载配置
		configPath := os.Getenv("PASSPORT_CONFIG")
		if configPath == "" {
			configPath = "./passport.conf.yaml"
		}

		fmt.Printf("尝试从配置文件加载: %s\n", configPath)
		if err := gocommon.LoadYamlConfig(configPath, &common.ServConfig); err != nil {
			return fmt.Errorf("加载配置文件失败: %v\n请使用 -passport 参数指定配置文件路径，或设置 PASSPORT_CONFIG 环境变量", err)
		}
	}

	// 检查数据库配置
	if common.ServConfig.DBDriver == "" || common.ServConfig.DBDSN == "" {
		return fmt.Errorf(`未找到数据库配置，请检查配置文件`)
	}

	// 初始化日志（如果需要，用于输出初始化过程的日志）
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
			// 日志初始化失败不影响数据库初始化
			fmt.Printf("警告: 初始化日志失败: %v\n", err)
		}
	}

	// 使用dao.Init初始化数据库结构
	fmt.Printf("开始初始化数据库结构... driver: %s, dsn: %s\n", common.ServConfig.DBDriver, common.ServConfig.DBDSN)
	if err := dao.Init(&common.ServConfig); err != nil {
		return fmt.Errorf("初始化数据库结构失败: %w", err)
	}

	fmt.Println("✓ 数据库结构初始化成功！")
	return nil
}

package main // 声明主包，Go 程序的入口包

import (
	"context"     // 上下文包，用于控制协程的生命周期和取消信号
	"database/sql" // 数据库 SQL 包，提供统一的数据库操作接口
	"fmt"         // 格式化包，用于打印输出和格式化字符串
	"goswitch/internal/config"   // 导入项目内部的配置模块
	"goswitch/internal/service"  // 导入项目内部的服务模块
	"goswitch/pkg/database"      // 导入数据库类型定义
	"goswitch/pkg/util"          // 导入工具包
	"os"          // 操作系统包，提供文件操作、环境变量等功能
	"os/signal"   // 信号包，用于监听操作系统信号
	"syscall"     // 系统调用包，定义系统信号常量

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
	_ "github.com/lib/pq"              // PostgreSQL 驱动
	_ "github.com/sijms/go-ora/v2"     // Oracle 驱动
	"github.com/spf13/cobra"           // Cobra 命令行框架

	// 导入数据库方言，触发 init() 注册
	_ "goswitch/internal/product/mysql"      // MySQL 产品实现
	_ "goswitch/internal/product/postgresql" // PostgreSQL 产品实现
	_ "goswitch/internal/product/oracle"     // Oracle 产品实现
)

// Version 版本号
const Version = "1.2.0"

// rootCmd 定义根命令，这是 CLI 工具的顶级命令
var rootCmd = &cobra.Command{
	Use:   "goswitch",                    // 命令名称
	Short: "goswitch - 异构数据库迁移工具", // 简短描述，显示在帮助列表中
	Long:  "goswitch 是一个轻量级的数据库迁移工具，支持 MySQL、PostgreSQL 和 Oracle 数据库的全量迁移，包括 MySQL/PostgreSQL/Oracle 之间的任意组合迁移。", // 详细描述，显示在 --help 中
}

// versionCmd 定义 version 子命令，用于显示版本信息
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("goswitch version %s\n", Version)
	},
}

// runCmd 定义 run 子命令，用于执行数据库迁移
var runCmd = &cobra.Command{
	Use:   "run",                          // 子命令名称
	Short: "执行数据库迁移",                 // 简短描述
	Long:  "根据配置文件执行数据库迁移任务",   // 详细描述
	RunE:  runMigration,                   // 命令执行函数，RunE 返回 error 类型
}

// configFile 存储配置文件路径，通过命令行参数 -c 传入
var configFile string

// logFile 存储日志文件路径，通过命令行参数 -l 传入
var logFile string

// init 函数在 main 之前自动执行，用于初始化命令行参数和注册子命令
func init() {
	// 为 run 命令添加 -c/--config 标志，绑定到 configFile 变量
	runCmd.Flags().StringVarP(&configFile, "config", "c", "", "配置文件路径 (必需)")
	// 为 run 命令添加 -l/--log-file 标志，绑定到 logFile 变量
	runCmd.Flags().StringVarP(&logFile, "log-file", "l", "", "日志文件路径 (可选)")
	// 标记 config 标志为必需参数
	runCmd.MarkFlagRequired("config")
	// 将 run 子命令添加到根命令下
	rootCmd.AddCommand(runCmd)
	// 将 version 子命令添加到根命令下
	rootCmd.AddCommand(versionCmd)
}

// main 函数是程序入口点
func main() {
	// 执行根命令，解析命令行参数并运行对应的命令
	if err := rootCmd.Execute(); err != nil {
		// 如果执行出错，退出程序并返回状态码 1
		os.Exit(1)
	}
}

// getDriverName 根据数据库类型返回对应的 sql 驱动名称
func getDriverName(dbType database.DBType) string {
	switch dbType {
	case database.MySQL:
		return "mysql"
	case database.PostgreSQL:
		return "postgres"
	case database.Oracle:
		return "oracle"
	default:
		return string(dbType)
	}
}

// runMigration 是 run 命令的实际执行函数，负责整个迁移流程
func runMigration(cmd *cobra.Command, args []string) error {
	// 1. 初始化日志
	if logFile != "" {
		if err := util.InitLogger(util.INFO, logFile); err != nil {
			return fmt.Errorf("failed to init logger: %w", err)
		}
		defer util.CloseLogger()
		fmt.Printf("✓ 日志文件: %s\n", logFile)
	}

	// 2. 加载配置文件
	// 从指定路径读取 YAML 配置文件并解析为 Config 结构体
	cfg, err := config.Load(configFile)
	if err != nil {
		// 加载失败时返回包装后的错误信息
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 3. 连接源端数据库
	// 根据源端数据库类型选择对应的驱动
	srcDB, err := sql.Open(getDriverName(cfg.Source.Type), cfg.GetSourceDSN())
	if err != nil {
		return fmt.Errorf("failed to connect source database: %w", err)
	}
	// defer 确保函数退出时关闭数据库连接，释放资源
	defer srcDB.Close()

	// Ping 验证数据库连接是否可用
	if err := srcDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping source database: %w", err)
	}
	// 源端连接池配置
	srcDB.SetMaxOpenConns(10)    // 最大打开连接数
	srcDB.SetMaxIdleConns(5)     // 最大空闲连接数
	srcDB.SetConnMaxLifetime(0)  // 连接最大生命周期，0 表示不限制
	fmt.Println("✓ 源端数据库连接成功")
	util.LogInfo("源端数据库连接成功: %s@%s:%d/%s", cfg.Source.Username, cfg.Source.Host, cfg.Source.Port, cfg.Source.Database)

	// 4. 连接目标端数据库
	// 根据目标端数据库类型选择对应的驱动
	dstDB, err := sql.Open(getDriverName(cfg.Target.Type), cfg.GetTargetDSN())
	if err != nil {
		return fmt.Errorf("failed to connect target database: %w", err)
	}
	// defer 确保函数退出时关闭数据库连接
	defer dstDB.Close()

	// Ping 验证目标端数据库连接是否可用
	if err := dstDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping target database: %w", err)
	}
	// 目标端连接池配置（根据并行度调整）
	dstDB.SetMaxOpenConns(cfg.Target.Parallel * 2) // 最大连接数为并行度的 2 倍
	dstDB.SetMaxIdleConns(cfg.Target.Parallel)      // 最大空闲连接数等于并行度
	dstDB.SetConnMaxLifetime(0)                      // 连接最大生命周期，0 表示不限制
	fmt.Println("✓ 目标端数据库连接成功")
	util.LogInfo("目标端数据库连接成功: %s@%s:%d/%s", cfg.Target.Username, cfg.Target.Host, cfg.Target.Port, cfg.Target.Database)

	// 5. 创建迁移服务实例
	// 传入配置和两个数据库连接，创建 MigrationService
	migration, err := service.NewMigrationService(cfg, srcDB, dstDB)
	if err != nil {
		return fmt.Errorf("failed to create migration service: %w", err)
	}

	// 6. 处理信号，支持优雅退出
	// 创建可取消的上下文，用于控制迁移任务的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	// defer 确保函数退出时取消上下文，通知所有子协程停止
	defer cancel()

	// 创建信号通道，缓冲大小为 1
	sigCh := make(chan os.Signal, 1)
	// 注册要监听的信号：Ctrl+C (SIGINT) 和 kill 命令 (SIGTERM)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动协程监听系统信号
	go func() {
		// 阻塞等待信号
		sig := <-sigCh
		fmt.Printf("\n收到信号 %v，正在停止...\n", sig)
		util.LogWarn("收到信号 %v，正在停止...", sig)
		// 收到信号后取消上下文，触发迁移任务停止
		cancel()
	}()

	// 7. 执行迁移任务
	// 传入上下文，支持中途取消；返回迁移结果或错误
	return migration.Run(ctx)
}

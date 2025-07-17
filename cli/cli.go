package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"TGFaqBot/cliutil"
)

// CLI 命令行界面
type CLI struct {
	args []string
}

// NewCLI 创建新的CLI实例
func NewCLI() *CLI {
	return &CLI{
		args: os.Args[1:],
	}
}

// Run 运行CLI
func (c *CLI) Run() {
	if len(c.args) == 0 {
		c.printUsage()
		return
	}

	command := c.args[0]

	switch command {
	case "init", "config":
		c.handleInit()
	case "install":
		c.handleInstall()
	case "uninstall":
		c.handleUninstall()
	case "start":
		c.handleStart()
	case "stop":
		c.handleStop()
	case "restart":
		c.handleRestart()
	case "status":
		c.handleStatus()
	case "version", "-v", "--version":
		c.handleVersion()
	case "help", "-h", "--help":
		c.printUsage()
	default:
		fmt.Printf("未知命令: %s\n\n", command)
		c.printUsage()
		os.Exit(1)
	}
}

// printUsage 打印使用说明
func (c *CLI) printUsage() {
	fmt.Printf(`Telegram FAQ Bot - 智能问答机器人

用法:
  %s <command> [options]

可用命令:
  init, config          生成配置文件
  install              安装为系统服务
  uninstall            卸载系统服务
  start                启动服务
  stop                 停止服务
  restart              重启服务
  status               查看服务状态
  version              显示版本信息
  help                 显示帮助信息

示例:
  %s init                    # 生成默认配置文件
  %s init --output custom.json  # 生成配置文件到指定路径
  %s install --name mybot    # 安装为服务，指定服务名称
  %s start                   # 启动服务

更多信息请访问: https://github.com/HsukqiLee/telegram-faq-bot
`, getExecutableName(), getExecutableName(), getExecutableName(), getExecutableName(), getExecutableName())
}

// handleInit 处理配置文件生成
func (c *CLI) handleInit() {
	var outputPath string
	var force bool

	// 解析子命令参数
	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	flagSet.StringVar(&outputPath, "output", "config.json", "配置文件输出路径")
	flagSet.StringVar(&outputPath, "o", "config.json", "配置文件输出路径 (简写)")
	flagSet.BoolVar(&force, "force", false, "强制覆盖已存在的配置文件")
	flagSet.BoolVar(&force, "f", false, "强制覆盖已存在的配置文件 (简写)")

	flagSet.Parse(c.args[1:])

	if err := generateConfigFile(outputPath, force); err != nil {
		fmt.Printf("生成配置文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 配置文件已生成: %s\n", outputPath)
	fmt.Println("📝 请编辑配置文件并填入必要的参数（如 Telegram Bot Token）")
}

// handleInstall 处理服务安装
func (c *CLI) handleInstall() {
	var serviceName string
	var description string

	flagSet := flag.NewFlagSet("install", flag.ExitOnError)
	flagSet.StringVar(&serviceName, "name", "telegram-faq-bot", "服务名称")
	flagSet.StringVar(&serviceName, "n", "telegram-faq-bot", "服务名称 (简写)")
	flagSet.StringVar(&description, "description", "Telegram FAQ Bot Service", "服务描述")
	flagSet.StringVar(&description, "d", "Telegram FAQ Bot Service", "服务描述 (简写)")

	flagSet.Parse(c.args[1:])

	if err := installService(serviceName, description); err != nil {
		fmt.Printf("安装服务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 服务已安装: %s\n", serviceName)
	fmt.Printf("🚀 使用 '%s start' 启动服务\n", getExecutableName())
}

// handleUninstall 处理服务卸载
func (c *CLI) handleUninstall() {
	var serviceName string

	flagSet := flag.NewFlagSet("uninstall", flag.ExitOnError)
	cliutil.SetupServiceFlagSet(flagSet, &serviceName)
	flagSet.Parse(c.args[1:])

	if err := uninstallService(serviceName); err != nil {
		fmt.Printf("卸载服务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 服务已卸载: %s\n", serviceName)
}

// handleStart 处理服务启动
func (c *CLI) handleStart() {
	var serviceName string

	flagSet := flag.NewFlagSet("start", flag.ExitOnError)
	cliutil.SetupServiceFlagSet(flagSet, &serviceName)
	flagSet.Parse(c.args[1:])

	if err := startService(serviceName); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 服务已启动: %s\n", serviceName)
}

// handleStop 处理服务停止
func (c *CLI) handleStop() {
	var serviceName string

	flagSet := flag.NewFlagSet("stop", flag.ExitOnError)
	cliutil.SetupServiceFlagSet(flagSet, &serviceName)
	flagSet.Parse(c.args[1:])

	if err := stopService(serviceName); err != nil {
		fmt.Printf("停止服务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 服务已停止: %s\n", serviceName)
}

// handleRestart 处理服务重启
func (c *CLI) handleRestart() {
	var serviceName string

	flagSet := flag.NewFlagSet("restart", flag.ExitOnError)
	cliutil.SetupServiceFlagSet(flagSet, &serviceName)
	flagSet.Parse(c.args[1:])

	if err := restartService(serviceName); err != nil {
		fmt.Printf("重启服务失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 服务已重启: %s\n", serviceName)
}

// handleStatus 处理服务状态查询
func (c *CLI) handleStatus() {
	var serviceName string

	flagSet := flag.NewFlagSet("status", flag.ExitOnError)
	cliutil.SetupServiceFlagSet(flagSet, &serviceName)
	flagSet.Parse(c.args[1:])

	status, err := getServiceStatus(serviceName)
	if err != nil {
		fmt.Printf("查询服务状态失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📊 服务状态: %s\n", status)
}

// handleVersion 处理版本信息
func (c *CLI) handleVersion() {
	fmt.Printf("Telegram FAQ Bot v1.0.0\n")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// getExecutableName 获取可执行文件名
func getExecutableName() string {
	execPath, err := os.Executable()
	if err != nil {
		return "telegram-faq-bot"
	}

	name := filepath.Base(execPath)
	// 移除 .exe 后缀（Windows）
	if runtime.GOOS == "windows" && strings.HasSuffix(name, ".exe") {
		name = strings.TrimSuffix(name, ".exe")
	}

	return name
}

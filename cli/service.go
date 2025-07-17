package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// installService 安装系统服务
func installService(serviceName, description string) error {
	switch runtime.GOOS {
	case "windows":
		return installWindowsService(serviceName, description)
	case "linux":
		return installLinuxService(serviceName, description)
	case "darwin":
		return installMacService(serviceName)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// uninstallService 卸载系统服务
func uninstallService(serviceName string) error {
	switch runtime.GOOS {
	case "windows":
		return uninstallWindowsService(serviceName)
	case "linux":
		return uninstallLinuxService(serviceName)
	case "darwin":
		return uninstallMacService(serviceName)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// startService 启动服务
func startService(serviceName string) error {
	switch runtime.GOOS {
	case "windows":
		return runCommand("sc", "start", serviceName)
	case "linux":
		return runCommand("sudo", "systemctl", "start", serviceName)
	case "darwin":
		return runCommand("sudo", "launchctl", "load", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName))
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// stopService 停止服务
func stopService(serviceName string) error {
	switch runtime.GOOS {
	case "windows":
		return runCommand("sc", "stop", serviceName)
	case "linux":
		return runCommand("sudo", "systemctl", "stop", serviceName)
	case "darwin":
		return runCommand("sudo", "launchctl", "unload", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName))
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// restartService 重启服务
func restartService(serviceName string) error {
	if err := stopService(serviceName); err != nil {
		// 如果停止失败，尝试继续启动
		fmt.Printf("警告: 停止服务失败: %v\n", err)
	}
	return startService(serviceName)
}

// getServiceStatus 获取服务状态
func getServiceStatus(serviceName string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getWindowsServiceStatus(serviceName)
	case "linux":
		return getLinuxServiceStatus(serviceName)
	case "darwin":
		return getMacServiceStatus(serviceName)
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// Windows 服务管理
func installWindowsService(serviceName, description string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 使用 sc 命令安装服务
	return runCommand("sc", "create", serviceName,
		"binPath=", fmt.Sprintf("\"%s\"", execPath),
		"DisplayName=", description,
		"start=", "auto")
}

func uninstallWindowsService(serviceName string) error {
	return runCommand("sc", "delete", serviceName)
}

func getWindowsServiceStatus(serviceName string) (string, error) {
	cmd := exec.Command("sc", "query", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "未安装", nil
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "RUNNING") {
		return "运行中", nil
	} else if strings.Contains(outputStr, "STOPPED") {
		return "已停止", nil
	} else if strings.Contains(outputStr, "PENDING") {
		return "状态变更中", nil
	}

	return "未知状态", nil
}

// Linux 服务管理
func installLinuxService(serviceName, description string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
User=root
ExecStart=%s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, description, execPath)

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	// 写入服务文件
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("写入服务文件失败: %v", err)
	}

	// 重新加载 systemd
	if err := runCommand("sudo", "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("重新加载 systemd 失败: %v", err)
	}

	// 启用服务
	return runCommand("sudo", "systemctl", "enable", serviceName)
}

func uninstallLinuxService(serviceName string) error {
	// 停止并禁用服务
	runCommand("sudo", "systemctl", "stop", serviceName)
	runCommand("sudo", "systemctl", "disable", serviceName)

	// 删除服务文件
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除服务文件失败: %v", err)
	}

	// 重新加载 systemd
	return runCommand("sudo", "systemctl", "daemon-reload")
}

func getLinuxServiceStatus(serviceName string) (string, error) {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "未安装", nil
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "active":
		return "运行中", nil
	case "inactive":
		return "已停止", nil
	case "failed":
		return "失败", nil
	default:
		return status, nil
	}
}

// macOS 服务管理
func installMacService(serviceName string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	   <key>Label</key>
	   <string>%s</string>
	   <key>ProgramArguments</key>
	   <array>
			   <string>%s</string>
	   </array>
	   <key>RunAtLoad</key>
	   <true/>
	   <key>KeepAlive</key>
	   <true/>
	   <key>StandardOutPath</key>
	   <string>/var/log/%s.log</string>
	   <key>StandardErrorPath</key>
	   <string>/var/log/%s.error.log</string>
</dict>
</plist>
`, serviceName, execPath, serviceName, serviceName)

	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)

	// 写入 plist 文件
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("写入 plist 文件失败: %v", err)
	}

	// 加载服务
	return runCommand("sudo", "launchctl", "load", plistPath)
}

func uninstallMacService(serviceName string) error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)

	// 卸载服务
	runCommand("sudo", "launchctl", "unload", plistPath)

	// 删除 plist 文件
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 plist 文件失败: %v", err)
	}

	return nil
}

func getMacServiceStatus(serviceName string) (string, error) {
	cmd := exec.Command("launchctl", "list", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "未安装", nil
	}

	if strings.Contains(string(output), serviceName) {
		return "运行中", nil
	}

	return "已停止", nil
}

// runCommand 运行系统命令
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("命令执行失败: %s\n输出: %s", err, string(output))
	}
	return nil
}

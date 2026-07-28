package skillhubd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const systemdUnitName = "skill-hubd.service"

// ServiceConfig describes the daemon arguments persisted in a user service.
type ServiceConfig struct {
	Host      string
	Port      int
	SecretKey string
	Start     bool
}

// ServiceStatus is the user-facing state of the managed daemon.
type ServiceStatus struct {
	Manager   string `json:"manager"`
	Unit      string `json:"unit"`
	UnitPath  string `json:"unit_path"`
	Installed bool   `json:"installed"`
	State     string `json:"state"`
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

// ServiceManager owns only operating-system service configuration for the
// skill-hubd process. It does not own daemon business state or HTTP lifecycle.
type ServiceManager struct {
	goos       string
	configDir  func() (string, error)
	executable func() (string, error)
	run        commandRunner
}

// NewServiceManager returns the manager for the current user and platform.
func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		goos:       runtime.GOOS,
		configDir:  os.UserConfigDir,
		executable: os.Executable,
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
	}
}

// RunServiceCommand executes the skill-hubd service command family.
func RunServiceCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	return runServiceCommand(ctx, NewServiceManager(), args, stdout, stderr)
}

func runServiceCommand(ctx context.Context, manager *ServiceManager, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		writeServiceUsage(stderr)
		return errors.New("缺少 service 子命令")
	}

	switch args[0] {
	case "install":
		if isServiceHelp(args[1:]) {
			writeServiceUsage(stdout)
			return nil
		}
		config, err := parseServiceInstallArgs(args[1:], stderr)
		if err != nil {
			return err
		}
		status, err := manager.Install(ctx, config)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "已安装 %s (%s)，状态: %s\n", status.Unit, status.Manager, status.State)
		return nil
	case "start", "stop", "restart", "uninstall":
		if isServiceHelp(args[1:]) {
			writeServiceUsage(stdout)
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("service %s 不接受参数", args[0])
		}
		var err error
		switch args[0] {
		case "start":
			err = manager.Start(ctx)
		case "stop":
			err = manager.Stop(ctx)
		case "restart":
			err = manager.Restart(ctx)
		case "uninstall":
			err = manager.Uninstall(ctx)
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "skill-hubd service %s 完成\n", args[0])
		return nil
	case "status":
		if isServiceHelp(args[1:]) {
			writeServiceUsage(stdout)
			return nil
		}
		jsonOutput, err := parseServiceStatusArgs(args[1:], stderr)
		if err != nil {
			return err
		}
		status, err := manager.Status(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return json.NewEncoder(stdout).Encode(status)
		}
		fmt.Fprintf(stdout, "服务管理器: %s\n服务单元: %s\n单元文件: %s\n已安装: %t\n状态: %s\n", status.Manager, status.Unit, status.UnitPath, status.Installed, status.State)
		return nil
	case "help", "--help", "-h":
		writeServiceUsage(stdout)
		return nil
	default:
		writeServiceUsage(stderr)
		return fmt.Errorf("未知 service 子命令: %s", args[0])
	}
}

func isServiceHelp(args []string) bool {
	return len(args) == 1 && (args[0] == "--help" || args[0] == "-h")
}

func parseServiceInstallArgs(args []string, stderr io.Writer) (ServiceConfig, error) {
	config := ServiceConfig{Host: "127.0.0.1", Port: 5525, Start: true}
	for len(args) > 0 {
		switch args[0] {
		case "--host":
			if len(args) < 2 {
				return config, errors.New("--host 缺少值")
			}
			config.Host, args = args[1], args[2:]
		case "--port":
			if len(args) < 2 {
				return config, errors.New("--port 缺少值")
			}
			port, err := strconv.Atoi(args[1])
			if err != nil {
				return config, fmt.Errorf("--port 必须是整数: %w", err)
			}
			config.Port, args = port, args[2:]
		case "--secret-key":
			if len(args) < 2 {
				return config, errors.New("--secret-key 缺少值")
			}
			config.SecretKey, args = args[1], args[2:]
		case "--no-start":
			config.Start, args = false, args[1:]
		default:
			return config, fmt.Errorf("未知 install 参数: %s", args[0])
		}
	}
	if strings.TrimSpace(config.Host) == "" {
		return config, errors.New("--host 不能为空")
	}
	if config.Port < 1 || config.Port > 65535 {
		return config, errors.New("--port 必须在 1-65535 之间")
	}
	return config, nil
}

func parseServiceStatusArgs(args []string, stderr io.Writer) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && args[0] == "--json" {
		return true, nil
	}
	return false, fmt.Errorf("未知 status 参数: %s", strings.Join(args, " "))
}

func writeServiceUsage(writer io.Writer) {
	fmt.Fprintln(writer, "用法: skill-hubd service <install|start|stop|restart|status|uninstall> [flags]")
	fmt.Fprintln(writer, "  install [--host HOST] [--port PORT] [--secret-key KEY] [--no-start]")
	fmt.Fprintln(writer, "  status [--json]")
}

// Install creates and optionally starts the current user's systemd service.
func (m *ServiceManager) Install(ctx context.Context, config ServiceConfig) (ServiceStatus, error) {
	if err := m.requireSystemd(); err != nil {
		return ServiceStatus{}, err
	}
	unitPath, err := m.unitPath()
	if err != nil {
		return ServiceStatus{}, err
	}
	executable, err := m.executable()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("获取 skill-hubd 可执行路径失败: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("解析 skill-hubd 可执行路径失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0700); err != nil {
		return ServiceStatus{}, fmt.Errorf("创建 systemd 用户配置目录失败: %w", err)
	}
	if err := writePrivateFile(unitPath, []byte(systemdUnit(executable, config))); err != nil {
		return ServiceStatus{}, fmt.Errorf("写入 systemd 服务单元失败: %w", err)
	}
	if err := m.systemctl(ctx, "daemon-reload"); err != nil {
		return ServiceStatus{}, err
	}
	if config.Start {
		if err := m.systemctl(ctx, "enable", "--now", systemdUnitName); err != nil {
			return ServiceStatus{}, err
		}
		return ServiceStatus{Manager: "systemd-user", Unit: systemdUnitName, UnitPath: unitPath, Installed: true, State: "active"}, nil
	}
	if err := m.systemctl(ctx, "enable", systemdUnitName); err != nil {
		return ServiceStatus{}, err
	}
	return ServiceStatus{Manager: "systemd-user", Unit: systemdUnitName, UnitPath: unitPath, Installed: true, State: "inactive"}, nil
}

func (m *ServiceManager) Start(ctx context.Context) error   { return m.control(ctx, "start") }
func (m *ServiceManager) Stop(ctx context.Context) error    { return m.control(ctx, "stop") }
func (m *ServiceManager) Restart(ctx context.Context) error { return m.control(ctx, "restart") }

func (m *ServiceManager) control(ctx context.Context, action string) error {
	if err := m.requireSystemd(); err != nil {
		return err
	}
	unitPath, err := m.unitPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(unitPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("skill-hubd 服务尚未安装，请先执行 'skill-hubd service install'")
		}
		return fmt.Errorf("读取 systemd 服务单元失败: %w", err)
	}
	return m.systemctl(ctx, action, systemdUnitName)
}

// Status returns installed and active state without treating an inactive unit as an error.
func (m *ServiceManager) Status(ctx context.Context) (ServiceStatus, error) {
	if err := m.requireSystemd(); err != nil {
		return ServiceStatus{}, err
	}
	unitPath, err := m.unitPath()
	if err != nil {
		return ServiceStatus{}, err
	}
	status := ServiceStatus{Manager: "systemd-user", Unit: systemdUnitName, UnitPath: unitPath, State: "not_installed"}
	if _, err := os.Stat(unitPath); err != nil {
		if os.IsNotExist(err) {
			return status, nil
		}
		return ServiceStatus{}, fmt.Errorf("读取 systemd 服务单元失败: %w", err)
	}
	status.Installed = true
	if _, err := m.run(ctx, "systemctl", "--user", "is-active", "--quiet", systemdUnitName); err == nil {
		status.State = "active"
	} else {
		status.State = "inactive"
	}
	return status, nil
}

// Uninstall stops, disables and removes the current user's unit.
func (m *ServiceManager) Uninstall(ctx context.Context) error {
	if err := m.requireSystemd(); err != nil {
		return err
	}
	unitPath, err := m.unitPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(unitPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("skill-hubd 服务尚未安装")
		}
		return fmt.Errorf("读取 systemd 服务单元失败: %w", err)
	}
	if err := m.systemctl(ctx, "disable", "--now", systemdUnitName); err != nil {
		return err
	}
	if err := os.Remove(unitPath); err != nil {
		return fmt.Errorf("删除 systemd 服务单元失败: %w", err)
	}
	return m.systemctl(ctx, "daemon-reload")
}

func (m *ServiceManager) requireSystemd() error {
	if m == nil || m.goos != "linux" {
		return errors.New("skill-hubd service 当前仅支持 Linux systemd 用户服务；请使用平台服务管理器启动 skill-hubd")
	}
	return nil
}

func (m *ServiceManager) unitPath() (string, error) {
	configDir, err := m.configDir()
	if err != nil {
		return "", fmt.Errorf("获取用户配置目录失败: %w", err)
	}
	return filepath.Join(configDir, "systemd", "user", systemdUnitName), nil
}

func (m *ServiceManager) systemctl(ctx context.Context, args ...string) error {
	output, err := m.run(ctx, "systemctl", append([]string{"--user"}, args...)...)
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return fmt.Errorf("执行 systemctl --user %s 失败: %s", strings.Join(args, " "), message)
}

func systemdUnit(executable string, config ServiceConfig) string {
	args := []string{systemdQuote(executable), "--host", systemdQuote(config.Host), "--port", strconv.Itoa(config.Port)}
	if config.SecretKey != "" {
		args = append(args, "--secret-key", systemdQuote(config.SecretKey))
	}
	return "[Unit]\nDescription=skill-hubd local daemon\nAfter=network-online.target\n\n[Service]\nType=simple\nExecStart=" + strings.Join(args, " ") + "\nRestart=on-failure\nRestartSec=3\n\n[Install]\nWantedBy=default.target\n"
}

func systemdQuote(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\r", "\\r", "\t", "\\t")
	return "\"" + replacer.Replace(value) + "\""
}

func writePrivateFile(path string, content []byte) error {
	temporary := path + ".new"
	if err := os.WriteFile(temporary, content, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

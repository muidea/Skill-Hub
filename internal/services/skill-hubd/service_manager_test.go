package skillhubd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceManagerInstallCreatesPrivateSystemdUnit(t *testing.T) {
	manager, calls := newTestServiceManager(t)
	status, err := manager.Install(context.Background(), ServiceConfig{
		Host:      "127.0.0.1",
		Port:      6600,
		SecretKey: "write secret",
		Start:     true,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if status.State != "active" || !status.Installed {
		t.Fatalf("Install() status = %+v", status)
	}
	content, err := os.ReadFile(status.UnitPath)
	if err != nil {
		t.Fatalf("read unit: %v", err)
	}
	unit := string(content)
	for _, want := range []string{
		`ExecStart="/opt/skill-hubd" --host "127.0.0.1" --port 6600 --secret-key "write secret"`,
		"Restart=on-failure",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit missing %q:\n%s", want, unit)
		}
	}
	info, err := os.Stat(status.UnitPath)
	if err != nil {
		t.Fatalf("stat unit: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("unit permissions = %o, want 600", info.Mode().Perm())
	}
	if got, want := *calls, []string{
		"systemctl --user daemon-reload",
		"systemctl --user enable --now skill-hubd.service",
	}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("systemctl calls = %v, want %v", got, want)
	}
}

func TestServiceManagerStatusDistinguishesNotInstalledAndInactive(t *testing.T) {
	manager, _ := newTestServiceManager(t)
	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() before install: %v", err)
	}
	if status.Installed || status.State != "not_installed" {
		t.Fatalf("status before install = %+v", status)
	}

	unitPath, err := manager.unitPath()
	if err != nil {
		t.Fatalf("unitPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0700); err != nil {
		t.Fatalf("mkdir unit dir: %v", err)
	}
	if err := os.WriteFile(unitPath, []byte("[Service]\n"), 0600); err != nil {
		t.Fatalf("write unit: %v", err)
	}
	manager.run = func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("inactive")
	}
	status, err = manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() after install: %v", err)
	}
	if !status.Installed || status.State != "inactive" {
		t.Fatalf("status after install = %+v", status)
	}
}

func TestServiceManagerControlAndUninstall(t *testing.T) {
	manager, calls := newTestServiceManager(t)
	if _, err := manager.Install(context.Background(), ServiceConfig{Host: "127.0.0.1", Port: 5525, Start: false}); err != nil {
		t.Fatalf("Install(): %v", err)
	}
	*calls = nil
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Start(): %v", err)
	}
	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("Stop(): %v", err)
	}
	if err := manager.Restart(context.Background()); err != nil {
		t.Fatalf("Restart(): %v", err)
	}
	if err := manager.Uninstall(context.Background()); err != nil {
		t.Fatalf("Uninstall(): %v", err)
	}
	for _, want := range []string{
		"systemctl --user start skill-hubd.service",
		"systemctl --user stop skill-hubd.service",
		"systemctl --user restart skill-hubd.service",
		"systemctl --user disable --now skill-hubd.service",
		"systemctl --user daemon-reload",
	} {
		if !strings.Contains(strings.Join(*calls, "\n"), want) {
			t.Fatalf("missing call %q in %v", want, *calls)
		}
	}
	unitPath, _ := manager.unitPath()
	if _, err := os.Stat(unitPath); !os.IsNotExist(err) {
		t.Fatalf("unit still exists after uninstall: %v", err)
	}
}

func TestServiceManagerRejectsUnsupportedPlatform(t *testing.T) {
	manager, _ := newTestServiceManager(t)
	manager.goos = "darwin"
	if _, err := manager.Status(context.Background()); err == nil || !strings.Contains(err.Error(), "仅支持 Linux") {
		t.Fatalf("Status() error = %v, want unsupported platform", err)
	}
}

func TestRunServiceCommandRendersJSONStatus(t *testing.T) {
	manager, _ := newTestServiceManager(t)
	var stdout strings.Builder
	if err := runServiceCommand(context.Background(), manager, []string{"status", "--json"}, &stdout, &strings.Builder{}); err != nil {
		t.Fatalf("runServiceCommand(): %v", err)
	}
	if !strings.Contains(stdout.String(), `"state":"not_installed"`) {
		t.Fatalf("unexpected json output: %s", stdout.String())
	}
}

func TestRunServiceCommandRejectsUnexpectedControlArguments(t *testing.T) {
	manager, _ := newTestServiceManager(t)
	err := runServiceCommand(context.Background(), manager, []string{"start", "unexpected"}, &strings.Builder{}, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "不接受参数") {
		t.Fatalf("runServiceCommand() error = %v, want argument rejection", err)
	}
}

func newTestServiceManager(t *testing.T) (*ServiceManager, *[]string) {
	t.Helper()
	var calls []string
	configDir := filepath.Join(t.TempDir(), ".config")
	return &ServiceManager{
		goos: "linux",
		configDir: func() (string, error) {
			return configDir, nil
		},
		executable: func() (string, error) { return "/opt/skill-hubd", nil },
		run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
			return nil, nil
		},
	}, &calls
}

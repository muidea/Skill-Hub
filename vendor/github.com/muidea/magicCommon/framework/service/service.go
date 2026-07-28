package service

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/framework/configuration"
	"github.com/muidea/magicCommon/framework/health"
	"github.com/muidea/magicCommon/framework/plugin/initiator"
	"github.com/muidea/magicCommon/framework/plugin/module"
	"github.com/muidea/magicCommon/task"
)

type Service interface {
	Startup(ctx context.Context, serviceName string, eventHub event.Hub, backgroundRoutine task.BackgroundRoutine) *cd.Error
	Run(ctx context.Context) *cd.Error
	Shutdown(ctx context.Context)
}

func DefaultService() Service {
	return &defaultService{}
}

type defaultService struct {
	serviceName string
}

func (s *defaultService) Startup(ctx context.Context, serviceName string, eventHub event.Hub, backgroundRoutine task.BackgroundRoutine) (ret *cd.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.serviceName = serviceName
	manager := health.DefaultManager()
	manager.SetService(serviceName)
	manager.MarkStarting()

	ret = initiator.Setup(ctx, eventHub, backgroundRoutine)
	if ret != nil {
		manager.MarkFailed(ret)
		slog.Error("service startup failed", "service", s.serviceName, "stage", "initiator.setup", "error", ret)
		return
	}

	dependencies, depErr := loadConfiguredDependencies()
	if depErr != nil {
		ret = cd.NewError(cd.Unexpected, depErr.Error())
		manager.MarkFailed(ret)
		initiator.Teardown(ctx)
		slog.Error("service startup failed", "service", s.serviceName, "stage", "dependency.config", "error", ret)
		return
	}
	ret = manager.CheckDependencies(ctx, dependencies)
	if ret != nil {
		manager.MarkFailed(ret)
		initiator.Teardown(ctx)
		slog.Error("service startup failed", "service", s.serviceName, "stage", "dependency.check", "error", ret)
		return
	}

	ret = module.Setup(ctx, eventHub, backgroundRoutine)
	if ret != nil {
		manager.MarkFailed(ret)
		module.Teardown(ctx)
		initiator.Teardown(ctx)
		slog.Error("service startup failed", "service", s.serviceName, "stage", "module.setup", "error", ret)
		return
	}

	//slog.Info("s.serviceName startup success", "field", s.serviceName)
	return
}

func (s *defaultService) Run(ctx context.Context) (ret *cd.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	manager := health.DefaultManager()
	defer func() {
		if errInfo := recover(); errInfo != nil {
			manager.MarkFailed(cd.NewError(cd.Unexpected, "service run panicked"))
			slog.Error("service run panicked", "service", s.serviceName, "panic", errInfo)
		}
	}()

	ret = initiator.Run(ctx)
	if ret != nil {
		manager.MarkFailed(ret)
		initiator.Teardown(ctx)
		slog.Error("service run failed", "service", s.serviceName, "stage", "initiator.run", "error", ret)
		return
	}
	ret = module.Run(ctx)
	if ret != nil {
		manager.MarkFailed(ret)
		module.Teardown(ctx)
		initiator.Teardown(ctx)
		slog.Error("service run failed", "service", s.serviceName, "stage", "module.run", "error", ret)
		return
	}

	manager.MarkReady()

	//slog.Info("s.serviceName running!", "field", s.serviceName)
	return
}

func loadConfiguredDependencies() ([]health.Dependency, error) {
	configManager := configuration.GetDefaultConfigManager()
	if configManager == nil {
		return nil, nil
	}

	exported, err := configuration.ExportAllConfigs()
	if err != nil {
		return nil, err
	}

	applicationCfg, ok := exported["application"].(map[string]any)
	if !ok {
		return nil, nil
	}

	dependenciesValue, exists := applicationCfg["serviceDependencies"]
	if !exists {
		return nil, nil
	}

	dependenciesMap, ok := dependenciesValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("serviceDependencies must be a map")
	}

	jsonBytes, err := json.Marshal(dependenciesMap)
	if err != nil {
		return nil, err
	}

	decoded := map[string]health.Dependency{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		return nil, err
	}

	ret := make([]health.Dependency, 0, len(decoded))
	for name, dep := range decoded {
		if dep.Kind == "" {
			dep.Kind = health.RequiredDependency
		}
		dep.Name = name
		ret = append(ret, dep)
	}

	return ret, nil
}

func (s *defaultService) Shutdown(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		if errInfo := recover(); errInfo != nil {
			slog.Error("service shutdown panicked", "service", s.serviceName, "panic", errInfo)
		}
	}()

	module.Teardown(ctx)
	initiator.Teardown(ctx)
	//slog.Info("s.serviceName shutdown success", "field", s.serviceName)
}

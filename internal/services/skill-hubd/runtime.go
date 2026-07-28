package skillhubd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	registrycommon "github.com/muidea/skill-hub/internal/initiators/routeregistry/pkg/common"
	"github.com/muidea/skill-hub/internal/pkg/skillhubdbootstrap"

	frameworkapplication "github.com/muidea/magicCommon/framework/application"
	"github.com/muidea/magicCommon/framework/plugin/initiator"
	frameworkservice "github.com/muidea/magicCommon/framework/service"
)

// Runtime is the daemon process lifecycle shell. Business behavior remains in
// framework Modules; this type only boots, activates and stops the process.
type Runtime struct {
	application frameworkapplication.Application
	config      Config
}

func NewRuntime(config Config) *Runtime {
	return &Runtime{
		config: config,
		application: frameworkapplication.NewApplication(frameworkapplication.Options{
			ConfigDir:   filepath.Clean("."),
			ServiceName: "skill-hubd",
		}),
	}
}

func Run(ctx context.Context, config Config) error {
	runtime := NewRuntime(config)
	if err := runtime.Startup(ctx); err != nil {
		return err
	}
	defer runtime.Shutdown(context.Background())
	return runtime.Run(ctx)
}

func (r *Runtime) Startup(ctx context.Context) error {
	if r == nil || r.application == nil {
		return errors.New("skill-hubd runtime is not initialized")
	}
	skillhubdbootstrap.Configure(skillhubdbootstrap.Config(r.config))
	if err := r.application.Startup(ctx, frameworkservice.DefaultService()); err != nil {
		return fmt.Errorf("start skill-hubd framework: %w", err)
	}
	return nil
}

func (r *Runtime) Run(ctx context.Context) error {
	if r == nil || r.application == nil {
		return errors.New("skill-hubd runtime is not initialized")
	}
	if err := r.application.Run(ctx); err != nil {
		return fmt.Errorf("run skill-hubd framework: %w", err)
	}
	gateway, err := initiator.GetEntity(registrycommon.RouteRegistryInitiator, registrycommon.GatewayRuntimeHelper(nil))
	if err != nil {
		return fmt.Errorf("get HTTP route registry: %s", err.Message)
	}
	if err := gateway.Start(); err != nil {
		return fmt.Errorf("start HTTP route registry: %w", err)
	}
	select {
	case err := <-gateway.Done():
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return nil
	}
}

func (r *Runtime) Shutdown(ctx context.Context) {
	if r != nil && r.application != nil {
		r.application.Shutdown(ctx)
	}
}

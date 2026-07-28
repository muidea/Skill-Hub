package service

import (
	"context"
	"fmt"
	"log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type LifecycleService interface {
	Startup(ctx context.Context) error
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

func AdaptLifecycle(name string, svc LifecycleService) Service {
	return &lifecycleAdapter{
		name: name,
		svc:  svc,
	}
}

type lifecycleAdapter struct {
	name string
	svc  LifecycleService
}

func (s *lifecycleAdapter) Startup(ctx context.Context, serviceName string, _ event.Hub, _ task.BackgroundRoutine) *cd.Error {
	if s.svc == nil {
		return cd.NewError(cd.IllegalParam, "lifecycle service is nil")
	}
	if serviceName != "" {
		s.name = serviceName
	}
	return wrapLifecycleError("lifecycle startup failed", s.svc.Startup(ctx))
}

func (s *lifecycleAdapter) Run(ctx context.Context) *cd.Error {
	if s.svc == nil {
		return cd.NewError(cd.IllegalParam, "lifecycle service is nil")
	}
	return wrapLifecycleError("lifecycle run failed", s.svc.Run(ctx))
}

func (s *lifecycleAdapter) Shutdown(ctx context.Context) {
	if s.svc == nil {
		return
	}
	if err := s.svc.Shutdown(ctx); err != nil {
		slog.Error("lifecycle shutdown failed", "service", s.name, "error", err)
	}
}

func wrapLifecycleError(message string, err error) *cd.Error {
	if err == nil {
		return nil
	}
	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr
	}
	return cd.NewError(cd.Unexpected, fmt.Sprintf("%s: %v", message, err))
}

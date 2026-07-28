package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	cd "github.com/muidea/magicCommon/def"
)

func HoldService() Service {
	return &holdService{
		defaultService: defaultService{},
	}
}

type holdService struct {
	defaultService
}

func (s *holdService) Run(ctx context.Context) (err *cd.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	err = s.defaultService.Run(ctx)
	if err != nil {
		return
	}

	sigChan := make(chan os.Signal, 1)
	defer close(sigChan)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		slog.Warn("service received shutdown signal", "service", s.serviceName, "signal", sig)
	case <-ctx.Done():
		slog.Warn("service context done", "service", s.serviceName, "err", ctx.Err())
	}
	return
}

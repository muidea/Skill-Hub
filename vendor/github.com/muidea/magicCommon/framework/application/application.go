package application

import (
	"context"
	"os"
	"strconv"
	"sync"

	_ "log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"

	"github.com/muidea/magicCommon/framework/configuration"
	"github.com/muidea/magicCommon/framework/health"
	"github.com/muidea/magicCommon/framework/service"
)

var defaultBackTaskQueueSize = 10000
var defaultEventHubQueueSize = 500000

func init() {
	taskQueueSize, taskQueueOK := os.LookupEnv("BG_TASK_QUEUE_SIZE")
	if taskQueueOK {
		iVal, iErr := strconv.Atoi(taskQueueSize)
		if iErr == nil && iVal > 0 {
			defaultBackTaskQueueSize = iVal
		}
	}

	eventQueueSize, eventQueueOK := os.LookupEnv("HUB_EVENT_QUEUE_SIZE")
	if eventQueueOK {
		iVal, iErr := strconv.Atoi(eventQueueSize)
		if iErr == nil && iVal > 0 {
			defaultEventHubQueueSize = iVal
		}
	}
}

type Application interface {
	Startup(ctx context.Context, service service.Service) *cd.Error
	Run(ctx context.Context) *cd.Error
	Shutdown(ctx context.Context)
	EventHub() event.Hub
	BackgroundRoutine() task.BackgroundRoutine
}

type State string

const (
	StateNew      State = "new"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateFailed   State = "failed"
	StateShutdown State = "shutdown"
)

type RuntimeOwnership struct {
	EventHub          bool
	BackgroundRoutine bool
}

type Options struct {
	ConfigDir           string
	ServiceName         string
	EventHubQueueSize   int
	BackgroundQueueSize int
	EventHub            event.Hub
	BackgroundRoutine   task.BackgroundRoutine
	Ownership           RuntimeOwnership
}

var application Application
var applicationOnce sync.Once

func Startup(ctx context.Context, service service.Service) *cd.Error {
	return Get().Startup(ctx, service)
}

func StartupWithOptions(ctx context.Context, service service.Service, opts Options) *cd.Error {
	if app, ok := Get().(*appImpl); ok {
		return app.startupWithOptions(ctx, service, opts, true)
	}
	return cd.NewError(cd.Unexpected, "default application implementation is unavailable")
}

func Run(ctx context.Context) *cd.Error {
	return Get().Run(ctx)
}

func Shutdown(ctx context.Context) {
	Get().Shutdown(ctx)
}

func Get() Application {
	applicationOnce.Do(func() {
		application = newAppImpl(Options{})
	})

	return application
}

func NewApplication(opts Options) Application {
	return newAppImpl(opts)
}

// ResetForTesting resets the application singleton for testing purposes
// This should only be used in tests
func ResetForTesting() {
	_ = configuration.CloseConfigManager()
	application = nil
	applicationOnce = sync.Once{}
	health.ResetDefaultManager()
}

type appImpl struct {
	mu                sync.Mutex
	opts              Options
	ownership         RuntimeOwnership
	state             State
	backgroundRoutine task.BackgroundRoutine
	eventHub          event.Hub
	service           service.Service
}

func newAppImpl(opts Options) *appImpl {
	ret := &appImpl{
		opts:  opts,
		state: StateNew,
	}
	ret.resetRuntimeLocked()
	return ret
}

func (s *appImpl) Startup(ctx context.Context, service service.Service) *cd.Error {
	return s.startupWithOptions(ctx, service, s.opts, false)
}

func (s *appImpl) startupWithOptions(ctx context.Context, svc service.Service, opts Options, replaceRuntime bool) *cd.Error {
	if ctx == nil {
		ctx = context.Background()
	}
	if svc == nil {
		return cd.NewError(cd.IllegalParam, "service is nil")
	}

	s.mu.Lock()
	if s.state != StateNew && s.state != StateShutdown {
		state := s.state
		s.mu.Unlock()
		return cd.NewError(cd.IllegalParam, "application startup is only allowed from new or shutdown state, current state:"+string(state))
	}
	var oldHub event.Hub
	var oldBackgroundRoutine task.BackgroundRoutine
	var oldOwnership RuntimeOwnership
	if replaceRuntime {
		oldHub = s.eventHub
		oldBackgroundRoutine = s.backgroundRoutine
		oldOwnership = s.ownership
		s.opts = opts
		s.resetRuntimeLocked()
	}
	hub := s.eventHub
	backgroundRoutine := s.backgroundRoutine
	s.service = svc
	s.state = StateStarting
	s.mu.Unlock()

	if replaceRuntime {
		shutdownRuntime(ctx, oldHub, oldBackgroundRoutine, oldOwnership)
	}

	err := configuration.InitDefaultConfigManager(opts.ConfigDir)
	if err != nil {
		s.failStartup(ctx)
		return cd.NewError(cd.Unexpected, err.Error())
	}

	nameVal := resolveServiceName(opts.ServiceName)

	startupErr := svc.Startup(ctx, nameVal, hub, backgroundRoutine)
	if startupErr != nil {
		s.failStartup(ctx)
		return startupErr
	}

	s.mu.Lock()
	s.state = StateRunning
	s.mu.Unlock()
	return nil
}

func (s *appImpl) Run(ctx context.Context) *cd.Error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	if s.state != StateRunning || s.service == nil {
		state := s.state
		s.mu.Unlock()
		return cd.NewError(cd.IllegalParam, "application is not running, current state:"+string(state))
	}
	svc := s.service
	s.mu.Unlock()

	return svc.Run(ctx)
}

func (s *appImpl) Shutdown(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	svc := s.service
	hub := s.eventHub
	backgroundRoutine := s.backgroundRoutine
	ownership := s.ownership
	if s.state == StateShutdown {
		s.mu.Unlock()
		return
	}
	s.state = StateShutdown
	s.service = nil
	s.mu.Unlock()

	if svc != nil {
		svc.Shutdown(ctx)
	}
	shutdownRuntime(ctx, hub, backgroundRoutine, ownership)

	_ = configuration.CloseConfigManager()
	health.ResetDefaultManager()

	s.mu.Lock()
	s.resetRuntimeLocked()
	s.mu.Unlock()
}

func (s *appImpl) EventHub() event.Hub {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eventHub
}

func (s *appImpl) BackgroundRoutine() task.BackgroundRoutine {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backgroundRoutine
}

func (s *appImpl) failStartup(ctx context.Context) {
	s.mu.Lock()
	s.state = StateFailed
	svc := s.service
	hub := s.eventHub
	backgroundRoutine := s.backgroundRoutine
	ownership := s.ownership
	s.service = nil
	s.mu.Unlock()

	if svc != nil {
		svc.Shutdown(ctx)
	}
	shutdownRuntime(ctx, hub, backgroundRoutine, ownership)
	_ = configuration.CloseConfigManager()
	health.ResetDefaultManager()
}

func (s *appImpl) resetRuntimeLocked() {
	backgroundQueueSize := s.opts.BackgroundQueueSize
	if backgroundQueueSize <= 0 {
		backgroundQueueSize = defaultBackTaskQueueSize
	}
	eventQueueSize := s.opts.EventHubQueueSize
	if eventQueueSize <= 0 {
		eventQueueSize = defaultEventHubQueueSize
	}

	if s.opts.BackgroundRoutine != nil {
		s.backgroundRoutine = s.opts.BackgroundRoutine
		s.ownership.BackgroundRoutine = s.opts.Ownership.BackgroundRoutine
	} else {
		s.backgroundRoutine = task.NewBackgroundRoutine(backgroundQueueSize)
		s.ownership.BackgroundRoutine = true
	}
	if s.opts.EventHub != nil {
		s.eventHub = s.opts.EventHub
		s.ownership.EventHub = s.opts.Ownership.EventHub
	} else {
		s.eventHub = event.NewHub(eventQueueSize)
		s.ownership.EventHub = true
	}
}

func resolveServiceName(explicitName string) string {
	if explicitName != "" {
		return explicitName
	}
	nameVal, nameErr := configuration.GetString("endpointName")
	if nameErr != nil {
		return "magicFramework"
	}
	return nameVal
}

func shutdownRuntime(ctx context.Context, hub event.Hub, backgroundRoutine task.BackgroundRoutine, ownership RuntimeOwnership) {
	if ownership.BackgroundRoutine && backgroundRoutine != nil {
		backgroundRoutine.Shutdown(ctx)
	}
	if ownership.EventHub && hub != nil {
		hub.Terminate(ctx)
	}
}

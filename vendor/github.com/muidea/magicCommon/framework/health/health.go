package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	cd "github.com/muidea/magicCommon/def"
)

type Status string

const (
	StatusStarting Status = "starting"
	StatusReady    Status = "ready"
	StatusFailed   Status = "failed"
)

type DependencyKind string

const (
	RequiredDependency DependencyKind = "required"
	OptionalDependency DependencyKind = "optional"
)

type Dependency struct {
	Name   string         `json:"-"`
	Kind   DependencyKind `json:"kind"`
	Target string         `json:"target"`
}

func Required(name string) Dependency {
	return Dependency{Name: name, Kind: RequiredDependency}
}

func Optional(name string) Dependency {
	return Dependency{Name: name, Kind: OptionalDependency}
}

type CheckStatus struct {
	Status string `json:"status"`
	Target string `json:"target,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ReadyResponse struct {
	Service   string                 `json:"service"`
	Status    Status                 `json:"status"`
	StartedAt string                 `json:"startedAt,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Checks    map[string]CheckStatus `json:"checks,omitempty"`
}

type Manager struct {
	mu        sync.RWMutex
	service   string
	status    Status
	startedAt time.Time
	errMsg    string
	checks    map[string]CheckStatus
	checkers  map[string]func(context.Context) *cd.Error
	client    *http.Client
}

var (
	defaultManager *Manager
	defaultOnce    sync.Once
)

func DefaultManager() *Manager {
	defaultOnce.Do(func() {
		defaultManager = NewManager()
	})
	return defaultManager
}

func ResetDefaultManager() {
	defaultManager = nil
	defaultOnce = sync.Once{}
}

func NewManager() *Manager {
	return &Manager{
		status:   StatusStarting,
		checks:   map[string]CheckStatus{},
		checkers: map[string]func(context.Context) *cd.Error{},
		client:   &http.Client{Timeout: 3 * time.Second},
	}
}

func (s *Manager) SetService(service string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.service = service
}

func (s *Manager) MarkStarting() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = StatusStarting
	s.errMsg = ""
	s.startedAt = time.Now().UTC()
	s.checks = map[string]CheckStatus{}
}

func (s *Manager) RegisterDependencyChecker(name string, checker func(context.Context) *cd.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

func (s *Manager) MarkReady() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = StatusReady
	s.errMsg = ""
}

func (s *Manager) MarkFailed(err *cd.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = StatusFailed
	if err != nil {
		s.errMsg = err.Error()
	}
}

func (s *Manager) Snapshot() ReadyResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checks := map[string]CheckStatus{}
	for key, val := range s.checks {
		checks[key] = val
	}

	resp := ReadyResponse{
		Service: s.service,
		Status:  s.status,
		Checks:  checks,
	}
	if !s.startedAt.IsZero() {
		resp.StartedAt = s.startedAt.Format(time.RFC3339)
	}
	if s.errMsg != "" {
		resp.Error = s.errMsg
	}

	return resp
}

func (s *Manager) CheckDependencies(ctx context.Context, deps []Dependency) *cd.Error {
	s.mu.Lock()
	s.checks = map[string]CheckStatus{}
	s.mu.Unlock()

	seen := map[string]struct{}{}
	for _, dep := range deps {
		key := dep.Kind.String() + ":" + dep.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		err := s.checkDependency(ctx, dep)
		if err != nil && dep.Kind == RequiredDependency {
			return err
		}
	}

	return nil
}

func (s *Manager) recordCheck(name, target, status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[name] = CheckStatus{
		Status: status,
		Target: target,
		Error:  errMsg,
	}
}

func (s *Manager) checkDependency(ctx context.Context, dep Dependency) *cd.Error {
	s.mu.RLock()
	checker := s.checkers[dep.Name]
	s.mu.RUnlock()

	if checker == nil {
		if dep.Target == "" {
			s.recordCheck(dep.Name, "", "declared", "")
			return nil
		}
		return s.checkTargetDependency(ctx, dep)
	}

	if err := checker(ctx); err != nil {
		s.recordCheck(dep.Name, "", "failed", err.Error())
		return err
	}

	s.recordCheck(dep.Name, "", "ready", "")
	return nil
}

func (s *Manager) checkTargetDependency(ctx context.Context, dep Dependency) *cd.Error {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, dep.Target+"/health/ready", nil)
	if reqErr != nil {
		err := cd.NewError(cd.Unexpected, reqErr.Error())
		s.recordCheck(dep.Name, dep.Target, "failed", err.Error())
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		cdErr := cd.NewError(cd.Unexpected, err.Error())
		s.recordCheck(dep.Name, dep.Target, "failed", cdErr.Error())
		return cdErr
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		cdErr := cd.NewError(cd.Unexpected, fmt.Sprintf("dependency %s not ready, status:%d", dep.Name, resp.StatusCode))
		s.recordCheck(dep.Name, dep.Target, "failed", cdErr.Error())
		return cdErr
	}

	s.recordCheck(dep.Name, dep.Target, "ready", "")
	return nil
}

func (s *Manager) LiveHandler(_ context.Context, res http.ResponseWriter, _ *http.Request) {
	writeJSON(res, http.StatusOK, map[string]string{"status": "live"})
}

func (s *Manager) ReadyHandler(_ context.Context, res http.ResponseWriter, _ *http.Request) {
	snapshot := s.Snapshot()
	statusCode := http.StatusOK
	if snapshot.Status != StatusReady {
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(res, statusCode, snapshot)
}

func writeJSON(res http.ResponseWriter, statusCode int, value any) {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(statusCode)
	_ = json.NewEncoder(res).Encode(value)
}

func (s DependencyKind) String() string {
	return string(s)
}

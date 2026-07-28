// Package routeregistry owns the daemon's single HTTP listener and the
// magicEngine route registry. It has no business state.
package routeregistry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/muidea/skill-hub/internal/initiators/routeregistry/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/skillhubdbootstrap"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/framework/plugin/initiator"
	"github.com/muidea/magicCommon/task"
	enginehttp "github.com/muidea/magicEngine/http"
)

func init() { initiator.Register(New()) }

type routeRegistry struct {
	routes   enginehttp.RouteRegistry
	server   *http.Server
	listener net.Listener
	done     chan error
	mu       sync.Mutex
	started  bool
}

var routeRegistryListen = net.Listen

func New() *routeRegistry { return &routeRegistry{} }

func (r *routeRegistry) ID() string { return common.RouteRegistryInitiator }

func (r *routeRegistry) Setup(_ context.Context, _ event.Hub, _ task.BackgroundRoutine) *cd.Error {
	config, ok := skillhubdbootstrap.Current()
	if !ok {
		return cd.NewError(cd.IllegalParam, "skill-hubd bootstrap is not configured")
	}
	if strings.TrimSpace(config.Host) == "" || config.Port <= 0 || config.Port > 65535 {
		return cd.NewError(cd.IllegalParam, "invalid skill-hubd listen configuration")
	}

	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	listener, err := routeRegistryListen("tcp", address)
	if err != nil {
		return cd.NewError(cd.Unexpected, fmt.Sprintf("listen %s: %v", address, err))
	}

	routes := enginehttp.NewRouteRegistry()
	engineServer := enginehttp.NewHTTPServer()
	engineServer.Bind(routes)
	handler, ok := engineServer.(http.Handler)
	if !ok {
		_ = listener.Close()
		return cd.NewError(cd.Unexpected, "magicEngine HTTP server does not implement http.Handler")
	}

	r.routes = routes
	r.server = &http.Server{
		Addr:              address,
		Handler:           secureLocalHandler(handler, config.Host, config.SecretKey),
		ReadHeaderTimeout: 5 * time.Second,
	}
	r.listener = listener
	r.done = make(chan error, 1)
	return nil
}

// Run only validates its prepared resources. The process service calls Start
// after application Modules have registered every route.
func (r *routeRegistry) Run(context.Context) *cd.Error {
	if r.server == nil || r.listener == nil || r.routes == nil || r.done == nil {
		return cd.NewError(cd.IllegalParam, "HTTP route registry is not configured")
	}
	return nil
}

func (r *routeRegistry) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.server == nil || r.listener == nil || r.done == nil {
		return errors.New("HTTP route registry is not configured")
	}
	if r.started {
		return nil
	}
	server, listener, done := r.server, r.listener, r.done
	r.started = true
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		done <- err
	}()
	return nil
}

func (r *routeRegistry) Teardown(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r.server != nil {
		_ = r.server.Shutdown(ctx)
	}
	if r.listener != nil {
		_ = r.listener.Close()
	}
	r.mu.Lock()
	r.routes = nil
	r.server = nil
	r.listener = nil
	r.started = false
	r.mu.Unlock()
}

func (r *routeRegistry) GetRouteRegistry() enginehttp.RouteRegistry { return r.routes }
func (r *routeRegistry) Done() <-chan error                         { return r.done }

func secureLocalHandler(next http.Handler, bindHost, secretKey string) http.Handler {
	return securityHeaders(localOnlyHostGuard(localOnlyBrowserGuard(remotePushGuard(next, secretKey), bindHost), bindHost))
}

func remotePushGuard(next http.Handler, secretKey string) http.Handler {
	secretKey = strings.TrimSpace(secretKey)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/skill-repository/push" {
			if secretKey == "" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = fmt.Fprintf(w, `{"code":%q,"message":%q}`, biz.CodeReadOnly, "skill-hubd 未配置 secretKey，禁止将本地仓库推送至远端")
				return
			}
			if r.Header.Get(biz.SecretKeyHeader) != secretKey {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprintf(w, `{"code":%q,"message":%q}`, biz.CodeUnauthorized, "secretKey 无效或缺失")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func localOnlyHostGuard(next http.Handler, bindHost string) http.Handler {
	if !isLoopbackHost(bindHost) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			http.Error(w, "skill-hubd only accepts loopback host headers", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func localOnlyBrowserGuard(next http.Handler, bindHost string) http.Handler {
	if !isLoopbackHost(bindHost) {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isUnsafeMethod(r.Method) && !isAllowedBrowserWriteRequest(r) {
			http.Error(w, "skill-hubd rejected non-loopback browser write request", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func isAllowedBrowserWriteRequest(r *http.Request) bool {
	if site := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")); strings.EqualFold(site, "cross-site") {
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return isLoopbackURL(origin)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return isLoopbackURL(referer)
	}
	return true
}

func isLoopbackURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && isLoopbackHost(parsed.Host)
}

func isLoopbackHost(value string) bool {
	host := normalizeHost(value)
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizeHost(value string) string {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(value, "[]")
	return strings.TrimSuffix(value, ".")
}

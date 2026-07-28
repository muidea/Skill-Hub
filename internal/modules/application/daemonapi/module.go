// Package daemonapi exposes the existing HTTP and Web UI contracts through a
// framework Application Module. It owns route declaration, not the listener.
package daemonapi

import (
	"context"
	"net/http"

	registrycommon "github.com/muidea/skill-hub/internal/initiators/routeregistry/pkg/common"
	httpapimodule "github.com/muidea/skill-hub/internal/modules/application/httpapi"
	webuimodule "github.com/muidea/skill-hub/internal/modules/application/webui"
	localruntime "github.com/muidea/skill-hub/internal/runtime/local"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/framework/plugin/initiator"
	pluginmodule "github.com/muidea/magicCommon/framework/plugin/module"
	"github.com/muidea/magicCommon/task"
	enginehttp "github.com/muidea/magicEngine/http"
)

const moduleID = "skillhub.application.daemonapi"

func init() { pluginmodule.Register(New()) }

type Module struct {
	routes  registrycommon.RouteRegistryHelper
	api     http.Handler
	webUI   http.Handler
	httpAPI *httpapimodule.HTTPAPI
}

func New() *Module            { return &Module{} }
func (m *Module) ID() string  { return moduleID }
func (m *Module) Weight() int { return 100 }

func (m *Module) Setup(_ context.Context, hub event.Hub, background task.BackgroundRoutine) *cd.Error {
	routes, err := initiator.GetEntity(registrycommon.RouteRegistryInitiator, registrycommon.RouteRegistryHelper(nil))
	if err != nil {
		return err
	}
	if routes.GetRouteRegistry() == nil {
		return cd.NewError(cd.IllegalParam, "HTTP route registry is unavailable")
	}
	m.routes = routes
	m.httpAPI = httpapimodule.NewWithRuntime(localruntime.NewWithEventHub(hub, background))
	m.api = m.httpAPI.Handler()
	m.webUI = webuimodule.New().Handler()
	return nil
}

func (m *Module) Run(context.Context) *cd.Error {
	if m.routes == nil || m.routes.GetRouteRegistry() == nil || m.api == nil || m.webUI == nil {
		return cd.NewError(cd.IllegalParam, "daemon API routes are not configured")
	}
	routes := m.routes.GetRouteRegistry()
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
		routes.AddHandler("/api", method, serve(m.api))
		routes.AddHandler("/api/**", method, serve(m.api))
		routes.AddHandler("/**", method, serve(m.webUI))
	}
	return nil
}

func (m *Module) Teardown(context.Context) {
	if m.httpAPI != nil {
		m.httpAPI.Close()
		m.httpAPI = nil
	}
	m.routes = nil
	m.api = nil
	m.webUI = nil
}

func serve(handler http.Handler) enginehttp.RouteHandleFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

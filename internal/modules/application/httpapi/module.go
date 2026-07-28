package httpapi

import (
	"net/http"

	"github.com/muidea/skill-hub/internal/modules/application/httpapi/service"
	localruntime "github.com/muidea/skill-hub/internal/runtime/local"
)

type HTTPAPI struct {
	servicePtr *service.HTTPAPI
}

func New() *HTTPAPI {
	return &HTTPAPI{
		servicePtr: service.New(),
	}
}

func NewWithRuntime(runtime *localruntime.Runtime) *HTTPAPI {
	return &HTTPAPI{servicePtr: service.NewWithRuntime(runtime)}
}

// Handler is the narrow inbound facade used by the daemon route module.
func (h *HTTPAPI) Handler() http.Handler { return h.servicePtr.Handler() }
func (h *HTTPAPI) Close()                { h.servicePtr.Close() }

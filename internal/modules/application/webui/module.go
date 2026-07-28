package webui

import (
	"net/http"

	"github.com/muidea/skill-hub/internal/modules/application/webui/service"
)

type WebUI struct {
	servicePtr *service.WebUI
}

func New() *WebUI {
	return &WebUI{
		servicePtr: service.New(),
	}
}

// Handler is the narrow inbound facade used by the daemon route module.
func (w *WebUI) Handler() http.Handler { return w.servicePtr.Handler() }

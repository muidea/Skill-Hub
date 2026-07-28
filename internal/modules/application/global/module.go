// Package global exposes machine-global skill use cases as an Application
// Module. It composes repository data with global agent state; it is not a
// resource Block.
package global

import (
	"github.com/muidea/skill-hub/internal/modules/application/global/biz"
	globalservice "github.com/muidea/skill-hub/internal/modules/application/global/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type Global struct {
	bizPtr     *biz.Global
	servicePtr *globalservice.Global
}

func New(hub event.Hub, background task.BackgroundRoutine) *Global {
	bizPtr := biz.New(hub, background)
	return &Global{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (g *Global) Service() *globalservice.Global { return g.servicePtr }
func (g *Global) Teardown() {
	if g.bizPtr != nil {
		g.bizPtr.Teardown()
		g.bizPtr = nil
	}
}

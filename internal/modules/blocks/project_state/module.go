package projectstate

import (
	"context"

	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/biz"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	pluginmodule "github.com/muidea/magicCommon/framework/plugin/module"
	"github.com/muidea/magicCommon/task"
)

func init() { pluginmodule.Register(New()) }

type ProjectState struct {
	bizPtr     *biz.ProjectState
	servicePtr *service.ProjectState
}

func New() *ProjectState {
	return &ProjectState{
		servicePtr: service.New(),
	}
}

func (p *ProjectState) ID() string  { return common.ModuleName }
func (p *ProjectState) Weight() int { return 10 }
func (p *ProjectState) Setup(_ context.Context, hub event.Hub, background task.BackgroundRoutine) *cd.Error {
	p.bizPtr = biz.New(hub, background, p.servicePtr)
	return nil
}
func (p *ProjectState) Run(context.Context) *cd.Error { return nil }
func (p *ProjectState) Teardown(context.Context) {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

func (p *ProjectState) Service() *service.ProjectState {
	return p.servicePtr
}

package project_lifecycle

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_lifecycle/biz"
	"github.com/muidea/skill-hub/internal/modules/application/project_lifecycle/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectLifecycle struct {
	bizPtr     *biz.ProjectLifecycle
	servicePtr *service.ProjectLifecycle
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectLifecycle {
	bizPtr := biz.New(hub, background)
	return &ProjectLifecycle{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (p *ProjectLifecycle) Service() *service.ProjectLifecycle {
	return p.servicePtr
}

func (p *ProjectLifecycle) Teardown() {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

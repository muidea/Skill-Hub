package project_status

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_status/biz"
	"github.com/muidea/skill-hub/internal/modules/application/project_status/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectStatus struct {
	bizPtr     *biz.ProjectStatus
	servicePtr *service.ProjectStatus
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectStatus {
	bizPtr := biz.New(hub, background)
	return &ProjectStatus{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (p *ProjectStatus) Service() *service.ProjectStatus {
	return p.servicePtr
}

func (p *ProjectStatus) Teardown() {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

package project_apply

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_apply/biz"
	"github.com/muidea/skill-hub/internal/modules/application/project_apply/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectApply struct {
	bizPtr     *biz.ProjectApply
	servicePtr *service.ProjectApply
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectApply {
	bizPtr := biz.New(hub, background)
	return &ProjectApply{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (p *ProjectApply) Service() *service.ProjectApply {
	return p.servicePtr
}

func (p *ProjectApply) Teardown() {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

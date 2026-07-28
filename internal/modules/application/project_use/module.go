package project_use

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_use/biz"
	"github.com/muidea/skill-hub/internal/modules/application/project_use/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectUse struct {
	bizPtr     *biz.ProjectUse
	servicePtr *service.ProjectUse
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectUse {
	bizPtr := biz.New(hub, background)
	return &ProjectUse{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (p *ProjectUse) Service() *service.ProjectUse {
	return p.servicePtr
}

func (p *ProjectUse) Teardown() {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

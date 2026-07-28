package project_inventory

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_inventory/biz"
	"github.com/muidea/skill-hub/internal/modules/application/project_inventory/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectInventory struct {
	bizPtr     *biz.ProjectInventory
	servicePtr *service.ProjectInventory
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectInventory {
	bizPtr := biz.New(hub, background)
	return &ProjectInventory{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (p *ProjectInventory) Service() *service.ProjectInventory {
	return p.servicePtr
}

func (p *ProjectInventory) Teardown() {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

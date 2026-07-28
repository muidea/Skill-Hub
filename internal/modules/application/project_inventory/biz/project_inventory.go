package biz

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_inventory/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "project_inventory"

type ProjectInventory struct {
	basebiz.Base
	service *service.ProjectInventory
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectInventory {
	return &ProjectInventory{
		Base:    basebiz.New(ownerID, hub, background),
		service: service.New(projectstateport.New(hub, ownerID)),
	}
}

func (b *ProjectInventory) Service() *service.ProjectInventory { return b.service }
func (b *ProjectInventory) Teardown()                          {}

package biz

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_lifecycle/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "project_lifecycle"

type ProjectLifecycle struct {
	basebiz.Base
	service *service.ProjectLifecycle
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectLifecycle {
	return &ProjectLifecycle{
		Base: basebiz.New(ownerID, hub, background),
		service: service.New(
			projectstateport.New(hub, ownerID),
			repositoryport.NewProjectSource(hub, ownerID),
		),
	}
}

func (b *ProjectLifecycle) Service() *service.ProjectLifecycle { return b.service }
func (b *ProjectLifecycle) Teardown()                          {}

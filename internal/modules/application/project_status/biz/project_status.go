package biz

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_status/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "project_status"

type ProjectStatus struct {
	basebiz.Base
	service *service.ProjectStatus
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectStatus {
	return &ProjectStatus{
		Base: basebiz.New(ownerID, hub, background),
		service: service.New(
			projectstateport.New(hub, ownerID),
			repositoryport.NewProjectSource(hub, ownerID),
		),
	}
}

func (b *ProjectStatus) Service() *service.ProjectStatus { return b.service }
func (b *ProjectStatus) Teardown()                       {}

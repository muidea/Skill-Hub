package biz

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_use/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "project_use"

type ProjectUse struct {
	basebiz.Base
	service *service.ProjectUse
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectUse {
	return &ProjectUse{
		Base: basebiz.New(ownerID, hub, background),
		service: service.New(
			projectstateport.New(hub, ownerID),
			repositoryport.NewProjectSource(hub, ownerID),
		),
	}
}

func (b *ProjectUse) Service() *service.ProjectUse { return b.service }
func (b *ProjectUse) Teardown()                    {}

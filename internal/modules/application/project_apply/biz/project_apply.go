// Package biz owns project-apply use-case orchestration and its EventHub
// identity. The service remains protocol-free and receives only typed ports.
package biz

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_apply/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "project_apply"

type ProjectApply struct {
	basebiz.Base
	service *service.ProjectApply
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectApply {
	return &ProjectApply{
		Base: basebiz.New(ownerID, hub, background),
		service: service.New(
			projectstateport.New(hub, ownerID),
			repositoryport.NewProjectSource(hub, ownerID),
		),
	}
}

func (b *ProjectApply) Service() *service.ProjectApply { return b.service }

// Teardown is intentionally empty: this owner owns no subscriptions or
// background tasks. Its dependencies are released by their respective owners.
func (b *ProjectApply) Teardown() {}

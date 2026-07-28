// Package biz owns project-feedback orchestration and its EventHub identity.
package biz

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_feedback/service"
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

const ownerID = "project_feedback"

type ProjectFeedback struct {
	basebiz.Base
	service *service.ProjectFeedback
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectFeedback {
	return &ProjectFeedback{
		Base: basebiz.New(ownerID, hub, background),
		service: service.New(
			projectstateport.New(hub, ownerID),
			repositoryport.NewProjectSource(hub, ownerID),
		),
	}
}

func (b *ProjectFeedback) Service() *service.ProjectFeedback { return b.service }
func (b *ProjectFeedback) Teardown()                         {}

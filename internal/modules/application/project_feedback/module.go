package project_feedback

import (
	"github.com/muidea/skill-hub/internal/modules/application/project_feedback/biz"
	"github.com/muidea/skill-hub/internal/modules/application/project_feedback/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectFeedback struct {
	bizPtr     *biz.ProjectFeedback
	servicePtr *service.ProjectFeedback
}

func New(hub event.Hub, background task.BackgroundRoutine) *ProjectFeedback {
	bizPtr := biz.New(hub, background)
	return &ProjectFeedback{bizPtr: bizPtr, servicePtr: bizPtr.Service()}
}

func (p *ProjectFeedback) Service() *service.ProjectFeedback {
	return p.servicePtr
}

func (p *ProjectFeedback) Teardown() {
	if p.bizPtr != nil {
		p.bizPtr.Teardown()
		p.bizPtr = nil
	}
}

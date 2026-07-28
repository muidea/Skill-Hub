package biz

import (
	"context"
	"testing"

	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/events"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func TestProjectStateRejectsInvalidLoadCommand(t *testing.T) {
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(1)
	owner := New(hub, background, service.New())
	t.Cleanup(func() {
		owner.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})

	result := hub.Send(event.NewEvent(events.TopicLoadProject, "test", common.ModuleName, nil, "invalid"))
	if result == nil || result.Error() == nil {
		t.Fatal("invalid project-state command should return an error")
	}
}

func TestProjectStateRejectsInvalidMaintenanceCommands(t *testing.T) {
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(1)
	owner := New(hub, background, service.New())
	t.Cleanup(func() {
		owner.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})

	for _, topic := range []string{events.TopicProjectHasSkill, events.TopicRemoveSkill, events.TopicPruneProjects} {
		result := hub.Send(event.NewEvent(topic, "test", common.ModuleName, nil, "invalid"))
		if result == nil || result.Error() == nil {
			t.Fatalf("%s should reject an invalid command", topic)
		}
	}
}

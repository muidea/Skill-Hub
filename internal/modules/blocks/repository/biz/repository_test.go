package biz

import (
	"context"
	"testing"

	"github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/pkg/events"
	"github.com/muidea/skill-hub/internal/modules/blocks/repository/service"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func TestRepositoryRejectsInvalidPathCommand(t *testing.T) {
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(1)
	owner := New(hub, background, service.New())
	t.Cleanup(func() {
		owner.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})

	result := hub.Send(event.NewEvent(events.TopicRepositoryPath, "test", common.ModuleName, nil, "invalid"))
	if result == nil || result.Error() == nil {
		t.Fatal("invalid repository command should return an error")
	}
}

func TestRepositoryRejectsInvalidGetAndCheckDefaultSkillCommands(t *testing.T) {
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(1)
	owner := New(hub, background, service.New())
	t.Cleanup(func() {
		owner.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})

	for _, topic := range []string{events.TopicGetRepository, events.TopicCheckDefaultSkill} {
		result := hub.Send(event.NewEvent(topic, "test", common.ModuleName, nil, "invalid"))
		if result == nil || result.Error() == nil {
			t.Fatalf("%s should reject an invalid command", topic)
		}
	}
}

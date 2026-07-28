package local

import (
	"context"
	"testing"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func TestRuntimeExposesFocusedLocalCapabilities(t *testing.T) {
	runtime := New()
	t.Cleanup(runtime.Close)
	if runtime.Repository() == nil {
		t.Fatal("repository capability is not configured")
	}
	if runtime.Git() == nil {
		t.Fatal("git capability is not configured")
	}
	if runtime.Skill() == nil {
		t.Fatal("skill capability is not configured")
	}
	if runtime.Project() == nil {
		t.Fatal("project capability is not configured")
	}
	if runtime.Global() == nil {
		t.Fatal("global capability is not configured")
	}
	if runtime.Maintenance() == nil {
		t.Fatal("maintenance capability is not configured")
	}
}

func TestRuntimeWithEventHubDoesNotTerminateFrameworkInfrastructure(t *testing.T) {
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(1)
	t.Cleanup(func() {
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})

	runtime := NewWithEventHub(hub, background)
	runtime.Close()

	observer := event.NewSimpleObserver("test", hub)
	observer.Subscribe("test.ping", func(_ event.Event, result event.Result) {
		if result != nil {
			result.Set("ok", nil)
		}
	})
	result := hub.Send(event.NewEvent("test.ping", "test", "test", nil, nil))
	value, err := result.Get()
	if err != nil || value != "ok" {
		t.Fatalf("framework Hub after local Close = (%v, %v), want (ok, nil)", value, err)
	}
	observer.Unsubscribe("test.ping")
}

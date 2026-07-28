package biz

import (
	"context"
	"testing"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func TestBaseSendsAndUnsubscribesTypedEvents(t *testing.T) {
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(2)
	t.Cleanup(func() {
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})

	base := New("owner", hub, background)
	base.SubscribeFunc("owner.query", func(_ event.Event, result event.Result) {
		if result != nil {
			result.Set("ok", nil)
		}
	})
	result := base.SendEvent(event.NewEvent("owner.query", "caller", "owner", nil, "request"))
	value, err := result.Get()
	if err != nil || value != "ok" {
		t.Fatalf("SendEvent() = (%v, %v), want (ok, nil)", value, err)
	}

	base.UnsubscribeFunc("owner.query")
}

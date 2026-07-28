// Package biz provides the owner-neutral EventHub base for framework Modules
// and Blocks. Business owners embed Base in their own biz type; this package
// never defines a business topic, DTO or lifecycle registration.
package biz

import (
	"context"
	"time"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type Base struct {
	id                string
	eventHub          event.Hub
	simpleObserver    event.SimpleObserver
	backgroundRoutine task.BackgroundRoutine
}

type routineTask struct{ funcPtr func() }

func (t *routineTask) Run() { t.funcPtr() }

func New(id string, eventHub event.Hub, background task.BackgroundRoutine) Base {
	return Base{
		id:                id,
		eventHub:          eventHub,
		simpleObserver:    event.NewSimpleObserver(id, eventHub),
		backgroundRoutine: background,
	}
}

func (b *Base) ID() string { return b.id }

// EventHub is available only to an embedding owner Biz for typed contracts.
// Module roots, protocol services and process services must not retain it.
func (b *Base) EventHub() event.Hub { return b.eventHub }

func (b *Base) BackgroundRoutine() task.BackgroundRoutine { return b.backgroundRoutine }
func (b *Base) Subscribe(eventID string, observer event.Observer) {
	b.eventHub.Subscribe(eventID, observer)
}
func (b *Base) Unsubscribe(eventID string, observer event.Observer) {
	b.eventHub.Unsubscribe(eventID, observer)
}
func (b *Base) SubscribeFunc(eventID string, observerFunc event.ObserverFunc) {
	b.simpleObserver.Subscribe(eventID, observerFunc)
}
func (b *Base) UnsubscribeFunc(eventID string)       { b.simpleObserver.Unsubscribe(eventID) }
func (b *Base) PostEvent(e event.Event)              { b.eventHub.Post(e) }
func (b *Base) SendEvent(e event.Event) event.Result { return b.eventHub.Send(e) }
func (b *Base) SyncTask(fn func())                   { _ = b.backgroundRoutine.SyncTask(&routineTask{funcPtr: fn}) }
func (b *Base) AsyncTask(fn func())                  { _ = b.backgroundRoutine.AsyncTask(&routineTask{funcPtr: fn}) }
func (b *Base) Timer(ctx context.Context, interval, offset time.Duration, fn func()) {
	_ = b.backgroundRoutine.Timer(ctx, &routineTask{funcPtr: fn}, interval, offset)
}

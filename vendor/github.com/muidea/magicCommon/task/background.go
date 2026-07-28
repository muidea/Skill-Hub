package task

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/muidea/magicCommon/execute"
)

// Task 任务对象
type Task interface {
	Run()
}

type routineTask struct {
	funcPtr func()
}

func (s *routineTask) Run() {
	s.funcPtr()
}

type BackgroundRoutine interface {
	AsyncTask(task Task) error
	SyncTask(task Task) error
	SyncTaskWithTimeOut(task Task, timeout time.Duration) error
	AsyncFunction(function func()) error
	SyncFunction(function func()) error
	SyncFunctionWithTimeOut(function func(), timeout time.Duration) error
	Timer(ctx context.Context, task Task, intervalValue time.Duration, offsetValue time.Duration) error
	Shutdown(ctx context.Context) bool
}

type syncTask struct {
	resultChannel chan bool
	rawTask       Task
	timedOut      atomic.Bool
}

func (s *syncTask) Run() {
	s.rawTask.Run()

	if !s.timedOut.Load() {
		s.resultChannel <- true
	}
}

func (s *syncTask) Wait(timeout time.Duration) {
	switch timeout {
	case -1:
		<-s.resultChannel
	default:
		select {
		case <-s.resultChannel:
		case <-time.After(timeout):
			s.timedOut.Store(true)
		}
	}
}

type taskChannel chan Task

// backgroundRoutine backGround routine
type backgroundRoutine struct {
	execute.Execute

	taskChannel taskChannel
	submitMu    sync.RWMutex
	closed      bool
	closeOnce   sync.Once
	loopDone    chan struct{}
}

// NewBackgroundRoutine new Background routine
func NewBackgroundRoutine(capacitySize int) BackgroundRoutine {
	bg := &backgroundRoutine{
		Execute:     execute.NewExecute(capacitySize),
		taskChannel: make(taskChannel, capacitySize),
		loopDone:    make(chan struct{}),
	}

	bg.run()

	return bg
}

func (s *backgroundRoutine) run() {
	s.Run(s.loop)
}

func (s *backgroundRoutine) loop() {
	defer close(s.loopDone)
	for task := range s.taskChannel {
		s.Run(func() {
			task.Run()
		})
	}
}

func (s *backgroundRoutine) AsyncTask(task Task) error {
	return s.submitTask(task)
}

func (s *backgroundRoutine) SyncTask(task Task) error {
	_ = s.SyncTaskWithTimeOut(task, -1)
	return nil
}

func (s *backgroundRoutine) SyncTaskWithTimeOut(task Task, timeout time.Duration) error {
	st := &syncTask{rawTask: task, resultChannel: make(chan bool, 1)}
	if err := s.submitTask(st); err != nil {
		return err
	}

	st.Wait(timeout)
	return nil
}

func (s *backgroundRoutine) AsyncFunction(function func()) error {
	if function == nil {
		return fmt.Errorf("function is nil")
	}
	return s.AsyncTask(&routineTask{funcPtr: function})
}

func (s *backgroundRoutine) SyncFunction(function func()) error {
	if function == nil {
		return fmt.Errorf("function is nil")
	}
	return s.SyncTask(&routineTask{funcPtr: function})
}

func (s *backgroundRoutine) SyncFunctionWithTimeOut(function func(), timeout time.Duration) error {
	if function == nil {
		return fmt.Errorf("function is nil")
	}
	return s.SyncTaskWithTimeOut(&routineTask{funcPtr: function}, timeout)
}

const onDayDuration = 24 * time.Hour

func (s *backgroundRoutine) Timer(ctx context.Context, task Task, intervalValue time.Duration, offsetValue time.Duration) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if intervalValue <= 0 {
		return fmt.Errorf("intervalValue must be positive")
	}

	go func() {
		curOffset := func() time.Duration {
			now := time.Now()
			nowOffset := time.Duration(now.Hour())*time.Hour + time.Duration(now.Minute())*time.Minute + time.Duration(now.Second())*time.Second
			if intervalValue < 24*time.Hour {
				return (nowOffset/intervalValue+1)*intervalValue - nowOffset
			}

			return (offsetValue + intervalValue - nowOffset + onDayDuration) % onDayDuration
		}()

		timer := time.NewTimer(curOffset)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if err := s.AsyncTask(task); err != nil {
			return
		}

		timeOutTimer := time.NewTicker(intervalValue)
		defer timeOutTimer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeOutTimer.C:
				if err := s.AsyncTask(task); err != nil {
					return
				}
			}
		}
	}()

	return nil
}

func (s *backgroundRoutine) Shutdown(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closeOnce.Do(func() {
		s.submitMu.Lock()
		s.closed = true
		close(s.taskChannel)
		s.submitMu.Unlock()
	})

	select {
	case <-s.loopDone:
	case <-ctx.Done():
		return false
	}
	return s.WaitContext(ctx)
}

func (s *backgroundRoutine) submitTask(task Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	s.submitMu.RLock()
	defer s.submitMu.RUnlock()

	if s.closed {
		return fmt.Errorf("background routine is closed")
	}

	s.taskChannel <- task
	return nil
}

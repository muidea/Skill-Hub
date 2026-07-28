package execute

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/muidea/magicCommon/foundation/util"
)

const defaultWaitTimeout = 5 * time.Second

type Execute struct {
	mu            sync.Mutex
	capacityQueue chan bool
	queueLength   int
	activeCount   int
	capacitySize  int
}

func NewExecute(capacitySize int) Execute {
	if capacitySize <= 0 {
		capacitySize = 10
	}

	return Execute{
		capacitySize:  capacitySize,
		capacityQueue: make(chan bool, capacitySize),
	}
}

func (s *Execute) Lock() { /* for noCopy */ }

func (s *Execute) Unlock() { /* for noCopy */ }

func (s *Execute) Run(funcPtr func()) {
	s.mu.Lock()
	if s.queueLength >= s.capacitySize {
		s.mu.Unlock()
		slog.Warn("execute queue is full, length:s.queueLength, capacity:s.capacitySize", "field", s.queueLength, "error", s.capacitySize)
	} else if s.queueLength >= int(math.Floor(float64(s.capacitySize)*0.8)) {
		s.mu.Unlock()
		slog.Warn("queue lengths are at warning levels, length:s.queueLength, capacity:s.capacitySize", "field", s.queueLength, "error", s.capacitySize)
	} else {
		s.mu.Unlock()
	}

	s.capacityQueue <- true
	s.mu.Lock()
	s.queueLength++
	s.activeCount++
	s.mu.Unlock()
	go func() {
		defer func() {
			if err := recover(); err != nil {
				stackInfo := util.GetStack(3)
				slog.Error("PANIC: err\nstackInfo", "field", err, "error", stackInfo)
			}

			<-s.capacityQueue
			s.mu.Lock()
			s.queueLength--
			s.activeCount--
			s.mu.Unlock()
		}()

		funcPtr()
	}()
}

func (s *Execute) Wait() {
	_ = s.WaitTimeout(defaultWaitTimeout)
}

func (s *Execute) Idle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activeCount == 0
}

// WaitTimeout waits until all submitted tasks drain or the timeout expires.
// It returns true when the queue becomes idle before the timeout.
// A non-positive timeout means wait indefinitely.
func (s *Execute) WaitTimeout(timeout time.Duration) bool {
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	for {
		s.mu.Lock()
		if s.activeCount == 0 {
			s.mu.Unlock()
			return true
		}
		s.mu.Unlock()

		if !deadline.IsZero() && time.Now().After(deadline) {
			return false
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// WaitContext waits until all submitted tasks drain or the context is canceled.
// It returns true when the queue becomes idle before the context finishes.
func (s *Execute) WaitContext(ctx context.Context) bool {
	for {
		if s.Idle() {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(10 * time.Millisecond):
		}
	}
}

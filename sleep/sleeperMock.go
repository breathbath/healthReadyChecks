package sleep

import (
	"sync"
	"time"
)

// SleeperMock pretends to be a sleeper for testing
type SleeperMock struct {
	TriggeredSleepDuration time.Duration
	WasTriggered           bool
	TriggerCount           int
	lock                   sync.Mutex
}

// NewSleeperMock constructor
func NewSleeperMock() *SleeperMock {
	return &SleeperMock{
		WasTriggered:           false,
		lock:                   sync.Mutex{},
		TriggeredSleepDuration: time.Duration(0),
		TriggerCount:           0,
	}
}

// Sleep fake sleep implementation
func (sm *SleeperMock) Sleep(t time.Duration) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.WasTriggered = true
	sm.TriggeredSleepDuration = t
	sm.TriggerCount++
}

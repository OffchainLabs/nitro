package util

import (
	"sync"
	"time"
)

type CountUpTimer struct {
	mutex             sync.Mutex
	timeReference     TimeReference
	running           bool
	zeropointRunning  time.Time
	elapsedNotRunning time.Duration
}

func NewCountUpTimer(timeReference TimeReference) *CountUpTimer {
	return &CountUpTimer{
		timeReference:     timeReference,
		running:           false,
		elapsedNotRunning: 0,
	}
}

func (ct *CountUpTimer) Clone() *CountUpTimer {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return &CountUpTimer{
		timeReference:     ct.timeReference,
		running:           ct.running,
		zeropointRunning:  ct.zeropointRunning,
		elapsedNotRunning: ct.elapsedNotRunning,
	}
}

func (ct *CountUpTimer) Start() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	if !ct.running {
		ct.running = true
		ct.zeropointRunning = ct.timeReference.Get().Add(-ct.elapsedNotRunning)
	}
}

func (ct *CountUpTimer) Stop() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	if ct.running {
		ct.running = false
		ct.elapsedNotRunning = ct.timeReference.Get().Sub(ct.zeropointRunning)
	}
}

func (ct *CountUpTimer) IsRunning() bool {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.running
}

func (ct *CountUpTimer) Get() time.Duration {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.getLocked()
}

func (ct *CountUpTimer) getLocked() time.Duration {
	if ct.running {
		return ct.timeReference.Get().Sub(ct.zeropointRunning)
	} else {
		return ct.elapsedNotRunning
	}
}

func (ct *CountUpTimer) Set(val time.Duration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(val)
}

func (ct *CountUpTimer) setLocked(val time.Duration) {
	if ct.running {
		ct.zeropointRunning = ct.timeReference.Get().Add(-val)
	} else {
		ct.elapsedNotRunning = val
	}
}

func (ct *CountUpTimer) Add(delta time.Duration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	if ct.running {
		ct.zeropointRunning = ct.zeropointRunning.Add(-delta)
	} else {
		ct.elapsedNotRunning = ct.elapsedNotRunning + delta
	}
}

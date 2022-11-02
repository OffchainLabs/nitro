package protocol

import (
	"sync"
)

type countUpTimer struct {
	mutex            sync.Mutex
	timeReference    TimeReference
	running          bool
	zeropointOrValue SecondsDuration
}

func newCountUpTimer(timeReference TimeReference) *countUpTimer {
	return &countUpTimer{
		timeReference:    timeReference,
		running:          false,
		zeropointOrValue: 0,
	}
}

func (ct *countUpTimer) clone() *countUpTimer {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return &countUpTimer{
		timeReference:    ct.timeReference,
		running:          ct.running,
		zeropointOrValue: ct.zeropointOrValue,
	}
}

func (ct *countUpTimer) start() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	if !ct.running {
		ct.running = true
		ct.zeropointOrValue = ct.timeReference.Get() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) stop() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	if ct.running {
		ct.running = false
		ct.zeropointOrValue = ct.timeReference.Get() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) isRunning() bool {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.running
}

func (ct *countUpTimer) get() SecondsDuration {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.getLocked()
}

func (ct *countUpTimer) getLocked() SecondsDuration {
	if ct.running {
		return ct.timeReference.Get() - ct.zeropointOrValue
	} else {
		return ct.zeropointOrValue
	}
}

func (ct *countUpTimer) set(val SecondsDuration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(val)
}

func (ct *countUpTimer) setLocked(val SecondsDuration) {
	wasRunning := ct.running
	if wasRunning {
		ct.running = false
		ct.zeropointOrValue = ct.timeReference.Get() - ct.zeropointOrValue
	}
	ct.zeropointOrValue = val
	if wasRunning {
		ct.running = true
		ct.zeropointOrValue = ct.timeReference.Get() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) add(delta SecondsDuration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(ct.getLocked() + delta)
}

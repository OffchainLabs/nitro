package protocol

import (
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"sync"
)

type countUpTimer struct {
	mutex            sync.Mutex
	timeReference    util.TimeReference
	running          bool
	zeropointOrValue util.SecondsDuration
}

func newCountUpTimer(timeReference util.TimeReference) *countUpTimer {
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

func (ct *countUpTimer) get() util.SecondsDuration {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.getLocked()
}

func (ct *countUpTimer) getLocked() util.SecondsDuration {
	if ct.running {
		return ct.timeReference.Get() - ct.zeropointOrValue
	} else {
		return ct.zeropointOrValue
	}
}

func (ct *countUpTimer) set(val util.SecondsDuration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(val)
}

func (ct *countUpTimer) setLocked(val util.SecondsDuration) {
	ct.zeropointOrValue = val
	if ct.running {
		ct.zeropointOrValue = ct.timeReference.Get() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) add(delta util.SecondsDuration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(ct.getLocked() + delta)
}

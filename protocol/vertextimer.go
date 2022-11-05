package protocol

import (
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"sync"
	"time"
)

type countUpTimer struct {
	mutex            sync.Mutex
	timeReference    util.TimeReference
	running          bool
	zeropointOrValue int64
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
		ct.zeropointOrValue = ct.timeReference.Get().UnixNano() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) stop() {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	if ct.running {
		ct.running = false
		ct.zeropointOrValue = ct.timeReference.Get().UnixNano() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) isRunning() bool {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.running
}

func (ct *countUpTimer) get() time.Duration {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	return ct.getLocked()
}

func (ct *countUpTimer) getLocked() time.Duration {
	if ct.running {
		return time.Duration(ct.timeReference.Get().UnixNano() - ct.zeropointOrValue)
	} else {
		return time.Duration(ct.zeropointOrValue)
	}
}

func (ct *countUpTimer) set(val time.Duration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(val)
}

func (ct *countUpTimer) setLocked(val time.Duration) {
	ct.zeropointOrValue = int64(val)
	if ct.running {
		ct.zeropointOrValue = ct.timeReference.Get().UnixNano() - ct.zeropointOrValue
	}
}

func (ct *countUpTimer) add(delta time.Duration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()
	ct.setLocked(ct.getLocked() + delta)
}

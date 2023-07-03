// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE
package time

import (
	"sync"
	"time"
)

type Reference interface {
	Get() time.Time
	Sleep(duration time.Duration)
	SleepUntil(wakeTime time.Time)
	NewTicker(duration time.Duration) GenericTimeTicker
}

type GenericTimeTicker interface {
	C() <-chan time.Time
	Stop()
}

type realTimeReference struct{}

func NewRealTimeReference() Reference {
	return realTimeReference{}
}

func (realTimeReference) Get() time.Time {
	return time.Now()
}

func (realTimeReference) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (realTimeReference) SleepUntil(wakeTime time.Time) {
	time.Sleep(time.Until(wakeTime))
}

func (realTimeReference) NewTicker(duration time.Duration) GenericTimeTicker {
	return newRealTimeTicker(duration)
}

type realTimeTicker struct {
	ticker *time.Ticker
}

func newRealTimeTicker(duration time.Duration) *realTimeTicker {
	return &realTimeTicker{time.NewTicker(duration)}
}

func (ticker *realTimeTicker) C() <-chan time.Time {
	return ticker.ticker.C
}

func (ticker *realTimeTicker) Stop() {
	ticker.ticker.Stop()
}

type ArtificialTimeReference struct {
	mutex       sync.RWMutex
	current     time.Time
	changedChan chan struct{} // every time the time changes, this is closed and re-generated
}

func NewArtificialTimeReference() *ArtificialTimeReference {
	return &ArtificialTimeReference{
		mutex:       sync.RWMutex{},
		current:     time.Unix(0, 0),
		changedChan: make(chan struct{}),
	}
}

func (atr *ArtificialTimeReference) Get() time.Time {
	atr.mutex.RLock()
	defer atr.mutex.RUnlock()
	return atr.current
}

func (atr *ArtificialTimeReference) Sleep(duration time.Duration) {
	atr.SleepUntil(atr.Get().Add(duration))
}

func (atr *ArtificialTimeReference) SleepUntil(wakeTime time.Time) {
	for {
		atr.mutex.RLock()
		current := atr.current
		changedChan := atr.changedChan
		atr.mutex.RUnlock()
		if !current.Before(wakeTime) {
			return
		}
		<-changedChan
	}
}

func (atr *ArtificialTimeReference) Set(newVal time.Time) {
	atr.mutex.Lock()
	defer atr.mutex.Unlock()
	atr.setLocked(newVal)
}

func (atr *ArtificialTimeReference) setLocked(newVal time.Time) {
	if newVal.Before(atr.current) {
		return
	}
	changed := newVal.After(atr.current)
	atr.current = newVal
	if changed {
		close(atr.changedChan)
		atr.changedChan = make(chan struct{})
	}
}

func (atr *ArtificialTimeReference) Add(delta time.Duration) {
	atr.mutex.Lock()
	defer atr.mutex.Unlock()
	atr.setLocked(atr.current.Add(delta))
}

func (atr *ArtificialTimeReference) NewTicker(interval time.Duration) GenericTimeTicker {
	ticker := &artificialTicker{
		timeRef:     atr,
		c:           make(chan time.Time),
		interval:    interval,
		next:        atr.Get().Add(interval),
		stoppedChan: make(chan struct{}),
		stopped:     false,
	}
	go func() {
		defer close(ticker.c)
		for {
			ticker.timeRef.mutex.RLock()
			current := atr.current
			changedChan := atr.changedChan
			ticker.timeRef.mutex.RUnlock()
			if current.Before(ticker.next) {
				select {
				case <-changedChan:
				case <-ticker.stoppedChan:
					return
				}
			} else {
				select {
				case ticker.c <- current:
					ticker.next = ticker.timeRef.Get().Add(ticker.interval)
				case <-ticker.stoppedChan:
					return
				}
			}
		}
	}()
	return ticker
}

type artificialTicker struct {
	timeRef     *ArtificialTimeReference
	c           chan time.Time
	interval    time.Duration
	next        time.Time
	closeMutex  sync.Mutex
	stoppedChan chan struct{}
	stopped     bool
}

func (ticker *artificialTicker) C() <-chan time.Time {
	return ticker.c
}

func (ticker *artificialTicker) Stop() {
	ticker.closeMutex.Lock()
	defer ticker.closeMutex.Unlock()
	if !ticker.stopped {
		ticker.stopped = true
		close(ticker.stoppedChan)
	}
}

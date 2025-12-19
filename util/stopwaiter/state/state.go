package state

import (
	"context"
	"errors"
	"sync"
)

// lint:require-exhaustive-initialization
type InternalState struct {
	mutex       sync.RWMutex
	lockedState LockedInternalState
}

func (s *InternalState) Lock() *LockedInternalState {
	s.mutex.Lock()
	return &s.lockedState
}

func (s *InternalState) RLock() *LockedInternalState {
	s.mutex.RLock()
	return &s.lockedState
}

func (s *InternalState) Unlock() {
	s.mutex.Unlock()
}

func (s *InternalState) RUnlock() {
	s.mutex.RUnlock()
}

// lint:require-exhaustive-initialization
type LockedInternalState struct {
	Started   bool
	Stopped   bool
	Name      string
	ctx       context.Context
	parentCtx context.Context
	StopFunc  func()
	WaitChan  <-chan interface{}
}

func (s *LockedInternalState) GetContext() (context.Context, error) {
	if s.Started {
		return s.ctx, nil
	}
	return nil, errors.New("not started")
}

func (s *LockedInternalState) SetCtx(ctx context.Context) {
	s.ctx = ctx
}

func (s *LockedInternalState) GetParentContext() (context.Context, error) {
	if s.Started {
		return s.parentCtx, nil
	}
	return nil, errors.New("not started")
}

func (s *LockedInternalState) SetParentCtx(parentCtx context.Context) {
	s.parentCtx = parentCtx
}

// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package state

import (
	"context"
	"errors"
	"sync"

	"github.com/offchainlabs/nitro/util/stopwaiter/stoppable"
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
	Started       bool
	Stopped       bool
	Name          string
	ctx           context.Context
	parentCtx     context.Context
	StopFunc      func()
	WaitChan      <-chan interface{}
	children      []stoppable.Stoppable
	takenChildren []stoppable.Stoppable // set once when childrenTaken becomes true; preserved for StopAndWait
	childrenTaken bool
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

func (s *LockedInternalState) IsChildrenTaken() bool {
	return s.childrenTaken
}

func (s *LockedInternalState) AppendChild(child stoppable.Stoppable) {
	s.children = append(s.children, child)
}

// TakeChildren atomically claims the children list: stores it in takenChildren,
// clears children, and marks childrenTaken. Must only be called once.
func (s *LockedInternalState) TakeChildren() []stoppable.Stoppable {
	children := s.children
	s.takenChildren = children
	s.children = nil
	s.childrenTaken = true
	return children
}

func (s *LockedInternalState) GetTakenChildren() []stoppable.Stoppable {
	return s.takenChildren
}

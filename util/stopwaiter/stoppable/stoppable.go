// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package stoppable

import "context"

// Stoppable is implemented by any type that can be stopped.
// Used by TrackChild to automatically stop children in reverse order.
type Stoppable interface {
	StopOnly()
	StopAndWait()
}

// StoppableChild extends Stoppable with a Start method that takes a context.
type StoppableChild interface {
	Stoppable
	Start(context.Context)
}

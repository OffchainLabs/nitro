// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package types includes types and interfaces specific to the challenge manager instance.
package types

import (
	"context"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
)

// RivalHandler is the interface between the challenge manager and the assertion
// manager.
//
// The challenge manager implements the interface promising to handle opening
// challenges on correct rival assertions, and the assertion manager is
// responsible for posting correct rival assertions, and notifying the rival
// handler of the existence of the correct rival assertion.
type RivalHandler interface {
	// HandleCorrectRival is called when the assertion manager has posted a correct
	// rival assertion on the chain.
	HandleCorrectRival(context.Context, protocol.AssertionHash) error
}

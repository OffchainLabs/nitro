// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// The debug-block branch pins go-ethereum to a PR branch that does not expose
// core.NewMessageSequencingContext yet. Sequencer execution still needs an
// on-chain mutating context, which matches the older commit context behavior.
func newMessageSequencingContext(wasmTargets []rawdb.WasmTarget) *core.MessageRunContext {
	return core.NewMessageCommitContext(wasmTargets)
}

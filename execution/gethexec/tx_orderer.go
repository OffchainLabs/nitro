// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

// txOrderer decides the tx order of one block. The sequencer creates an orderer per regular-tx
// block and drives it under the createBlockMutex, so implementations don't need to be
// thread-safe.
type txOrderer interface {
	// nextTxFetcher yields the candidates of the block armed by StartBlock one at a time.
	nextTxFetcher

	// StartBlock collects and orders the block's candidate txs, reporting whether there is
	// any work.
	StartBlock() (hasWork bool)

	// TakeRemaining removes and returns the never-yielded candidates for the caller to
	// dispose of.
	TakeRemaining() []txQueueItem
}

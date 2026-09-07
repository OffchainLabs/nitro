// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mevfeed

import "github.com/ethereum/go-ethereum/core/types"

// CanonicalBlockObserver is intentionally a one-way, non-blocking interface.
// Implementations must never make block execution depend on a consumer.
type CanonicalBlockObserver interface {
	TryPublish(block *types.Block, receipts types.Receipts)
}

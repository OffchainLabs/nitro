// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package consensus

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
)

const RPCNamespace = "nitroconsensus"

type InboxBatch struct {
	BatchNum uint64
	Found    bool
}

// not implemented in execution, used as input
// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FindBatchContainingMessage(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64]
	GetL1Confirmations(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64]
}

type ConsensusInfo interface {
	BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[common.BlockMetadata]
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
}

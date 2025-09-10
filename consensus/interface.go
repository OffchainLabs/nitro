package consensus

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
)

const RPCNamespace = "nitroconsensus"

var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

type MessageResult struct {
	BlockHash common.Hash
	SendRoot  common.Hash
}

type InboxBatch struct {
	BatchNum uint64
	Found    bool
}

// not implemented in execution, used as input
// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FindInboxBatchContainingMessage(message arbutil.MessageIndex) containers.PromiseInterface[InboxBatch]
	GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64]
}

type ConsensusInfo interface {
	Synced() containers.PromiseInterface[bool]
	FullSyncProgressMap() containers.PromiseInterface[map[string]interface{}]
	SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex]
	BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[common.BlockMetadata]
}

type ConsensusSequencer interface {
	WriteMessageFromSequencer(msgIdx arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult MessageResult, blockMetadata common.BlockMetadata) containers.PromiseInterface[struct{}]
	ExpectChosenSequencer() containers.PromiseInterface[struct{}]
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
	ConsensusSequencer
}

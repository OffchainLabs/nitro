package consensus

import (
	"errors"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

const RPCNamespace = "nitroconsensus"

var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

// BatchFetcher is required for any execution node
type BatchFetcher interface {
	FindL1BatchForMessage(message arbutil.MessageIndex) containers.PromiseInterface[uint64]
	GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64]
}

type ConsensusInfo interface {
	SyncProgressMap() containers.PromiseInterface[map[string]interface{}]
	SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex]

	// TODO: switch from pulling to pushing safe/finalized
	GetSafeMsgCount() containers.PromiseInterface[arbutil.MessageIndex]
	GetFinalizedMsgCount() containers.PromiseInterface[arbutil.MessageIndex]
}

type ConsensusSequencer interface {
	WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, result execution.MessageResult) containers.PromiseInterface[struct{}]
	ExpectChosenSequencer() containers.PromiseInterface[struct{}]
}

type FullConsensusClient interface {
	BatchFetcher
	ConsensusInfo
	ConsensusSequencer
}

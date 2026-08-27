// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package mel

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
)

// RPCNamespace is where a MEL provider serves the read/query side of MELNative.
const RPCNamespace = "meldataprovider"

// ConsumerRPCNamespace is where a node serves the provider->node side: the provider connects as
// the client and pushes extracted messages and reorg notifications, as in consensus->execution.
const ConsumerRPCNamespace = "nitromelconsumer"

// SequencerMessageResult carries a Go multi-return, which JSON-RPC cannot.
type SequencerMessageResult struct {
	Data      []byte      `json:"data"`
	BlockHash common.Hash `json:"blockHash"`
}

// FinalizedDelayedResult carries ErrDelayedMessageNotYetFinalized in-band as NotYetFinalized, so
// ParentChainBlockNumber (which the delayed sequencer needs) survives; an error would drop it.
type FinalizedDelayedResult struct {
	Message                *arbostypes.L1IncomingMessage `json:"message"`
	AfterInboxAcc          common.Hash                   `json:"afterInboxAcc"`
	ParentChainBlockNumber uint64                        `json:"parentChainBlockNumber"`
	NotYetFinalized        bool                          `json:"notYetFinalized"`
}

// FindInboxBatchResult carries a Go multi-return, which JSON-RPC cannot.
type FindInboxBatchResult struct {
	SeqNum uint64 `json:"seqNum"`
	Found  bool   `json:"found"`
}

// MELNative is the union of everything the node consumes from MEL, implemented by both the
// in-process runner.MessageExtractor and the rpc_client.Client. It is a superset of the narrow
// consumer interfaces in arbnode (BatchDataProvider, DelayedMessageFetcher, ...) and of
// staker.InboxReaderInterface / InboxTrackerInterface.
//
// TODO: the MEL validator adds GetPreimagesForValidation and FindMessageOriginMELState here.
type MELNative interface {
	// Lifecycle / wiring.
	SetMessageConsumer(consumer MessageConsumer) error
	Start(ctx context.Context) error
	StopAndWait()
	Started() bool
	CaughtUp() chan struct{}
	ReorgTo(parentChainBlockNumber uint64) error
	// GetL1Reader cannot go over RPC, so the RPC client returns the local node's reader.
	GetL1Reader() *headerreader.HeaderReader

	// Batch / delayed-message queries.
	GetBatchCount() (uint64, error)
	GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error)
	GetBatchMetadata(seqNum uint64) (BatchMetadata, error)
	GetBatchAcc(seqNum uint64) (common.Hash, error)
	GetBatchParentChainBlock(seqNum uint64) (uint64, error)
	GetDelayedAcc(seqNum uint64) (common.Hash, error)
	GetDelayedCount() (uint64, error)
	GetDelayedMessage(index uint64) (*DelayedInboxMessage, error)
	GetDelayedMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error)
	FindInboxBatchContainingMessage(pos arbutil.MessageIndex) (uint64, bool, error)
	FindParentChainBlockContainingDelayed(ctx context.Context, index uint64) (uint64, error)

	// Sequencer-message + finality queries.
	GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, common.Hash, error)
	GetSequencerMessageBytesForParentBlock(ctx context.Context, seqNum uint64, parentChainBlock uint64) ([]byte, common.Hash, error)
	FinalizedDelayedMessageAtPosition(ctx context.Context, finalizedBlock uint64, lastDelayedAccumulator common.Hash, requestedPosition uint64) (*arbostypes.L1IncomingMessage, common.Hash, uint64, error)
	GetMsgCount() (arbutil.MessageIndex, error)
	GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
	GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error)
	GetSyncProgress(ctx context.Context) (MessageSyncProgress, error)
	SupportsPushingFinalityData() bool

	// MEL state queries.
	GetState(parentChainBlockNumber uint64) (*State, error)
	GetHeadState() (*State, error)
}

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

// RPCNamespace is the JSON-RPC namespace under which a MEL provider serves the
// read/query side of MELNative. Mirrors execution.RPCNamespace ("nitroexecution")
// and consensus.RPCNamespace ("nitroconsensus").
const RPCNamespace = "meldataprovider"

// ConsumerRPCNamespace is the JSON-RPC namespace under which a Nitro node serves the
// provider->node side: a remote MEL provider connects to it and pushes the messages it
// extracts (PushMessages) and notifies it of reorgs (ReorgedToParentChainBlock). The
// producer is the RPC client and the node is the server, exactly as in consensus->execution.
const ConsumerRPCNamespace = "nitromelconsumer"

// SequencerMessageResult is the JSON-RPC return shape for GetSequencerMessageBytes and
// GetSequencerMessageBytesForParentBlock (JSON-RPC cannot carry Go multi-returns).
type SequencerMessageResult struct {
	Data      []byte      `json:"data"`
	BlockHash common.Hash `json:"blockHash"`
}

// FinalizedDelayedResult is the JSON-RPC return shape for FinalizedDelayedMessageAtPosition.
// NotYetFinalized encodes the mel.ErrDelayedMessageNotYetFinalized sentinel inside the result
// (rather than as a Go error) so that ParentChainBlockNumber — which the delayed sequencer uses
// for its short-circuit — survives the round-trip; a non-nil Go error would drop the result.
type FinalizedDelayedResult struct {
	Message                *arbostypes.L1IncomingMessage `json:"message"`
	AfterInboxAcc          common.Hash                   `json:"afterInboxAcc"`
	ParentChainBlockNumber uint64                        `json:"parentChainBlockNumber"`
	NotYetFinalized        bool                          `json:"notYetFinalized"`
}

// FindInboxBatchResult is the JSON-RPC return shape for FindInboxBatchContainingMessage.
type FindInboxBatchResult struct {
	SeqNum uint64 `json:"seqNum"`
	Found  bool   `json:"found"`
}

// MELNative is implemented by both the in-process message extractor
// (arbnode/mel/runner.MessageExtractor) and the JSON-RPC query client
// (arbnode/mel/rpc_runner/rpc_client.Client). It is the union of every method the
// rest of the node consumes from MEL, whether directly or via the narrow consumer
// interfaces defined in arbnode: BatchDataProvider, DelayedMessageFetcher,
// BatchMetadataFetcher, MessageSyncProgressFetcher, MessageCountFetcher,
// ParentChainDataSource, staker.InboxReaderInterface and staker.InboxTrackerInterface.
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
	// GetL1Reader returns the parent chain header reader. A *headerreader.HeaderReader
	// cannot be transported over RPC, so the RPC client returns the local node's reader
	// (it connects to the same parent chain); see rpc_client.
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

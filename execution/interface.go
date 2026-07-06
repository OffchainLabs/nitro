// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package execution

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/containers"
)

const RPCNamespace = "nitroexecution"

type MaintenanceStatus struct {
	IsRunning bool `json:"isRunning"`
}

type SequencedMsg struct {
	MsgIdx        arbutil.MessageIndex
	MsgWithMeta   arbostypes.MessageWithMetadata
	MsgResult     *MessageResult
	BlockMetadata common.BlockMetadata
}

type MessageResult struct {
	BlockHash common.Hash
	SendRoot  common.Hash
}

type RecordResult struct {
	Pos       arbutil.MessageIndex
	BlockHash common.Hash
	Preimages map[common.Hash][]byte
	UserWasms state.UserWasms
}

// ConsensusSyncData contains sync status information pushed from consensus to execution
type ConsensusSyncData struct {
	Synced          bool
	MaxMessageCount arbutil.MessageIndex
	SyncProgressMap map[string]interface{} // Only populated when !Synced for debugging
	UpdatedAt       time.Time
}

var (
	ErrRetrySequencer                   = errors.New("please retry transaction")
	ExecutionEngineBlockCreationStopped = errors.New("block creation stopped in execution engine")
)

// always needed
type ExecutionClient interface {
	ArbOSVersionGetter

	DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*MessageResult]
	Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo) containers.PromiseInterface[[]*MessageResult]
	HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex]
	ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*MessageResult]
	SetFinalityData(safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}]
	SetConsensusSyncData(syncData *ConsensusSyncData) containers.PromiseInterface[struct{}]
	MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}]

	TriggerMaintenance() containers.PromiseInterface[struct{}]
	ShouldTriggerMaintenance() containers.PromiseInterface[bool]
	MaintenanceStatus() containers.PromiseInterface[*MaintenanceStatus]

	Start(ctx context.Context) error
	StopAndWait()
}

// needed for validators / stakers
type ExecutionRecorder interface {
	RecordBlockCreation(
		pos arbutil.MessageIndex,
		msg *arbostypes.MessageWithMetadata,
		wasmTargets []rawdb.WasmTarget,
	) containers.PromiseInterface[*RecordResult]
	PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}]
}

// ExecutionSequencer is implemented by the execution node and driven by the
// consensus node (the TransactionStreamer). It extends ExecutionClient with the
// operations needed to produce new blocks locally, rather than only digesting
// blocks received from consensus.
//
// The consensus node drives sequencing as a repeated, single-threaded loop and
// must not call these methods concurrently with one another. Each iteration:
//
//  1. StartSequencing produces at most one block and stages it, or returns
//     (nil, wait) when there is nothing to do yet (wait is how long to back off).
//  2. If a block was produced, the consensus node durably stores it in its own
//     database.
//  3. If a block was produced and durably stored, AppendLastSequencedBlock
//     commits that block to the execution node's chain. An append failure is
//     not forwarded to EndSequencing: the message is already durable, so the
//     execution node recovers by re-digesting it (DigestMessage).
//  4. EndSequencing finalizes the turn. It is ALWAYS called after StartSequencing
//     (not only on error), and receives whatever error the consensus node hit
//     while durably storing the block in its own database (nil on success).
//
// Independently of that loop:
//   - EnqueueDelayedMessages may be called at any time to feed delayed (L1-inbox)
//     messages; StartSequencing drains them one per block on delayed turns.
//     NextDelayedMessageNumber reports where to resume feeding them.
//   - ResequenceReorgedMessage is called, per message, while replaying messages
//     that survived a reorg.
//   - Pause / Activate / ForwardTo / IsActive control whether this node sequences
//     locally, is idle, or forwards transactions to another sequencer.
type ExecutionSequencer interface {
	ExecutionClient

	// Pause stops this node from acting as the active sequencer: it drops any
	// forwarder and marks the node inactive. Idempotent.
	Pause()
	// Activate makes this node the active sequencer: it drops any forwarder and
	// marks the node active. Idempotent.
	Activate()
	// IsActive reports whether this node is currently the active sequencer (as
	// opposed to paused or forwarding).
	IsActive() bool
	// ForwardTo configures this node to forward received transactions to the
	// sequencer at url instead of sequencing locally, marking the node inactive.
	// It returns an error if the forwarder cannot be initialized (leaving no
	// forwarder set); forwarding to the current target is a no-op.
	ForwardTo(url string) error

	// StartSequencing runs one sequencing turn and stages, but does not commit,
	// its result. It produces at most one block per call: either from pending
	// regular transactions or from a single pending delayed message, alternating
	// so neither starves the other.
	//
	// It returns a non-nil SequencedMsg when a block was produced, which the
	// consensus node must persist and then commit via AppendLastSequencedBlock.
	// It returns (nil, wait) when nothing was produced (idle or throttled), where
	// wait is how long to back off before calling again.
	//
	// The staged result is owned by the sequencer until EndSequencing is called.
	// StartSequencing is not safe for concurrent use; it must be called from a
	// single serialized loop and each call must be followed by EndSequencing.
	StartSequencing(ctx context.Context) (*SequencedMsg, time.Duration)
	// EndSequencing finalizes the turn started by StartSequencing and must be
	// called exactly once after each StartSequencing, whether or not a block was
	// produced. errWhileSequencing is the error (if any) the consensus node hit
	// while durably persisting the produced block in its own database (an
	// AppendLastSequencedBlock failure is not passed here; see above):
	//   - nil: the block was durably persisted; the sequencer pops the sequenced
	//     delayed message, finalizes nonce state, and returns results to waiting
	//     tx submitters.
	//   - ErrRetrySequencer: the block could not be persisted now; staged regular
	//     transactions are re-queued (or forwarded) for retry.
	//   - any other error: staged regular transactions are failed back to their
	//     submitters with that error.
	// A staged delayed message is left queued for a later turn on any non-nil
	// error. When StartSequencing produced nothing, EndSequencing is a no-op.
	EndSequencing(ctx context.Context, errWhileSequencing error)

	// EnqueueDelayedMessages feeds delayed (L1-inbox) messages to the execution
	// node, assigning them sequential delayed indices starting at firstMsgIdx. It
	// may be called at any time, independently of the StartSequencing loop, and
	// expects messages in contiguous index order (it does not dedupe). The queue
	// is cleared on Reorg and must be refilled by the consensus node.
	EnqueueDelayedMessages(msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64)
	// AppendLastSequencedBlock commits the block staged by the most recent
	// StartSequencing (or ResequenceReorgedMessage) to the execution chain,
	// caching its L1 pricing data. It returns an error if there is no staged
	// block or if the append fails. The staged block is consumed even on
	// failure; recovery is by re-digesting the durably stored message, never by
	// retrying the append.
	AppendLastSequencedBlock() error
	// ResequenceReorgedMessage re-sequences a single message that was sequenced
	// before a reorg, re-applying it on the reorged chain and staging the
	// resulting block (to be committed via AppendLastSequencedBlock). It requires
	// the node to be the active sequencer and handles both a delayed message (only
	// if its delayed index matches the next expected one) and a regular
	// batch-poster message. It returns (nil, nil) when the message should be
	// skipped (e.g. unexpected delayed index or a non-standard message), and
	// ExecutionEngineBlockCreationStopped when block creation is halted, on which
	// the caller should stop resequencing.
	ResequenceReorgedMessage(msg *arbostypes.MessageWithMetadata) (*SequencedMsg, error)
	// NextDelayedMessageNumber returns the index of the next delayed message the
	// execution node expects: one past the last enqueued message, or, if none are
	// queued, one past the last already sequenced into a block. The consensus node
	// uses it to know where to resume EnqueueDelayedMessages.
	NextDelayedMessageNumber() (uint64, error)

	// Synced reports whether the execution node considers itself caught up, based
	// on the most recent sync data pushed by consensus via SetConsensusSyncData.
	// It returns false if no data has been received or the data is stale.
	Synced(ctx context.Context) bool
	// FullSyncProgressMap returns a diagnostic snapshot of sync progress
	// (consensus target vs. execution head) for debugging and health endpoints.
	FullSyncProgressMap(ctx context.Context) map[string]interface{}
}

// needed for batch poster
type ArbOSVersionGetter interface {
	ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64]
}

type FullExecutionClient interface {
	ExecutionClient
	ExecutionSequencer
	ExecutionRecorder
}

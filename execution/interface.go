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

var ErrRetrySequencer = errors.New("please retry transaction")
var ErrSequencerInsertLockTaken = errors.New("insert lock taken")

// always needed
type ExecutionClient interface {
	ArbOSVersionGetter

	DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*MessageResult]
	Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*MessageResult]
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

// needed for sequencer
type ExecutionSequencer interface {
	ExecutionClient
	Pause() containers.PromiseInterface[struct{}]
	Activate() containers.PromiseInterface[struct{}]
	ForwardTo(url string) containers.PromiseInterface[struct{}]
	SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) containers.PromiseInterface[struct{}]
	NextDelayedMessageNumber() containers.PromiseInterface[uint64]
	Synced() containers.PromiseInterface[bool]
	FullSyncProgressMap() containers.PromiseInterface[map[string]interface{}]
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

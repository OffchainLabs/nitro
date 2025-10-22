package executionrpcserver

import (
	"context"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
)

type ExecutionRPCServer struct {
	executionClient execution.ExecutionClient
	// executionSequencer   execution.ExecutionSequencer
	// executionRecorder    execution.ExecutionRecorder
	// executionBatchPoster execution.ExecutionBatchPoster
}

func NewExecutionRpcServer(executionClient execution.ExecutionClient) *ExecutionRPCServer {
	return &ExecutionRPCServer{executionClient}
}

// ExecutionClient methods

func (c *ExecutionRPCServer) DigestMessage(ctx context.Context, msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*consensus.MessageResult, error) {
	return c.executionClient.DigestMessage(msgIdx, msg, msgForPrefetch).Await(ctx)
}

func (c *ExecutionRPCServer) Reorg(ctx context.Context, msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*consensus.MessageResult, error) {
	return c.executionClient.Reorg(msgIdxOfFirstMsgToAdd, newMessages, oldMessages).Await(ctx)
}

func (c *ExecutionRPCServer) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.executionClient.HeadMessageIndex().Await(ctx)
}

func (c *ExecutionRPCServer) ResultAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (*consensus.MessageResult, error) {
	return c.executionClient.ResultAtMessageIndex(msgIdx).Await(ctx)
}

func (c *ExecutionRPCServer) MessageIndexToBlockNumber(ctx context.Context, messageNum arbutil.MessageIndex) (uint64, error) {
	return c.executionClient.MessageIndexToBlockNumber(messageNum).Await(ctx)
}

func (c *ExecutionRPCServer) BlockNumberToMessageIndex(ctx context.Context, blockNum uint64) (arbutil.MessageIndex, error) {
	return c.executionClient.BlockNumberToMessageIndex(blockNum).Await(ctx)
}

func (c *ExecutionRPCServer) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	_, err := c.executionClient.SetFinalityData(safeFinalityData, finalizedFinalityData, validatedFinalityData).Await(ctx)
	return err
}

func (c *ExecutionRPCServer) SetConsensusSyncData(ctx context.Context, syncData *execution.ConsensusSyncData) error {
	_, err := c.executionClient.SetConsensusSyncData(syncData).Await(ctx)
	return err
}

func (c *ExecutionRPCServer) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	_, err := c.executionClient.MarkFeedStart(to).Await(ctx)
	return err
}

func (c *ExecutionRPCServer) TriggerMaintenance(ctx context.Context) error {
	_, err := c.executionClient.TriggerMaintenance().Await(ctx)
	return err
}

func (c *ExecutionRPCServer) ShouldTriggerMaintenance(ctx context.Context) (bool, error) {
	return c.executionClient.ShouldTriggerMaintenance().Await(ctx)
}

func (c *ExecutionRPCServer) MaintenanceStatus(ctx context.Context) (*execution.MaintenanceStatus, error) {
	return c.executionClient.MaintenanceStatus().Await(ctx)
}

// // ExecutionRecorder methods

// func (c *ExecutionRPCServer) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
// 	if c.executionRecorder == nil {
// 		return nil, errors.New("recordBlockCreation method is not available")
// 	}
// 	return c.executionRecorder.RecordBlockCreation(ctx, pos, msg, wasmTargets)
// }

// func (c *ExecutionRPCServer) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
// 	if c.executionRecorder != nil {
// 		c.executionRecorder.MarkValid(pos, resultHash)
// 	}
// }

// func (c *ExecutionRPCServer) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
// 	if c.executionRecorder == nil {
// 		return errors.New("PrepareForRecord method is not available")
// 	}
// 	return c.executionRecorder.PrepareForRecord(ctx, start, end)
// }

// // ExecutionSequencer methods
// func (c *ExecutionRPCServer) Pause() {
// 	if c.executionSequencer != nil {
// 		c.executionSequencer.Pause()
// 	}
// }

// func (c *ExecutionRPCServer) Activate() {
// 	if c.executionSequencer != nil {
// 		c.executionSequencer.Activate()
// 	}
// }

// func (c *ExecutionRPCServer) ForwardTo(url string) error {
// 	if c.executionSequencer == nil {
// 		return errors.New("ForwardTo method is not available")
// 	}
// 	return c.executionSequencer.ForwardTo(url)
// }

// func (c *ExecutionRPCServer) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
// 	if c.executionSequencer == nil {
// 		return errors.New("SequenceDelayedMessage method is not available")
// 	}
// 	return c.executionSequencer.SequenceDelayedMessage(message, delayedSeqNum)
// }

// func (c *ExecutionRPCServer) NextDelayedMessageNumber() (uint64, error) {
// 	if c.executionSequencer == nil {
// 		return 0, errors.New("NextDelayedMessageNumber method is not available")
// 	}
// 	return c.executionSequencer.NextDelayedMessageNumber()
// }

// func (c *ExecutionRPCServer) Synced(ctx context.Context) bool {
// 	if c.executionSequencer == nil {
// 		return false
// 	}
// 	return c.executionSequencer.Synced(ctx)
// }

// func (c *ExecutionRPCServer) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
// 	if c.executionSequencer == nil {
// 		return nil
// 	}
// 	return c.executionSequencer.FullSyncProgressMap(ctx)
// }

// // ExecutionBatchPoster methods
// func (c *ExecutionRPCServer) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) (uint64, error) {
// 	if c.executionBatchPoster == nil {
// 		return 0, errors.New("ArbOSVersionForMessageIndex method is not available")
// 	}
// 	return c.executionBatchPoster.ArbOSVersionForMessageIndex(msgIdx)
// }

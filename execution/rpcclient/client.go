package executionrpcclient

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ExecutionRPCClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewExecutionRpcClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ExecutionRPCClient {
	return &ExecutionRPCClient{
		client: rpcclient.NewRpcClient(config, stack),
	}
}

func (c *ExecutionRPCClient) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	return c.client.Start(ctx)
}

func convertError(err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	if strings.Contains(errStr, execution.ErrRetrySequencer.Error()) {
		return execution.ErrRetrySequencer
	}
	return err
}

// ExecutionClient methods

func (c *ExecutionRPCClient) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*consensus.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*consensus.MessageResult, error) {
		var res consensus.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_digestMessage", msgIdx, msg, msgForPrefetch)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *ExecutionRPCClient) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*consensus.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) ([]*consensus.MessageResult, error) {
		var res []*consensus.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_reorg", msgIdxOfFirstMsgToAdd, newMessages, oldMessages)
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRPCClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_headMessageIndex")
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRPCClient) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*consensus.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*consensus.MessageResult, error) {
		var res *consensus.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_resultAtMessageIndex", msgIdx)
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRPCClient) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_messageIndexToBlockNumber", messageNum)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRPCClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_blockNumberToMessageIndex", blockNum)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRPCClient) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_setFinalityData", safeFinalityData, finalizedFinalityData, validatedFinalityData)
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRPCClient) SetConsensusSyncData(ctx context.Context, syncData *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_setConsensusSyncData", syncData)
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRPCClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_markFeedStart", to)
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRPCClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_triggerMaintenance")
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRPCClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (bool, error) {
		var res bool
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_shouldTriggerMaintenance")
		if err != nil {
			return false, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRPCClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MaintenanceStatus, error) {
		var res *execution.MaintenanceStatus
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_maintenanceStatus")
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

// func (c *ExecutionRPCClient) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
// 	var res *execution.RecordResult
// 	err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_recordBlockCreation", pos, msg, wasmTargets)
// 	if err != nil {
// 		return nil, convertError(err)
// 	}
// 	return res, nil
// }

// func (c *ExecutionRPCClient) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
// 	err := c.client.CallContext(c.GetContext(), nil, execution.RPCNamespace+"_markValid", pos, resultHash)
// 	if err != nil {
// 		log.Error("ExecutionRPCClient errored calling MarkValid", "err", err)
// 	}
// }

// func (c *ExecutionRPCClient) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
// 	err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_prepareForRecord", start, end)
// 	if err != nil {
// 		return convertError(err)
// 	}
// 	return nil
// }

// func (c *ExecutionRPCClient) Pause() {
// 	err := c.client.CallContext(c.GetContext(), nil, execution.RPCNamespace+"_pause")
// 	if err != nil {
// 		log.Error("ExecutionRPCClient errored calling Pause", "err", err)
// 	}
// }

// func (c *ExecutionRPCClient) Activate() {
// 	err := c.client.CallContext(c.GetContext(), nil, execution.RPCNamespace+"_activate")
// 	if err != nil {
// 		log.Error("ExecutionRPCClient errored calling Activate", "err", err)
// 	}
// }

// func (c *ExecutionRPCClient) ForwardTo(url string) error {
// 	err := c.client.CallContext(c.GetContext(), nil, execution.RPCNamespace+"_forwardTo", url)
// 	if err != nil {
// 		return convertError(err)
// 	}
// 	return nil
// }

// func (c *ExecutionRPCClient) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
// 	err := c.client.CallContext(c.GetContext(), nil, execution.RPCNamespace+"_sequenceDelayedMessage", message, delayedSeqNum)
// 	if err != nil {
// 		return convertError(err)
// 	}
// 	return nil
// }

// func (c *ExecutionRPCClient) NextDelayedMessageNumber() (uint64, error) {
// 	var res uint64
// 	err := c.client.CallContext(c.GetContext(), &res, execution.RPCNamespace+"_nextDelayedMessageNumber")
// 	if err != nil {
// 		return 0, convertError(err)
// 	}
// 	return res, nil
// }

// func (c *ExecutionRPCClient) Synced(ctx context.Context) bool {
// 	var res bool
// 	err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_synced")
// 	if err != nil {
// 		log.Error("ExecutionRPCClient errored calling Synced", "err", err)
// 		return false
// 	}
// 	return res
// }

// func (c *ExecutionRPCClient) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
// 	var res map[string]interface{}
// 	err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_fullSyncProgressMap")
// 	if err != nil {
// 		log.Error("ExecutionRPCClient errored calling FullSyncProgressMap", "err", err)
// 		return nil
// 	}
// 	return res
// }

// func (c *ExecutionRPCClient) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) (uint64, error) {
// 	var res uint64
// 	err := c.client.CallContext(c.GetContext(), &res, execution.RPCNamespace+"_arbOSVersionForMessageIndex")
// 	if err != nil {
// 		return 0, convertError(err)
// 	}
// 	return res, nil
// }

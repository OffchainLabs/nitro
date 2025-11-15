package executionrpcclient

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ExecutionRPCClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewExecutionRPCClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ExecutionRPCClient {
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

func (c *ExecutionRPCClient) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_digestMessage", msgIdx, msg, msgForPrefetch)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *ExecutionRPCClient) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) ([]*execution.MessageResult, error) {
		var res []*execution.MessageResult
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
		return res, convertError(err)

	})
}

func (c *ExecutionRPCClient) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res *execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_resultAtMessageIndex", msgIdx)
		return res, convertError(err)
	})
}

func (c *ExecutionRPCClient) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_messageIndexToBlockNumber", messageNum)
		return res, convertError(err)
	})
}

func (c *ExecutionRPCClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_blockNumberToMessageIndex", blockNum)
		return res, convertError(err)
	})
}

func (c *ExecutionRPCClient) SetFinalityData(safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_setFinalityData", safeFinalityData, finalizedFinalityData, validatedFinalityData)
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRPCClient) SetConsensusSyncData(syncData *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
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
		return res, convertError(err)
	})
}

func (c *ExecutionRPCClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MaintenanceStatus, error) {
		var res *execution.MaintenanceStatus
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_maintenanceStatus")
		return res, convertError(err)
	})
}

func (c *ExecutionRPCClient) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_messageIndexToBlockNumber", msgIdx)
		return res, convertError(err)
	})
}

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

type ExecutionRpcClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewExecutionRpcClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ExecutionRpcClient {
	return &ExecutionRpcClient{
		client: rpcclient.NewRpcClient(config, stack),
	}
}

func (c *ExecutionRpcClient) Start(ctx_in context.Context) error {
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

func (c *ExecutionRpcClient) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*consensus.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*consensus.MessageResult, error) {
		var res consensus.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_digestMessage", msgIdx, msg, msgForPrefetch)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *ExecutionRpcClient) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*consensus.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) ([]*consensus.MessageResult, error) {
		var res []*consensus.MessageResult
		err := c.client.CallContext(ctx, res, execution.RPCNamespace+"_reorg", msgIdxOfFirstMsgToAdd, newMessages, oldMessages)
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRpcClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_headMessageIndex")
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRpcClient) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*consensus.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*consensus.MessageResult, error) {
		var res *consensus.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_resultAtMessageIndex", msgIdx)
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRpcClient) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_messageIndexToBlockNumber", messageNum)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRpcClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_blockNumberToMessageIndex", blockNum)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRpcClient) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_setFinalityData", safeFinalityData, finalizedFinalityData, validatedFinalityData)
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRpcClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_markFeedStart", to)
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRpcClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_triggerMaintenance")
		return struct{}{}, convertError(err)
	})
}

func (c *ExecutionRpcClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (bool, error) {
		var res bool
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_shouldTriggerMaintenance")
		if err != nil {
			return false, convertError(err)
		}
		return res, nil
	})
}

func (c *ExecutionRpcClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MaintenanceStatus, error) {
		var res *execution.MaintenanceStatus
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_maintenanceStatus")
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

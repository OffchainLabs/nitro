package consensusclient

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Client struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *Client {
	return &Client{
		client: rpcclient.NewRpcClient(config, stack),
	}
}

func (c *Client) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	return c.client.Start(ctx)
}

func convertError(err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	if strings.Contains(errStr, consensus.ErrSequencerInsertLockTaken.Error()) {
		return consensus.ErrSequencerInsertLockTaken
	}
	return err
}

func (c *Client) FindL1BatchForMessage(message arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_findL1BatchForMessage", message)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *Client) GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getBatchParentChainBlock", seqNum)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *Client) SyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	return stopwaiter.LaunchPromiseThread[map[string]interface{}](c, func(ctx context.Context) (map[string]interface{}, error) {
		var res map[string]interface{}
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_syncProgressMap")
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *Client) SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_syncTargetMessageCount")
		if err != nil {
			return 0, convertError(err)
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) GetSafeMsgCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getSafeMsgCount")
		if err != nil {
			return 0, convertError(err)
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) GetFinalizedMsgCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getFinalizedMsgCount")
		if err != nil {
			return 0, convertError(err)
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, consensus.RPCNamespace+"_writeMessageFromSequencer", pos, msgWithMeta)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) ExpectChosenSequencer() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, consensus.RPCNamespace+"_expectChosenSequencer")
		return struct{}{}, convertError(err)
	})
}

package execclient

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Client struct {
	stopwaiter.StopWaiter
	client *rpc.Client
	config *rpcclient.ClientConfig
	stack  *node.Node
}

func NewClient(config *rpcclient.ClientConfig, stack *node.Node) *Client {
	return &Client{
		config: config,
		stack:  stack,
	}
}

func (c *Client) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	client, err := rpcclient.CreateRPCClient(ctx, c.config, c.stack)
	if err != nil {
		return err
	}
	c.client = client
	return nil
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

func (c *Client) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread[*execution.MessageResult](c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_digestMessage", num, msg)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *Client) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_reorg", count, newMessages, oldMessages)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) HeadMessageNumber() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_headMessageNumber")
		if err != nil {
			return 0, convertError(err)
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) ResultAtPos(pos arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread[*execution.MessageResult](c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_resultAtPos", pos)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *Client) RecordBlockCreation(pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.RecordResult] {
	return stopwaiter.LaunchPromiseThread[*execution.RecordResult](c, func(ctx context.Context) (*execution.RecordResult, error) {
		var res execution.RecordResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_recordBlockCreation", pos, msg)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *Client) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	c.LaunchThread(func(ctx context.Context) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_markValid", pos, resultHash)
		if err != nil && ctx.Err() == nil {
			log.Warn("markValid failed", "err", convertError(err))
		}
	})
}

func (c *Client) PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_prepareForRecord", start, end)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) Pause() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_seqPause")
		return struct{}{}, convertError(err)
	})
}

func (c *Client) Activate() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_seqActivate")
		return struct{}{}, convertError(err)
	})
}

func (c *Client) ForwardTo(url string) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_seqForwardTo", url)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_sequenceDelayedMessage", message, delayedSeqNum)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) Maintenance() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_maintenance")
		return struct{}{}, convertError(err)
	})
}

func (c *Client) NextDelayedMessageNumber() containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_nextDelayedMessageNumber")
		return res, convertError(err)
	})
}

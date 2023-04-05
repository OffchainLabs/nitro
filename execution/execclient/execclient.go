package execclient

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ExecClient struct {
	stopwaiter.StopWaiter
	client    *rpc.Client
	url       string
	jwtSecret []byte
}

func NewValidationClient(url string, jwtSecret []byte) *ExecClient {
	return &ExecClient{
		url:       url,
		jwtSecret: jwtSecret,
	}
}

func (c *ExecClient) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread[*execution.MessageResult](c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_digestMessage", msg)
		if err != nil {
			return nil, err
		}
		return &res, nil
	})
}

func (c *ExecClient) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadata, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_reorg", newMessages, oldMessages)
		return struct{}{}, err
	})
}

func (c *ExecClient) HeadMessageNumber() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_headMessageNumber")
		if err != nil {
			return 0, err
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *ExecClient) ResultAtPos(pos arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread[*execution.MessageResult](c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_resultAtPos", pos)
		if err != nil {
			return nil, err
		}
		return &res, nil
	})
}

func (c *ExecClient) RecordBlockCreation(pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.RecordResult] {
	return stopwaiter.LaunchPromiseThread[*execution.RecordResult](c, func(ctx context.Context) (*execution.RecordResult, error) {
		var res execution.RecordResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_recordBlockCreation", msg)
		if err != nil {
			return nil, err
		}
		return &res, nil
	})
}

func (c *ExecClient) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	c.LaunchThread(func(ctx context.Context) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_markValid", pos, resultHash)
		if err != nil && ctx.Err() == nil {
			log.Warn("markValid failed", "err", err)
		}
	})
}

func (c *ExecClient) PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_prepareForRecord", start, end)
		return struct{}{}, err
	})
}

func (c *ExecClient) Pause() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_seqPause")
		return struct{}{}, err
	})
}

func (c *ExecClient) Activate() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_seqActivate")
		return struct{}{}, err
	})
}

func (c *ExecClient) ForwardTo(url string) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_seqForwardTo", url)
		return struct{}{}, err
	})
}

func (c *ExecClient) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_sequenceDelayedMessage", message, delayedSeqNum)
		return struct{}{}, err
	})
}

func (c *ExecClient) Maintenance() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_maintenance")
		return struct{}{}, err
	})
}

func (c *ExecClient) NextDelayedMessageNumber() containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_nextDelayedMessageNumber")
		return res, err
	})
}

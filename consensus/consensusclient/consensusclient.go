package consensuscient

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Client struct {
	stopwaiter.StopWaiter
	client    *rpc.Client
	url       string
	jwtSecret []byte
}

func NewClient(url string, jwtSecret []byte) *Client {
	return &Client{
		url:       url,
		jwtSecret: jwtSecret,
	}
}

func (c *Client) FetchBatch(batchNum uint64) containers.PromiseInterface[[]byte] {
	return stopwaiter.LaunchPromiseThread[[]byte](c, func(ctx context.Context) ([]byte, error) {
		var res []byte
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_fetchBatch", batchNum)
		if err != nil {
			return nil, err
		}
		return res, nil
	})
}

func (c *Client) FindL1BatchForMessage(message arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_findL1BatchForMessage", message)
		if err != nil {
			return 0, err
		}
		return res, nil
	})
}

func (c *Client) GetBatchL1Block(seqNum uint64) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getBatchL1Block", seqNum)
		if err != nil {
			return 0, err
		}
		return res, nil
	})
}

func (c *Client) SyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	return stopwaiter.LaunchPromiseThread[map[string]interface{}](c, func(ctx context.Context) (map[string]interface{}, error) {
		var res map[string]interface{}
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_syncProgressMap")
		if err != nil {
			return nil, err
		}
		return res, nil
	})
}

func (c *Client) SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_syncTargetMessageCount")
		if err != nil {
			return 0, err
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) GetSafeMsgCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getSafeMsgCount")
		if err != nil {
			return 0, err
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) GetFinalizedMsgCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread[arbutil.MessageIndex](c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getFinalizedMsgCount")
		if err != nil {
			return 0, err
		}
		return arbutil.MessageIndex(res), nil
	})
}

func (c *Client) WriteMessageFromSequencer(pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, consensus.RPCNamespace+"_writeMessageFromSequencer")
		return struct{}{}, err
	})
}

func (c *Client) ExpectChosenSequencer() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, consensus.RPCNamespace+"_expectChosenSequencer")
		return struct{}{}, err
	})
}

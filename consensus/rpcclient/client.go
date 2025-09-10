package consensusrpcclient

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ConsensusRpcClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
}

func NewConsensusRpcClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ConsensusRpcClient {
	return &ConsensusRpcClient{
		client: rpcclient.NewRpcClient(config, stack),
	}
}

func (c *ConsensusRpcClient) Start(ctx_in context.Context) error {
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

func (c *ConsensusRpcClient) FindInboxBatchContainingMessage(message arbutil.MessageIndex) containers.PromiseInterface[consensus.InboxBatch] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (consensus.InboxBatch, error) {
		var res consensus.InboxBatch
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_findInboxBatchContainingMessage", message)
		if err != nil {
			return consensus.InboxBatch{BatchNum: 0, Found: false}, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) GetBatchParentChainBlock(seqNum uint64) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_getBatchParentChainBlock", seqNum)
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) Synced() containers.PromiseInterface[bool] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (bool, error) {
		var res bool
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_synced")
		if err != nil {
			return false, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) FullSyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (map[string]interface{}, error) {
		var res map[string]interface{}
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_fullSyncProgressMap")
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) SyncTargetMessageCount() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_syncTargetMessageCount")
		if err != nil {
			return 0, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) BlockMetadataAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[common.BlockMetadata] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (common.BlockMetadata, error) {
		var res common.BlockMetadata
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_blockMetadataAtMessageIndex", msgIdx)
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) WriteMessageFromSequencer(msgIdx arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult consensus.MessageResult, blockMetadata common.BlockMetadata) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		var res struct{}
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_writeMessageFromSequencer", msgIdx, msgWithMeta, msgResult, blockMetadata)
		if err != nil {
			return struct{}{}, convertError(err)
		}
		return res, nil
	})
}

func (c *ConsensusRpcClient) ExpectChosenSequencer() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		var res struct{}
		err := c.client.CallContext(ctx, &res, consensus.RPCNamespace+"_expectChosenSequencer")
		if err != nil {
			return struct{}{}, convertError(err)
		}
		return res, nil
	})
}

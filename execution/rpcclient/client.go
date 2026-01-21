// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package rpcclient

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
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
	if strings.Contains(errStr, execution.ErrRetrySequencer.Error()) {
		return execution.ErrRetrySequencer
	}
	return err
}

func (c *Client) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_digestMessage", msgIdx, msg, msgForPrefetch)
		if err != nil {
			return nil, convertError(err)
		}
		return &res, nil
	})
}

func (c *Client) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) ([]*execution.MessageResult, error) {
		var res []*execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_reorg", msgIdxOfFirstMsgToAdd, newMessages, oldMessages)
		if err != nil {
			return nil, convertError(err)
		}
		return res, nil
	})
}

func (c *Client) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (arbutil.MessageIndex, error) {
		var res arbutil.MessageIndex
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_headMessageIndex")
		return res, convertError(err)

	})
}

func (c *Client) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MessageResult, error) {
		var res *execution.MessageResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_resultAtMessageIndex", msgIdx)
		return res, convertError(err)
	})
}

func (c *Client) SetFinalityData(safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_setFinalityData", safeFinalityData, finalizedFinalityData, validatedFinalityData)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) SetConsensusSyncData(syncData *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_setConsensusSyncData", syncData)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_markFeedStart", to)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_triggerMaintenance")
		return struct{}{}, convertError(err)
	})
}

func (c *Client) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (bool, error) {
		var res bool
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_shouldTriggerMaintenance")
		return res, convertError(err)
	})
}

func (c *Client) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.MaintenanceStatus, error) {
		var res *execution.MaintenanceStatus
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_maintenanceStatus")
		return res, convertError(err)
	})
}

func (c *Client) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_arbOSVersionForMessageIndex", msgIdx)
		return res, convertError(err)
	})
}

func (c *Client) RecordBlockCreation(pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) containers.PromiseInterface[*execution.RecordResult] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (*execution.RecordResult, error) {
		var res execution.RecordResult
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_recordBlockCreation", pos, msg, wasmTargets)
		return &res, convertError(err)
	})
}

func (c *Client) PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_prepareForRecord", start, end)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) Pause() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_pause")
		return struct{}{}, convertError(err)
	})
}

func (c *Client) Activate() containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_activate")
		return struct{}{}, convertError(err)
	})
}

func (c *Client) ForwardTo(url string) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_forwardTo", url)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, execution.RPCNamespace+"_sequenceDelayedMessage", message, delayedSeqNum)
		return struct{}{}, convertError(err)
	})
}

func (c *Client) NextDelayedMessageNumber() containers.PromiseInterface[uint64] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (uint64, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_nextDelayedMessageNumber")
		return res, convertError(err)
	})
}

func (c *Client) Synced() containers.PromiseInterface[bool] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (bool, error) {
		var res bool
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_synced")
		return res, convertError(err)
	})
}

func (c *Client) FullSyncProgressMap() containers.PromiseInterface[map[string]interface{}] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (map[string]interface{}, error) {
		var res map[string]interface{}
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+"_fullSyncProgressMap")
		return res, convertError(err)
	})
}

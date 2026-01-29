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

func sendRequest[T any](c *Client, method string, args ...any) containers.PromiseInterface[T] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (T, error) {
		var res T
		err := c.client.CallContext(ctx, &res, execution.RPCNamespace+method, args...)
		return res, convertError(err)
	})
}

func (c *Client) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return sendRequest[*execution.MessageResult](c, "_digestMessage", msgIdx, msg, msgForPrefetch)
}

func (c *Client) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	return sendRequest[[]*execution.MessageResult](c, "_reorg", msgIdxOfFirstMsgToAdd, newMessages, oldMessages)
}

func (c *Client) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return sendRequest[arbutil.MessageIndex](c, "_headMessageIndex")
}

func (c *Client) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return sendRequest[*execution.MessageResult](c, "_resultAtMessageIndex", msgIdx)
}

func (c *Client) SetFinalityData(safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return sendRequest[struct{}](c, "_setFinalityData", safeFinalityData, finalizedFinalityData, validatedFinalityData)
}

func (c *Client) SetConsensusSyncData(syncData *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
	return sendRequest[struct{}](c, "_setConsensusSyncData", syncData)
}

func (c *Client) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return sendRequest[struct{}](c, "_markFeedStart", to)
}

func (c *Client) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	return sendRequest[struct{}](c, "_triggerMaintenance")
}

func (c *Client) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return sendRequest[bool](c, "_shouldTriggerMaintenance")
}

func (c *Client) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return sendRequest[*execution.MaintenanceStatus](c, "_maintenanceStatus")
}

func (c *Client) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return sendRequest[uint64](c, "_arbOSVersionForMessageIndex", msgIdx)
}

func (c *Client) RecordBlockCreation(pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) containers.PromiseInterface[*execution.RecordResult] {
	return sendRequest[*execution.RecordResult](c, "_recordBlockCreation", pos, msg, wasmTargets)
}

func (c *Client) PrepareForRecord(start, end arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return sendRequest[struct{}](c, "_prepareForRecord", start, end)
}

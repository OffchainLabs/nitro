// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package rpcserver

import (
	"context"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type Server struct {
	client execution.FullExecutionClient
}

func NewServer(client execution.FullExecutionClient) *Server {
	return &Server{client}
}

func (c *Server) DigestMessage(ctx context.Context, msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	return c.client.DigestMessage(msgIdx, msg, msgForPrefetch).Await(ctx)
}

func (c *Server) Reorg(ctx context.Context, msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	return c.client.Reorg(msgIdxOfFirstMsgToAdd, newMessages, oldMessages).Await(ctx)
}

func (c *Server) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.client.HeadMessageIndex().Await(ctx)
}

func (c *Server) ResultAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (*execution.MessageResult, error) {
	return c.client.ResultAtMessageIndex(msgIdx).Await(ctx)
}

func (c *Server) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	_, err := c.client.SetFinalityData(safeFinalityData, finalizedFinalityData, validatedFinalityData).Await(ctx)
	return err
}

func (c *Server) SetConsensusSyncData(ctx context.Context, syncData *execution.ConsensusSyncData) error {
	_, err := c.client.SetConsensusSyncData(syncData).Await(ctx)
	return err
}

func (c *Server) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	_, err := c.client.MarkFeedStart(to).Await(ctx)
	return err
}

func (c *Server) TriggerMaintenance(ctx context.Context) error {
	_, err := c.client.TriggerMaintenance().Await(ctx)
	return err
}

func (c *Server) ShouldTriggerMaintenance(ctx context.Context) (bool, error) {
	return c.client.ShouldTriggerMaintenance().Await(ctx)
}

func (c *Server) MaintenanceStatus(ctx context.Context) (*execution.MaintenanceStatus, error) {
	return c.client.MaintenanceStatus().Await(ctx)
}

func (c *Server) ArbOSVersionForMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (uint64, error) {
	return c.client.ArbOSVersionForMessageIndex(msgIdx).Await(ctx)
}

func (c *Server) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
	return c.client.RecordBlockCreation(pos, msg, wasmTargets).Await(ctx)
}

func (c *Server) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	_, err := c.client.PrepareForRecord(start, end).Await(ctx)
	return err
}

func (c *Server) Pause(ctx context.Context) error {
	_, err := c.client.Pause().Await(ctx)
	return err
}

func (c *Server) Activate(ctx context.Context) error {
	_, err := c.client.Activate().Await(ctx)
	return err
}

func (c *Server) ForwardTo(ctx context.Context, url string) error {
	_, err := c.client.ForwardTo(url).Await(ctx)
	return err
}

func (c *Server) SequenceDelayedMessage(ctx context.Context, message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	_, err := c.client.SequenceDelayedMessage(message, delayedSeqNum).Await(ctx)
	return err
}

func (c *Server) NextDelayedMessageNumber(ctx context.Context) (uint64, error) {
	return c.client.NextDelayedMessageNumber().Await(ctx)
}

func (c *Server) Synced(ctx context.Context) (bool, error) {
	return c.client.Synced().Await(ctx)
}

func (c *Server) FullSyncProgressMap(ctx context.Context) (map[string]interface{}, error) {
	return c.client.FullSyncProgressMap().Await(ctx)
}

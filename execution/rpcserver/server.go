package rpcserver

import (
	"context"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type Server struct {
	executionClient   execution.ExecutionClient
	executionRecorder execution.ExecutionRecorder
}

func NewServer(executionClient execution.ExecutionClient, executionRecorder execution.ExecutionRecorder) *Server {
	return &Server{executionClient, executionRecorder}
}

func (c *Server) DigestMessage(ctx context.Context, msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	return c.executionClient.DigestMessage(msgIdx, msg, msgForPrefetch).Await(ctx)
}

func (c *Server) Reorg(ctx context.Context, msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	return c.executionClient.Reorg(msgIdxOfFirstMsgToAdd, newMessages, oldMessages).Await(ctx)
}

func (c *Server) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.executionClient.HeadMessageIndex().Await(ctx)
}

func (c *Server) ResultAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (*execution.MessageResult, error) {
	return c.executionClient.ResultAtMessageIndex(msgIdx).Await(ctx)
}

func (c *Server) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	_, err := c.executionClient.SetFinalityData(safeFinalityData, finalizedFinalityData, validatedFinalityData).Await(ctx)
	return err
}

func (c *Server) SetConsensusSyncData(ctx context.Context, syncData *execution.ConsensusSyncData) error {
	_, err := c.executionClient.SetConsensusSyncData(syncData).Await(ctx)
	return err
}

func (c *Server) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	_, err := c.executionClient.MarkFeedStart(to).Await(ctx)
	return err
}

func (c *Server) TriggerMaintenance(ctx context.Context) error {
	_, err := c.executionClient.TriggerMaintenance().Await(ctx)
	return err
}

func (c *Server) ShouldTriggerMaintenance(ctx context.Context) (bool, error) {
	return c.executionClient.ShouldTriggerMaintenance().Await(ctx)
}

func (c *Server) MaintenanceStatus(ctx context.Context) (*execution.MaintenanceStatus, error) {
	return c.executionClient.MaintenanceStatus().Await(ctx)
}

func (c *Server) ArbOSVersionForMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (uint64, error) {
	return c.executionClient.ArbOSVersionForMessageIndex(msgIdx).Await(ctx)
}

func (c *Server) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
	return c.executionRecorder.RecordBlockCreation(pos, msg, wasmTargets).Await(ctx)
}

func (c *Server) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	_, err := c.executionRecorder.PrepareForRecord(start, end).Await(ctx)
	return err
}

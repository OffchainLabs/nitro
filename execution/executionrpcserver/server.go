package executionrpcserver

import (
	"context"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type ExecutionRPCServer struct {
	executionClient execution.ExecutionClient
}

func NewExecutionRPCServer(executionClient execution.ExecutionClient) *ExecutionRPCServer {
	return &ExecutionRPCServer{executionClient}
}

func (c *ExecutionRPCServer) DigestMessage(ctx context.Context, msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*execution.MessageResult, error) {
	return c.executionClient.DigestMessage(msgIdx, msg, msgForPrefetch).Await(ctx)
}

func (c *ExecutionRPCServer) Reorg(ctx context.Context, msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	return c.executionClient.Reorg(msgIdxOfFirstMsgToAdd, newMessages, oldMessages).Await(ctx)
}

func (c *ExecutionRPCServer) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.executionClient.HeadMessageIndex().Await(ctx)
}

func (c *ExecutionRPCServer) ResultAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (*execution.MessageResult, error) {
	return c.executionClient.ResultAtMessageIndex(msgIdx).Await(ctx)
}

func (c *ExecutionRPCServer) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	_, err := c.executionClient.SetFinalityData(safeFinalityData, finalizedFinalityData, validatedFinalityData).Await(ctx)
	return err
}

func (c *ExecutionRPCServer) SetConsensusSyncData(ctx context.Context, syncData *execution.ConsensusSyncData) error {
	_, err := c.executionClient.SetConsensusSyncData(syncData).Await(ctx)
	return err
}

func (c *ExecutionRPCServer) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	_, err := c.executionClient.MarkFeedStart(to).Await(ctx)
	return err
}

func (c *ExecutionRPCServer) TriggerMaintenance(ctx context.Context) error {
	_, err := c.executionClient.TriggerMaintenance().Await(ctx)
	return err
}

func (c *ExecutionRPCServer) ShouldTriggerMaintenance(ctx context.Context) (bool, error) {
	return c.executionClient.ShouldTriggerMaintenance().Await(ctx)
}

func (c *ExecutionRPCServer) MaintenanceStatus(ctx context.Context) (*execution.MaintenanceStatus, error) {
	return c.executionClient.MaintenanceStatus().Await(ctx)
}

func (c *ExecutionRPCServer) ArbOSVersionForMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (uint64, error) {
	return c.executionClient.ArbOSVersionForMessageIndex(msgIdx).Await(ctx)
}

package executionrpcserver

import (
	"context"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
)

type ExecutionRpcServer struct {
	executionClient execution.ExecutionClient
}

func NewExecutionRpcServer(executionClient execution.ExecutionClient) *ExecutionRpcServer {
	return &ExecutionRpcServer{executionClient}
}

// ExecutionClient methods

func (c *ExecutionRpcServer) DigestMessage(ctx context.Context, msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) (*consensus.MessageResult, error) {
	return c.executionClient.DigestMessage(msgIdx, msg, msgForPrefetch).Await(ctx)
}

func (c *ExecutionRpcServer) Reorg(ctx context.Context, msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*consensus.MessageResult, error) {
	return c.executionClient.Reorg(msgIdxOfFirstMsgToAdd, newMessages, oldMessages).Await(ctx)
}

func (c *ExecutionRpcServer) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	return c.executionClient.HeadMessageIndex().Await(ctx)
}

func (c *ExecutionRpcServer) ResultAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (*consensus.MessageResult, error) {
	return c.executionClient.ResultAtMessageIndex(msgIdx).Await(ctx)
}

func (c *ExecutionRpcServer) MessageIndexToBlockNumber(ctx context.Context, messageNum arbutil.MessageIndex) (uint64, error) {
	return c.executionClient.MessageIndexToBlockNumber(messageNum).Await(ctx)
}

func (c *ExecutionRpcServer) BlockNumberToMessageIndex(ctx context.Context, blockNum uint64) (arbutil.MessageIndex, error) {
	return c.executionClient.BlockNumberToMessageIndex(blockNum).Await(ctx)
}

func (c *ExecutionRpcServer) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	_, err := c.executionClient.SetFinalityData(ctx, safeFinalityData, finalizedFinalityData, validatedFinalityData).Await(ctx)
	return err
}

func (c *ExecutionRpcServer) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	_, err := c.executionClient.MarkFeedStart(to).Await(ctx)
	return err
}

func (c *ExecutionRpcServer) TriggerMaintenance(ctx context.Context) error {
	_, err := c.executionClient.TriggerMaintenance().Await(ctx)
	return err
}

func (c *ExecutionRpcServer) ShouldTriggerMaintenance(ctx context.Context) (bool, error) {
	return c.executionClient.ShouldTriggerMaintenance().Await(ctx)
}

func (c *ExecutionRpcServer) MaintenanceStatus(ctx context.Context) (*execution.MaintenanceStatus, error) {
	return c.executionClient.MaintenanceStatus().Await(ctx)
}

package consensusapi

import (
	"context"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
)

type ConsensusAPI struct {
	consensus consensus.FullConsensusClient
}

func NewConsensusAPI(consensus consensus.FullConsensusClient) *ConsensusAPI {
	return &ConsensusAPI{consensus}
}

func (a *ConsensusAPI) FetchBatch(ctx context.Context, batchNum uint64) ([]byte, error) {
	return a.consensus.FetchBatch(batchNum).Await(ctx)
}

func (a *ConsensusAPI) FindL1BatchForMessage(ctx context.Context, message arbutil.MessageIndex) (uint64, error) {
	return a.consensus.FindL1BatchForMessage(message).Await(ctx)
}

func (a *ConsensusAPI) GetBatchParentChainBlock(ctx context.Context, seqNum uint64) (uint64, error) {
	return a.consensus.GetBatchParentChainBlock(seqNum).Await(ctx)
}

func (a *ConsensusAPI) SyncProgressMap(ctx context.Context) (map[string]interface{}, error) {
	return a.consensus.SyncProgressMap().Await(ctx)
}

func (a *ConsensusAPI) SyncTargetMessageCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return a.consensus.SyncTargetMessageCount().Await(ctx)
}

func (a *ConsensusAPI) GetSafeMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return a.consensus.GetSafeMsgCount().Await(ctx)
}

func (a *ConsensusAPI) GetFinalizedMsgCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return a.consensus.GetFinalizedMsgCount().Await(ctx)
}

func (a *ConsensusAPI) WriteMessageFromSequencer(ctx context.Context, pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata) error {
	_, err := a.consensus.WriteMessageFromSequencer(pos, msgWithMeta).Await(ctx)
	return err
}

func (a *ConsensusAPI) ExpectChosenSequencer(ctx context.Context) error {
	_, err := a.consensus.ExpectChosenSequencer().Await(ctx)
	return err
}

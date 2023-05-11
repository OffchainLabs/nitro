package consensusapi

import (
	"context"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
)

type ConsensusAPI struct {
	consensus consensus.FullConsensusClient
}

func NewConsensusAPI(consensus consensus.FullConsensusClient) *ConsensusAPI {
	return &ConsensusAPI{consensus}
}

func (a *ConsensusAPI) FindL1BatchForMessage(ctx context.Context, message arbutil.MessageIndex) (uint64, error) {
	return a.consensus.FindL1BatchForMessage(message).Await(ctx)
}

func (a *ConsensusAPI) GetBatchL1Block(ctx context.Context, seqNum uint64) (uint64, error) {
	return a.consensus.GetBatchL1Block(seqNum).Await(ctx)
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

func (a *ConsensusAPI) WriteMessageFromSequencer(ctx context.Context, pos arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, result execution.MessageResult) error {
	_, err := a.consensus.WriteMessageFromSequencer(pos, msgWithMeta, result).Await(ctx)
	return err
}

func (a *ConsensusAPI) ExpectChosenSequencer(ctx context.Context) error {
	_, err := a.consensus.ExpectChosenSequencer().Await(ctx)
	return err
}

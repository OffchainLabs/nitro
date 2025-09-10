package consensusrpcserver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
)

type ConsensusRpcServer struct {
	consensus consensus.FullConsensusClient
}

func NewConsensusRpcServer(consensus consensus.FullConsensusClient) *ConsensusRpcServer {
	return &ConsensusRpcServer{consensus}
}

func (a *ConsensusRpcServer) FindInboxBatchContainingMessage(ctx context.Context, message arbutil.MessageIndex) (consensus.InboxBatch, error) {
	return a.consensus.FindInboxBatchContainingMessage(message).Await(ctx)
}

func (a *ConsensusRpcServer) GetBatchParentChainBlock(ctx context.Context, seqNum uint64) (uint64, error) {
	return a.consensus.GetBatchParentChainBlock(seqNum).Await(ctx)
}

func (a *ConsensusRpcServer) Synced(ctx context.Context) (bool, error) {
	return a.consensus.Synced().Await(ctx)
}

func (a *ConsensusRpcServer) FullSyncProgressMap(ctx context.Context) (map[string]interface{}, error) {
	return a.consensus.FullSyncProgressMap().Await(ctx)
}

func (a *ConsensusRpcServer) SyncTargetMessageCount(ctx context.Context) (arbutil.MessageIndex, error) {
	return a.consensus.SyncTargetMessageCount().Await(ctx)
}

func (a *ConsensusRpcServer) BlockMetadataAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (common.BlockMetadata, error) {
	return a.consensus.BlockMetadataAtMessageIndex(msgIdx).Await(ctx)
}

func (a *ConsensusRpcServer) WriteMessageFromSequencer(ctx context.Context, msgIdx arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult consensus.MessageResult, blockMetadata common.BlockMetadata) error {
	_, err := a.consensus.WriteMessageFromSequencer(msgIdx, msgWithMeta, msgResult, blockMetadata).Await(ctx)
	return err
}

func (a *ConsensusRpcServer) ExpectChosenSequencer(ctx context.Context) error {
	_, err := a.consensus.ExpectChosenSequencer().Await(ctx)
	return err
}

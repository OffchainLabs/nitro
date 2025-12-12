package consensusrpcserver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/execution"
)

type ConsensusRPCServer struct {
	consensus consensus.FullConsensusClient
}

func NewConsensusRPCServer(consensus consensus.FullConsensusClient) *ConsensusRPCServer {
	return &ConsensusRPCServer{consensus}
}

func (a *ConsensusRPCServer) FindInboxBatchContainingMessage(ctx context.Context, message arbutil.MessageIndex) (consensus.InboxBatch, error) {
	return a.consensus.FindInboxBatchContainingMessage(message).Await(ctx)
}

func (a *ConsensusRPCServer) GetBatchParentChainBlock(ctx context.Context, seqNum uint64) (uint64, error) {
	return a.consensus.GetBatchParentChainBlock(seqNum).Await(ctx)
}

func (a *ConsensusRPCServer) BlockMetadataAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (common.BlockMetadata, error) {
	return a.consensus.BlockMetadataAtMessageIndex(msgIdx).Await(ctx)
}

func (a *ConsensusRPCServer) WriteMessageFromSequencer(ctx context.Context, msgIdx arbutil.MessageIndex, msgWithMeta arbostypes.MessageWithMetadata, msgResult execution.MessageResult, blockMetadata common.BlockMetadata) error {
	_, err := a.consensus.WriteMessageFromSequencer(msgIdx, msgWithMeta, msgResult, blockMetadata).Await(ctx)
	return err
}

func (a *ConsensusRPCServer) ExpectChosenSequencer(ctx context.Context) error {
	_, err := a.consensus.ExpectChosenSequencer().Await(ctx)
	return err
}

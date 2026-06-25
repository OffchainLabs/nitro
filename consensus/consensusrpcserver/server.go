// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package consensusrpcserver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
)

type ConsensusRPCServer struct {
	consensus consensus.FullConsensusClient
}

func NewConsensusRPCServer(consensus consensus.FullConsensusClient) *ConsensusRPCServer {
	return &ConsensusRPCServer{consensus}
}

func (a *ConsensusRPCServer) GetL1Confirmations(ctx context.Context, msgIdx arbutil.MessageIndex) (uint64, error) {
	return a.consensus.GetL1Confirmations(msgIdx).Await(ctx)
}

func (a *ConsensusRPCServer) FindBatchContainingMessage(ctx context.Context, msgIdx arbutil.MessageIndex) (uint64, error) {
	return a.consensus.FindBatchContainingMessage(msgIdx).Await(ctx)
}

func (a *ConsensusRPCServer) BlockMetadataAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (common.BlockMetadata, error) {
	return a.consensus.BlockMetadataAtMessageIndex(msgIdx).Await(ctx)
}

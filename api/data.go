package api

import (
	"context"
	"github.com/ethereum/go-ethereum/common"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

type EdgesProvider interface {
	GetHonestEdges() []protocol.SpecEdge
	GetEdges(ctx context.Context) ([]protocol.SpecEdge, error)
	GetEdge(ctx context.Context, hash common.Hash) (protocol.SpecEdge, error)
}

type AssertionsProvider interface {
	ReadAssertionCreationInfo(context.Context, protocol.AssertionHash) (*protocol.AssertionCreatedInfo, error)
	LatestCreatedAssertionHashes(ctx context.Context) ([]protocol.AssertionHash, error)
}

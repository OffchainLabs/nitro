package api

import (
	"context"
	"github.com/ethereum/go-ethereum/common"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"
)

type EdgesProvider interface {
	GetHonestEdges() []protocol.SpecEdge
	GetEdges(ctx context.Context) ([]protocol.SpecEdge, error)
	GetEdge(ctx context.Context, hash common.Hash) (protocol.SpecEdge, error)
	GetHonestConfirmableEdges(ctx context.Context) (map[string][]protocol.SpecEdge, error)
	ComputeHonestPathTimer(ctx context.Context, topLevelAssertionHash protocol.AssertionHash, edgeId protocol.EdgeId) (challengetree.PathTimer, challengetree.HonestAncestors, []challengetree.EdgeLocalTimer, error)
}

type AssertionsProvider interface {
	ReadAssertionCreationInfo(context.Context, protocol.AssertionHash) (*protocol.AssertionCreatedInfo, error)
	LatestCreatedAssertionHashes(ctx context.Context) ([]protocol.AssertionHash, error)
}

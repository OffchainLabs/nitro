package api

import (
	"context"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

type EdgesProvider interface {
	GetEdges() []protocol.SpecEdge
}

type AssertionsProvider interface {
	ReadAssertionCreationInfo(context.Context, protocol.AssertionHash) (*protocol.AssertionCreatedInfo, error)
	LatestCreatedAssertionHashes(ctx context.Context) ([]protocol.AssertionHash, error)
}

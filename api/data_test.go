package api_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	watcher "github.com/OffchainLabs/bold/challenge-manager/chain-watcher"
	challengetree "github.com/OffchainLabs/bold/challenge-manager/challenge-tree"

	"github.com/ethereum/go-ethereum/common"
)

var _ = api.EdgesProvider(&FakeEdgesProvider{})
var _ = api.AssertionsProvider(&FakeAssertionProvider{})

type FakeEdgesProvider struct {
	Edges []protocol.SpecEdge
}

func (f *FakeEdgesProvider) GetHonestEdges() []protocol.SpecEdge {
	return f.Edges
}

func (f *FakeEdgesProvider) GetEdges(ctx context.Context) ([]protocol.SpecEdge, error) {
	return f.Edges, nil
}

func (f *FakeEdgesProvider) GetEdge(ctx context.Context, edgeId common.Hash) (protocol.SpecEdge, error) {
	for _, e := range f.Edges {
		if e.Id().Hash == edgeId {
			return e, nil
		}
	}
	return nil, fmt.Errorf("no edge found with id %#x", edgeId)
}

func (f *FakeEdgesProvider) GetHonestConfirmableEdges(ctx context.Context) (map[string][]protocol.SpecEdge, error) {
	honestConfirmableEdges := make(map[string][]protocol.SpecEdge)
	honestConfirmableEdges[watcher.ConfirmableByTimer] = f.Edges
	return honestConfirmableEdges, nil
}

func (f *FakeEdgesProvider) ComputeHonestPathTimer(ctx context.Context, topLevelAssertionHash protocol.AssertionHash, edgeId protocol.EdgeId) (challengetree.PathTimer, challengetree.HonestAncestors, []challengetree.EdgeLocalTimer, error) {
	return 0, nil, nil, nil
}

type FakeAssertionProvider struct {
	Hashes                 []protocol.AssertionHash
	AssertionCreationInfos []*protocol.AssertionCreatedInfo
}

func (f *FakeAssertionProvider) ReadAssertionCreationInfo(ctx context.Context, ah protocol.AssertionHash) (*protocol.AssertionCreatedInfo, error) {
	if len(f.AssertionCreationInfos) == 0 {
		return nil, errors.New("no mock responses left")
	}
	r := f.AssertionCreationInfos[0]
	f.AssertionCreationInfos = f.AssertionCreationInfos[1:]
	return r, nil
}

func (f *FakeAssertionProvider) LatestCreatedAssertionHashes(ctx context.Context) ([]protocol.AssertionHash, error) {
	return f.Hashes, nil
}

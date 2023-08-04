package api_test

import (
	"context"
	"errors"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

var _ = api.EdgesProvider(&FakeEdgesProvider{})
var _ = api.AssertionsProvider(&FakeAssertionProvider{})

type FakeEdgesProvider struct {
	Edges []protocol.SpecEdge
}

func (f *FakeEdgesProvider) GetEdges() []protocol.SpecEdge {
	return f.Edges
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

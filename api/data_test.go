package api_test

import (
	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

var _ = api.DataAccessor(&FakeDataAccessor{})

type FakeDataAccessor struct {
	Edges []protocol.SpecEdge
}

func (f *FakeDataAccessor) GetEdges() []protocol.SpecEdge {
	return f.Edges
}

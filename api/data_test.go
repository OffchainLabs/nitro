package api_test

import (
	"github.com/OffchainLabs/challenge-protocol-v2/api"
	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
)

var _ = api.DataAccessor(&FakeDataAccessor{})

type FakeDataAccessor struct {
	Edges []protocol.SpecEdge
}

func (f *FakeDataAccessor) GetEdges() []protocol.SpecEdge {
	return f.Edges
}

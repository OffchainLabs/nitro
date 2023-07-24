package api

import (
	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
)

type DataAccessor interface {
	GetEdges() []protocol.SpecEdge
}

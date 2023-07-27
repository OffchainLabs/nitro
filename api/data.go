package api

import (
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

type DataAccessor interface {
	GetEdges() []protocol.SpecEdge
}

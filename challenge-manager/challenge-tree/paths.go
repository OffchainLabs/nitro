package challengetree

import (
	"context"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
)

// Keep track of the (distance, path weight) of
// an honest edge to all its essential refinements or root node.
type essentialNode struct {
	id         protocol.EdgeId
	distance   uint64
	pathWeight uint64
}

// Function invariant: the list of essential nodes are honest ancestors of the
// specified node.
// TODO: Perhaps we should not let anyone call this with an arbitrary edge id?
func (ht *RoyalChallengeTree) UpdatePathWeightsToEssentialNodes(
	ctx context.Context, node protocol.EdgeId,
) {
	essentialNodes, ok := ht.pathWeightsToEssentialNodes[node]
	if !ok {
		return
	}
}

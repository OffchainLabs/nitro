package challengetree

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/pkg/errors"
)

type ComputePathWeightArgs struct {
	Child    protocol.EdgeId
	Ancestor protocol.EdgeId
	BlockNum uint64
}

// ComputePathWeight from a child edge to a specified ancestor edge. A weight is the sum of the local timers
// of all edges along the path.
//
// Invariant: assumes ComputeAncestors returns a list of ancestors ordered from child to parent,
// not including the edge id we are querying ancestors for.
func (ht *RoyalChallengeTree) ComputePathWeight(
	ctx context.Context,
	args ComputePathWeightArgs,
) (uint64, error) {
	child, ok := ht.edges.TryGet(args.Child)
	if !ok {
		return 0, fmt.Errorf("child edge not yet tracked %#x", args.Child.Hash)
	}
	if !ht.edges.Has(args.Ancestor) {
		return 0, fmt.Errorf("ancestor not yet tracked %#x", args.Ancestor.Hash)
	}
	localTimer, err := ht.LocalTimer(child, args.BlockNum)
	if err != nil {
		return 0, err
	}
	if args.Child == args.Ancestor {
		return localTimer, nil
	}
	ancestors, err := ht.ComputeAncestors(ctx, args.Child, args.BlockNum)
	if err != nil {
		return 0, err
	}
	pathWeight := localTimer
	found := false
	for _, an := range ancestors {
		localTimer, err := ht.LocalTimer(an, args.BlockNum)
		if err != nil {
			return 0, err
		}
		pathWeight += localTimer
		if an.Id() == args.Ancestor {
			found = true
			break
		}
	}
	if !found {
		return 0, errors.New("expected ancestor not found in computed ancestors list")
	}
	return pathWeight, nil
}

func (ht *RoyalChallengeTree) isClaimedEdge(ctx context.Context, edge protocol.ReadOnlyEdge) (bool, protocol.ReadOnlyEdge) {
	if isProofNode(ctx, edge) {
		return false, nil
	}
	if !hasLengthOne(edge) {
		return false, nil
	}
	// Note: the specification requires that the claiming edge is correctly constructed.
	// This is not checked here, because the honest validator only tracks
	// essential edges as an invariant.
	claimingEdge, ok := ht.findClaimingEdge(ctx, edge.Id())
	if !ok {
		return false, nil
	}
	return true, claimingEdge
}

// Proof nodes are nodes that have length one at the lowest challenge level.
func isProofNode(ctx context.Context, edge protocol.ReadOnlyEdge) bool {
	isSmallStep := edge.GetChallengeLevel() == protocol.ChallengeLevel(edge.GetTotalChallengeLevels(ctx)-1)
	return isSmallStep && hasLengthOne(edge)
}

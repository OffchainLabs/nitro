package challengetree

import (
	"context"
	"math"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/ethereum/go-ethereum/common"
)

func (ht *RoyalChallengeTree) ComputeRootInheritedTimer(
	ctx context.Context,
	challengedAssertionHash protocol.AssertionHash,
	blockNum uint64,
) (protocol.InheritedTimer, error) {
	royalRootEdge, err := ht.BlockChallengeRootEdge(ctx)
	if err != nil {
		return 0, err
	}
	assertionUnrivaledBlocks, err := ht.metadataReader.AssertionUnrivaledBlocks(
		ctx,
		protocol.AssertionHash{
			Hash: common.Hash(royalRootEdge.ClaimId().Unwrap()),
		},
	)
	if err != nil {
		return 0, err
	}
	inheritedTimer, err := ht.recursiveInheritedTimerCompute(ctx, royalRootEdge.Id(), blockNum)
	if err != nil {
		return 0, err
	}
	return saturatingSum(inheritedTimer, protocol.InheritedTimer(assertionUnrivaledBlocks)), nil
}

func (ht *RoyalChallengeTree) recursiveInheritedTimerCompute(
	ctx context.Context,
	edgeId protocol.EdgeId,
	blockNum uint64,
) (protocol.InheritedTimer, error) {
	edge, ok := ht.edges.TryGet(edgeId)
	if !ok {
		return 0, nil
	}
	status, err := edge.Status(ctx)
	if err != nil {
		return 0, err
	}
	if isOneStepProven(ctx, edge, status) {
		return math.MaxUint64, nil
	}
	localTimer, err := ht.LocalTimer(edge, blockNum)
	if err != nil {
		return 0, err
	}
	// If length one, find the edge that claims it,
	// compute the recursive timer for it. If the onchain is bigger, return the onchain here.
	if hasLengthOne(edge) {
		onchainTimer, innerErr := edge.InheritedTimer(ctx)
		if innerErr != nil {
			return 0, innerErr
		}
		claimingEdgeTimer := protocol.InheritedTimer(0)
		claimingEdge, ok := ht.findClaimingEdge(ctx, edge.Id())
		if ok {
			claimingEdgeTimer, innerErr = ht.recursiveInheritedTimerCompute(
				ctx,
				claimingEdge.Id(),
				blockNum,
			)
			if innerErr != nil {
				return 0, innerErr
			}
		}
		claimedEdgeInheritedTimer := saturatingSum(protocol.InheritedTimer(localTimer), claimingEdgeTimer)
		if onchainTimer > claimedEdgeInheritedTimer {
			return onchainTimer, nil
		}
		return claimedEdgeInheritedTimer, nil
	}
	hasChildren, err := edge.HasChildren(ctx)
	if err != nil {
		return 0, err
	}
	if !hasChildren {
		return protocol.InheritedTimer(localTimer), nil
	}
	lowerChildId, err := edge.LowerChild(ctx)
	if err != nil {
		return 0, err
	}
	upperChildId, err := edge.UpperChild(ctx)
	if err != nil {
		return 0, err
	}
	lowerChildTimer, err := ht.recursiveInheritedTimerCompute(ctx, lowerChildId.Unwrap(), blockNum)
	if err != nil {
		return 0, err
	}
	upperChildTimer, err := ht.recursiveInheritedTimerCompute(ctx, upperChildId.Unwrap(), blockNum)
	if err != nil {
		return 0, err
	}
	minTimer := lowerChildTimer
	if upperChildTimer < lowerChildTimer {
		minTimer = upperChildTimer
	}
	return saturatingSum(protocol.InheritedTimer(localTimer), minTimer), nil
}

func IsClaimingAnEdge(edge protocol.ReadOnlyEdge) bool {
	return edge.ClaimId().IsSome() && edge.GetChallengeLevel() != protocol.NewBlockChallengeLevel()
}

func hasLengthOne(edge protocol.ReadOnlyEdge) bool {
	startHeight, _ := edge.StartCommitment()
	endHeight, _ := edge.EndCommitment()
	return endHeight-startHeight == 1
}

func isOneStepProven(
	ctx context.Context, edge protocol.ReadOnlyEdge, status protocol.EdgeStatus,
) bool {
	isSmallStep := edge.GetChallengeLevel() == protocol.ChallengeLevel(edge.GetTotalChallengeLevels(ctx)-1)
	return isSmallStep && status == protocol.EdgeConfirmed && hasLengthOne(edge)
}

func saturatingSum(a, b protocol.InheritedTimer) protocol.InheritedTimer {
	if math.MaxUint64-a < b {
		return math.MaxUint64
	}
	return a + b
}

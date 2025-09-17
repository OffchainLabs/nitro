// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package challengetree

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/pkg/errors"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers"
	"github.com/offchainlabs/nitro/bold/containers/option"
)

type ComputePathWeightArgs struct {
	Child    protocol.EdgeId
	Ancestor protocol.EdgeId
	BlockNum uint64
}

var ErrChildrenNotYetSeen = errors.New("child not yet tracked")

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
	localTimer, err := ht.LocalTimer(ctx, child, args.BlockNum)
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
		localTimer, err := ht.LocalTimer(ctx, an, args.BlockNum)
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

type EssentialPath []protocol.EdgeId

type IsConfirmableArgs struct {
	EssentialEdge         protocol.EdgeId
	ConfirmationThreshold uint64
	BlockNum              uint64
}

// Find all the paths down from an essential edge, and
// compute the local timer of each edge along the path. This is
// a recursive computation that goes down the tree rooted at the essential
// edge and ends once it finds edges that either do not have children,
// or are terminal edges that end in children that are incorrectly constructed
// or non-essential.
//
// After the paths are computed, we then compute the path weight of each
// and if the min element of this list has a weight >= the confirmation threshold,
// the essential edge is then confirmable.
//
// Note: the specified argument essential edge must indeed be essential, otherwise,
// this function will error.
func (ht *RoyalChallengeTree) IsConfirmableEssentialEdge(
	ctx context.Context,
	args IsConfirmableArgs,
) (bool, []EssentialPath, uint64, error) {
	essentialEdge, ok := ht.edges.TryGet(args.EssentialEdge)
	if !ok {
		return false, nil, 0, fmt.Errorf("essential edge not found")
	}
	if essentialEdge.ClaimId().IsNone() {
		return false, nil, 0, fmt.Errorf("specified input argument %#x is not essential", args.EssentialEdge.Hash)
	}
	essentialPaths, essentialTimers, err := ht.findEssentialPaths(
		ctx,
		essentialEdge,
		args.BlockNum,
	)
	if err != nil {
		return false, nil, 0, err
	}
	if len(essentialPaths) == 0 || len(essentialTimers) == 0 {
		return false, nil, 0, fmt.Errorf("no essential paths found")
	}
	// An essential edge is confirmable if all of its essential paths
	// down the tree have a path weight >= the confirmation threshold.
	// To do this, we compute the path weight of each path
	// and find the minimum.
	// Then, it is sufficient to check that the minimum is
	// greater than or equal to the confirmation threshold.
	minWeight := uint64(math.MaxUint64)
	for _, timers := range essentialTimers {
		pathWeight := uint64(0)
		for _, timer := range timers {
			pathWeight = saturatingUAdd(pathWeight, timer)
		}
		if pathWeight < minWeight {
			minWeight = pathWeight
		}
	}
	allEssentialPathsConfirmable := minWeight >= args.ConfirmationThreshold
	return allEssentialPathsConfirmable, essentialPaths, minWeight, nil
}

type essentialLocalTimers []uint64

// Use a depth-first-search approach (DFS) to gather the
// essential branches of the protocol graph. We manage our own
// visitor stack to avoid recursion.
//
// Invariant: the input edge must be essential.
func (ht *RoyalChallengeTree) findEssentialPaths(
	ctx context.Context,
	essentialEdge protocol.ReadOnlyEdge,
	blockNum uint64,
) ([]EssentialPath, []essentialLocalTimers, error) {
	allPaths := make([]EssentialPath, 0)
	allTimers := make([]essentialLocalTimers, 0)

	type visited struct {
		essentialEdge protocol.ReadOnlyEdge
		path          EssentialPath
		localTimers   essentialLocalTimers
	}
	stack := newStack[*visited]()

	localTimer, err := ht.LocalTimer(ctx, essentialEdge, blockNum)
	if err != nil {
		return nil, nil, err
	}

	stack.push(&visited{
		essentialEdge: essentialEdge,
		path:          EssentialPath{essentialEdge.Id()},
		localTimers:   essentialLocalTimers{localTimer},
	})

	for stack.len() > 0 {
		curr := stack.pop().Unwrap()
		currentEdge, currentTimers, path := curr.essentialEdge, curr.localTimers, curr.path
		isClaimedEdge, claimingEdge := ht.isClaimedEdge(ctx, currentEdge)

		hasChildren, err := currentEdge.HasChildren(ctx)
		if err != nil {
			return nil, nil, err
		}
		if hasChildren {
			lowerChildIdOpt, err := currentEdge.LowerChild(ctx)
			if err != nil {
				return nil, nil, err
			}
			upperChildIdOpt, err := currentEdge.UpperChild(ctx)
			if err != nil {
				return nil, nil, err
			}
			lowerChildId, upperChildId := lowerChildIdOpt.Unwrap(), upperChildIdOpt.Unwrap()
			lowerChild, ok := ht.edges.TryGet(lowerChildId)
			if !ok {
				return nil, nil, errors.Wrap(ErrChildrenNotYetSeen, "lower child")
			}
			upperChild, ok := ht.edges.TryGet(upperChildId)
			if !ok {
				return nil, nil, errors.Wrap(ErrChildrenNotYetSeen, "upper child")
			}
			lowerTimer, err := ht.LocalTimer(ctx, lowerChild, blockNum)
			if err != nil {
				return nil, nil, err
			}
			upperTimer, err := ht.LocalTimer(ctx, upperChild, blockNum)
			if err != nil {
				return nil, nil, err
			}
			lowerPath := append(slices.Clone(path), lowerChildId)
			upperPath := append(slices.Clone(path), upperChildId)
			lowerTimers := append(slices.Clone(currentTimers), lowerTimer)
			upperTimers := append(slices.Clone(currentTimers), upperTimer)
			stack.push(&visited{
				essentialEdge: lowerChild,
				path:          lowerPath,
				localTimers:   lowerTimers,
			})
			stack.push(&visited{
				essentialEdge: upperChild,
				path:          upperPath,
				localTimers:   upperTimers,
			})
			continue
		} else if isClaimedEdge {
			// Figure out if the edge is a terminal edge that has a refinement, in which
			// case we need to continue the search down the next challenge level,
			claimingEdgeTimer, err := ht.LocalTimer(ctx, claimingEdge, blockNum)
			if err != nil {
				return nil, nil, err
			}
			claimingPath := append(slices.Clone(path), claimingEdge.Id())
			claimingTimers := append(slices.Clone(currentTimers), claimingEdgeTimer)
			stack.push(&visited{
				essentialEdge: claimingEdge,
				path:          claimingPath,
				localTimers:   claimingTimers,
			})
			continue
		}

		// Otherwise, the edge is a qualified leaf and we can push to the list of paths
		// and all the timers of the path.
		// Onchain actions expect ordered paths from leaf to root, so we
		// preserve that ordering to make it easier for callers to use this data.
		containers.Reverse(path)
		containers.Reverse(currentTimers)
		allPaths = append(allPaths, path)
		allTimers = append(allTimers, currentTimers)
	}
	return allPaths, allTimers, nil
}

func (ht *RoyalChallengeTree) isClaimedEdge(ctx context.Context, edge protocol.ReadOnlyEdge) (bool, protocol.ReadOnlyEdge) {
	if isProofEdge(ctx, edge) {
		return false, nil
	}
	if !hasLengthOne(edge) {
		return false, nil
	}
	// Note: the specification requires that the claiming edge is correctly constructed.
	// This is not checked here, because the honest validator only tracks
	// essential edges as an invariant.
	claimingEdge, ok := ht.findClaimingEdge(edge.Id())
	if !ok {
		return false, nil
	}
	return true, claimingEdge
}

func IsClaimingAnEdge(edge protocol.ReadOnlyEdge) bool {
	return edge.ClaimId().IsSome() && edge.GetChallengeLevel() != protocol.NewBlockChallengeLevel()
}

func hasLengthOne(edge protocol.ReadOnlyEdge) bool {
	startHeight, _ := edge.StartCommitment()
	endHeight, _ := edge.EndCommitment()
	return endHeight-startHeight == 1
}

// Proof edges are edges that have length one at the lowest challenge level.
func isProofEdge(ctx context.Context, edge protocol.ReadOnlyEdge) bool {
	isSmallStep := edge.GetChallengeLevel() == protocol.ChallengeLevel(edge.GetTotalChallengeLevels(ctx)-1)
	return isSmallStep && hasLengthOne(edge)
}

type stack[T any] struct {
	dll *list.List
}

func newStack[T any]() *stack[T] {
	return &stack[T]{dll: list.New()}
}

func (s *stack[T]) len() int {
	return s.dll.Len()
}

func (s *stack[T]) push(x T) {
	s.dll.PushBack(x)
}

func (s *stack[T]) pop() option.Option[T] {
	if s.dll.Len() == 0 {
		return option.None[T]()
	}
	tail := s.dll.Back()
	val := tail.Value
	s.dll.Remove(tail)
	// nolint:errcheck
	return option.Some(val.(T))
}

type unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func saturatingUAdd[T unsigned](a, b T) T {
	sum := a + b
	if sum < a || sum < b {
		sum = ^T(0)
	}
	return sum
}

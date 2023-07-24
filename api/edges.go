package api

import (
	"context"
	"fmt"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/errgroup"
)

type Edge struct {
	ID                  common.Hash            `json:"id"`
	Type                string                 `json:"type"`
	StartCommitment     *Commitment            `json:"startCommitment"`
	EndCommitment       *Commitment            `json:"endCommitment"`
	CreatedAtBlock      uint64                 `json:"createdAtBlock"`
	MutualID            common.Hash            `json:"mutualId"`
	OriginID            common.Hash            `json:"originId"`
	ClaimID             common.Hash            `json:"claimId"`
	HasChildren         bool                   `json:"hasChildren"`
	LowerChildID        common.Hash            `json:"lowerChildId"`
	UpperChildID        common.Hash            `json:"upperChildId"`
	MiniStaker          common.Address         `json:"miniStaker"`
	AssertionHash       common.Hash            `json:"assertionHash"`
	TimeUnrivaled       uint64                 `json:"timeUnrivaled"`
	HasRival            bool                   `json:"hasRival"`
	Status              string                 `json:"status"`
	HasLengthOneRival   bool                   `json:"hasLengthOneRival"`
	TopLevelClaimHeight protocol.OriginHeights `json:"topLevelClaimHeight"`

	// Validator's point of view
	// IsHonest bool `json:"isHonest"`
	// AgreesWithStartCommitment `json:"agreesWithStartCommitment"`
}

type Commitment struct {
	Height uint64      `json:"height"`
	Hash   common.Hash `json:"hash"`
}

func convertSpecEdgeEdgesToEdges(ctx context.Context, e []protocol.SpecEdge) ([]*Edge, error) {
	// Convert concurrently as some of the underlying methods are API calls.
	eg, ctx := errgroup.WithContext(ctx)

	edges := make([]*Edge, len(e))
	for i, edge := range e {
		index := i
		ee := edge

		eg.Go(func() (err error) {
			edges[index], err = convertSpecEdgeEdgeToEdge(ctx, ee)
			return
		})
	}
	return edges, eg.Wait()
}

func convertSpecEdgeEdgeToEdge(ctx context.Context, e protocol.SpecEdge) (*Edge, error) {
	edge := &Edge{
		ID:              common.Hash(e.Id()),
		Type:            e.GetType().String(),
		StartCommitment: toCommitment(e.StartCommitment),
		EndCommitment:   toCommitment(e.EndCommitment),
		MutualID:        common.Hash(e.MutualId()),
		OriginID:        common.Hash(e.OriginId()),
		ClaimID: func() common.Hash {
			if !e.ClaimId().IsNone() {
				return common.Hash(e.ClaimId().Unwrap())
			}
			return common.Hash{}
		}(),
		MiniStaker: func() common.Address {
			if !e.MiniStaker().IsNone() {
				return common.Address(e.MiniStaker().Unwrap())
			}
			return common.Address{}
		}(),
		CreatedAtBlock: func() uint64 {
			cab, err := e.CreatedAtBlock()
			if err != nil {
				return 0
			}
			return cab
		}(),
	}

	// The following methods include calls to the backend, so we run them concurrently.
	// Note: No rate limiting currently in place.
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		hasChildren, err := e.HasChildren(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge children: %w", err)
		}
		edge.HasChildren = hasChildren
		return nil
	})

	eg.Go(func() error {
		lowerChild, err := e.LowerChild(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge lower child: %w", err)
		}
		if !lowerChild.IsNone() {
			edge.LowerChildID = common.Hash(lowerChild.Unwrap())
		}
		return nil
	})

	eg.Go(func() error {
		upperChild, err := e.UpperChild(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge upper child: %w", err)
		}
		if !upperChild.IsNone() {
			edge.UpperChildID = common.Hash(upperChild.Unwrap())
		}
		return nil
	})

	eg.Go(func() error {
		ah, err := e.AssertionHash(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge assertion hash: %w", err)
		}
		edge.AssertionHash = common.Hash(ah)
		return nil
	})

	eg.Go(func() error {
		timeUnrivaled, err := e.TimeUnrivaled(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge time unrivaled: %w", err)
		}
		edge.TimeUnrivaled = timeUnrivaled
		return nil
	})

	eg.Go(func() error {
		hasRival, err := e.HasRival(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge has rival: %w", err)
		}
		edge.HasRival = hasRival
		return nil
	})

	eg.Go(func() error {
		status, err := e.Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge status: %w", err)
		}
		edge.Status = status.String()
		return nil
	})

	eg.Go(func() error {
		hasLengthOneRival, err := e.HasLengthOneRival(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge has length one rival: %w", err)
		}
		edge.HasLengthOneRival = hasLengthOneRival
		return nil
	})

	eg.Go(func() error {
		topLevelClaimHeight, err := e.TopLevelClaimHeight(ctx)
		if err != nil {
			return fmt.Errorf("failed to get edge top level claim height: %w", err)
		}
		edge.TopLevelClaimHeight = topLevelClaimHeight
		return nil
	})

	return edge, eg.Wait()
}

func toCommitment(fn func() (protocol.Height, common.Hash)) *Commitment {
	h, hs := fn()
	return &Commitment{
		Height: uint64(h),
		Hash:   hs,
	}
}

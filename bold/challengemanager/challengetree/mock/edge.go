// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

// Package mock includes specific mock setups for edge types used in internal tests.
package mock

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/chainabstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
)

var _ = chainabstraction.ReadOnlyEdge(&Edge{})

type EdgeId string
type Commit string
type OriginId string

// Edge for challenge tree specific tests, making it easier for test ergonomics.
type Edge struct {
	ID                   EdgeId
	EdgeType             chainabstraction.ChallengeLevel
	InnerStatus          chainabstraction.EdgeStatus
	InnerInheritedTimer  chainabstraction.InheritedTimer
	StartHeight          uint64
	StartCommit          Commit
	EndHeight            uint64
	EndCommit            Commit
	OriginID             OriginId
	ClaimID              string
	LowerChildID         EdgeId
	UpperChildID         EdgeId
	CreationBlock        uint64
	TotalChallengeLevels uint8
	IsHonest             bool
}

func (e *Edge) Id() chainabstraction.EdgeId {
	return chainabstraction.EdgeId{Hash: common.BytesToHash([]byte(e.ID))}
}

func (e *Edge) GetChallengeLevel() chainabstraction.ChallengeLevel {
	return e.EdgeType
}

func (e *Edge) GetReversedChallengeLevel() chainabstraction.ChallengeLevel {
	return chainabstraction.ChallengeLevel(e.TotalChallengeLevels) - 1 - e.EdgeType
}

func (e *Edge) GetTotalChallengeLevels(ctx context.Context) uint8 {
	return e.TotalChallengeLevels
}

func (e *Edge) StartCommitment() (chainabstraction.Height, common.Hash) {
	return chainabstraction.Height(e.StartHeight), common.BytesToHash([]byte(e.StartCommit))
}

func (e *Edge) EndCommitment() (chainabstraction.Height, common.Hash) {
	return chainabstraction.Height(e.EndHeight), common.BytesToHash([]byte(e.EndCommit))
}

func (e *Edge) CreatedAtBlock() (uint64, error) {
	return e.CreationBlock, nil
}

func (e *Edge) OriginId() chainabstraction.OriginId {
	return chainabstraction.OriginId(common.BytesToHash([]byte(e.OriginID)))
}

func (e *Edge) MutualId() chainabstraction.MutualId {
	return chainabstraction.MutualId(common.BytesToHash([]byte(e.ComputeMutualId())))
}

func (e *Edge) ComputeMutualId() string {
	return fmt.Sprintf(
		"%d-%s-%d-%s-%d",
		e.EdgeType,
		e.OriginID,
		e.StartHeight,
		e.StartCommit,
		e.EndHeight,
	)
}

// ClaimId of the edge, if any
func (e *Edge) ClaimId() option.Option[chainabstraction.ClaimId] {
	if e.ClaimID == "" {
		return option.None[chainabstraction.ClaimId]()
	}
	return option.Some(chainabstraction.ClaimId(common.BytesToHash([]byte(e.ClaimID))))
}

// LowerChild of the edge, if any.
func (e *Edge) LowerChild(_ context.Context) (option.Option[chainabstraction.EdgeId], error) {
	if e.LowerChildID == "" {
		return option.None[chainabstraction.EdgeId](), nil
	}
	return option.Some(chainabstraction.EdgeId{Hash: common.BytesToHash([]byte(e.LowerChildID))}), nil
}

// UpperChild of the edge, if any.
func (e *Edge) UpperChild(_ context.Context) (option.Option[chainabstraction.EdgeId], error) {
	if e.UpperChildID == "" {
		return option.None[chainabstraction.EdgeId](), nil
	}
	return option.Some(chainabstraction.EdgeId{Hash: common.BytesToHash([]byte(e.UpperChildID))}), nil
}

func (e *Edge) HasChildren(ctx context.Context) (bool, error) {
	return e.LowerChildID != "" && e.UpperChildID != "", nil
}

// MiniStaker of an edge. Only existing for level zero edges.
func (*Edge) MiniStaker() option.Option[common.Address] {
	return option.None[common.Address]()
}

// AssertionHash of the parent assertion that originated the challenge
// at the top-level.
func (*Edge) AssertionHash(_ context.Context) (chainabstraction.AssertionHash, error) {
	return chainabstraction.AssertionHash{}, nil
}

// TimeUnrivaled in seconds an edge has been unrivaled.
func (*Edge) TimeUnrivaled(_ context.Context) (uint64, error) {
	return 0, nil
}

// LatestInheritedTimer in seconds an edge has been unrivaled.
func (e *Edge) LatestInheritedTimer(_ context.Context) (chainabstraction.InheritedTimer, error) {
	return e.InnerInheritedTimer, nil
}

// Status of an edge.
func (e *Edge) Status(_ context.Context) (chainabstraction.EdgeStatus, error) {
	return e.InnerStatus, nil
}

// HasRival if an edge has rivals.
func (*Edge) HasRival(_ context.Context) (bool, error) {
	return false, nil
}

// HasLengthOneRival checks if an edge has a length one rival.
func (*Edge) HasLengthOneRival(_ context.Context) (bool, error) {
	return false, nil
}

// TopLevelClaimHeight for the top-level edge the current edge's challenge is made upon.
// This is used at subchallenge creation boundaries.
func (*Edge) TopLevelClaimHeight(_ context.Context) (chainabstraction.OriginHeights, error) {
	return chainabstraction.OriginHeights{}, nil
}

// HasLengthOneRival checks if an edge has a length one rival.
func (e *Edge) MarkAsHonest() {
	e.IsHonest = true
}

func (e *Edge) AsVerifiedHonest() (chainabstraction.VerifiedRoyalEdge, bool) {
	if e.IsHonest {
		return &MockHonestEdge{e}, true
	}
	return nil, false
}

func (*Edge) Bisect(
	_ context.Context,
	_ common.Hash,
	_ []byte,
) (chainabstraction.VerifiedRoyalEdge, chainabstraction.VerifiedRoyalEdge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (*Edge) ConfirmByTimer(_ context.Context, _ chainabstraction.AssertionHash) (*types.Transaction, error) {
	return nil, errors.New("unimplemented")
}

func (*Edge) ConfirmedAtBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

type MockHonestEdge struct {
	*Edge
}

func (m *MockHonestEdge) Honest() {}

func (m *MockHonestEdge) Bisect(
	ctx context.Context,
	prefixHistoryRoot common.Hash,
	prefixProof []byte,
) (chainabstraction.VerifiedRoyalEdge, chainabstraction.VerifiedRoyalEdge, error) {
	return m.Edge.Bisect(ctx, prefixHistoryRoot, prefixProof)
}

func (m *MockHonestEdge) ConfirmByTimer(ctx context.Context, claimedAssertion chainabstraction.AssertionHash) (*types.Transaction, error) {
	return m.Edge.ConfirmByTimer(ctx, claimedAssertion)
}

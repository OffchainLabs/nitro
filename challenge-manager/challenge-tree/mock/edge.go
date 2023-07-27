package mock

import (
	"context"
	"errors"
	"fmt"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	"github.com/ethereum/go-ethereum/common"
)

var _ = protocol.ReadOnlyEdge(&Edge{})

type EdgeId string
type Commit string
type OriginId string

// Mock Edge for challenge tree specific tests, making it easier for test ergonomics.
type Edge struct {
	ID            EdgeId
	EdgeType      protocol.EdgeType
	StartHeight   uint64
	StartCommit   Commit
	EndHeight     uint64
	EndCommit     Commit
	OriginID      OriginId
	ClaimID       string
	LowerChildID  EdgeId
	UpperChildID  EdgeId
	CreationBlock uint64
}

func (e *Edge) Id() protocol.EdgeId {
	return protocol.EdgeId(common.BytesToHash([]byte(e.ID)))
}

func (e *Edge) GetType() protocol.EdgeType {
	return e.EdgeType
}

func (e *Edge) StartCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.StartHeight), common.BytesToHash([]byte(e.StartCommit))
}

func (e *Edge) EndCommitment() (protocol.Height, common.Hash) {
	return protocol.Height(e.EndHeight), common.BytesToHash([]byte(e.EndCommit))
}

func (e *Edge) CreatedAtBlock() (uint64, error) {
	return e.CreationBlock, nil
}

func (e *Edge) OriginId() protocol.OriginId {
	return protocol.OriginId(common.BytesToHash([]byte(e.OriginID)))
}

func (e *Edge) MutualId() protocol.MutualId {
	return protocol.MutualId(common.BytesToHash([]byte(e.ComputeMutualId())))
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

// The claim id of the edge, if any
func (e *Edge) ClaimId() option.Option[protocol.ClaimId] {
	if e.ClaimID == "" {
		return option.None[protocol.ClaimId]()
	}
	return option.Some(protocol.ClaimId(common.BytesToHash([]byte(e.ClaimID))))
}

// The lower child of the edge, if any.
func (e *Edge) LowerChild(_ context.Context) (option.Option[protocol.EdgeId], error) {
	if e.LowerChildID == "" {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId(common.BytesToHash([]byte(e.LowerChildID)))), nil
}

// The upper child of the edge, if any.
func (e *Edge) UpperChild(_ context.Context) (option.Option[protocol.EdgeId], error) {
	if e.UpperChildID == "" {
		return option.None[protocol.EdgeId](), nil
	}
	return option.Some(protocol.EdgeId(common.BytesToHash([]byte(e.UpperChildID)))), nil
}

func (e *Edge) HasChildren(ctx context.Context) (bool, error) {
	return e.LowerChildID != "" && e.UpperChildID != "", nil
}

// The ministaker of an edge. Only existing for level zero edges.
func (*Edge) MiniStaker() option.Option[common.Address] {
	return option.None[common.Address]()
}

// The assertion hash of the parent assertion that originated the challenge
// at the top-level.
func (*Edge) AssertionHash(_ context.Context) (protocol.AssertionHash, error) {
	return protocol.AssertionHash{}, nil
}

// The time in seconds an edge has been unrivaled.
func (*Edge) TimeUnrivaled(_ context.Context) (uint64, error) {
	return 0, nil
}

// The status of an edge.
func (*Edge) Status(_ context.Context) (protocol.EdgeStatus, error) {
	return 0, nil
}

// Whether or not an edge has rivals.
func (*Edge) HasRival(_ context.Context) (bool, error) {
	return false, nil
}

// Checks if an edge has a length one rival.
func (*Edge) HasLengthOneRival(_ context.Context) (bool, error) {
	return false, nil
}

// The history commitment for the top-level edge the current edge's challenge is made upon.
// This is used at subchallenge creation boundaries.
func (*Edge) TopLevelClaimHeight(_ context.Context) (protocol.OriginHeights, error) {
	return protocol.OriginHeights{}, nil
}

func (*Edge) Bisect(
	_ context.Context,
	_ common.Hash,
	_ []byte,
) (protocol.SpecEdge, protocol.SpecEdge, error) {
	return nil, nil, errors.New("unimplemented")
}

func (*Edge) ConfirmByTimer(_ context.Context, _ []protocol.EdgeId) error {
	return errors.New("unimplemented")
}

func (*Edge) ConfirmByClaim(_ context.Context, _ protocol.ClaimId) error {
	return errors.New("unimplemented")
}

func (*Edge) ConfirmByChildren(_ context.Context) error {
	return errors.New("unimplemented")
}

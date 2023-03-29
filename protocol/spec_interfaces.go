package protocol

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

type EdgeType uint8

const (
	BlockChallengeEdge EdgeType = iota
	BigStepChallengeEdge
	SmallStepChallengeEdge
)

type OriginId common.Hash
type MutualId common.Hash
type EdgeId common.Hash
type ClaimId common.Hash

// SpecChallengeManager implements the research specification.
type SpecChallengeManager interface {
	// Address of the challenge manager contract.
	Address() common.Address
	// Duration of the challenge period.
	ChallengePeriodSeconds(ctx context.Context) (time.Duration, error)
	// Gets an edge by its hash.
	GetEdge(ctx context.Context, edgeId EdgeId) (util.Option[SpecEdge], error)
	// Calculates a mutual id for an edge.
	CalculateMutualId(
		ctx context.Context,
		edgeType EdgeType,
		originId OriginId,
		startHeight Height,
		startHistoryRoot common.Hash,
		endHeight Height,
	) (MutualId, error)
	// Calculates an edge id for an edge.
	CalculateEdgeId(
		ctx context.Context,
		edgeType EdgeType,
		originId OriginId,
		startHeight Height,
		startHistoryRoot common.Hash,
		endHeight Height,
		endHistoryRoot common.Hash,
	) (MutualId, error)
	// Adds a level-zero edge to a block challenge given an assertion and a history commitment.
	AddBlockChallengeLevelZeroEdge(
		ctx context.Context,
		assertion Assertion,
		startHeight Height,
		startHistoryRoot common.Hash,
		endHeight Height,
		endHistoryRoot common.Hash,
	) (SpecEdge, error)
	// Adds a level-zero edge to sub block challenge given a source edge and a history commitment.
	AddSubChallengeLevelZeroEdge(
		ctx context.Context,
		challengedEdge SpecEdge,
		startHeight Height,
		startHistoryRoot common.Hash,
		endHeight Height,
		endHistoryRoot common.Hash,
	) (SpecEdge, error)
}

// Height if defined as the height of a history commitment in the specification.
// Heights are 0-indexed.
type Height uint64

// EdgeStatus of an edge in the protocol.
type EdgeStatus uint8

const (
	EdgePending EdgeStatus = iota
	EdgeConfirmed
)

type SpecEdge interface {
	Id() EdgeId
	GetType() EdgeType
	MiniStaker() (common.Address, error)
	StartCommitment() (Height, common.Hash)
	TargetCommitment() (Height, common.Hash)
	PresumptiveTimer(ctx context.Context) (uint64, error)
	IsPresumptive(ctx context.Context) (bool, error)
	Status(ctx context.Context) (EdgeStatus, error)
	HasConfirmedRival(ctx context.Context) (bool, error)
	// Challenge moves
	Bisect(
		ctx context.Context,
		history util.HistoryCommitment,
		proof []byte,
	) (SpecEdge, SpecEdge, error)
	// Confirms an edge for having a presumptive timer >= a challenge period.
	ConfirmByTimer(ctx context.Context, ancestorIds []EdgeId) error
	// Confirms an edge for having a subchallenge winner of a one-step-proof.
	ConfirmByClaim(ctx context.Context, claimId ClaimId) error
	// Checks the start commitment of an edge is the source of a one-step fork.
	IsOneStepForkSource(ctx context.Context) (bool, error)
	// The history commitment for the top-level edge the current edge's challenge is made upon.
	// This is used at subchallenge creation boundaries.
	TopLevelClaimCommitment(ctx context.Context) (Height, common.Hash, error)
}

package protocol

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// EdgeHash is a unique identifier for an edge in the protocol.
type EdgeHash common.Hash

// SpecChallengeManager implements the research specification.
type SpecChallengeManager interface {
	// Address of the challenge manager contract.
	Address() common.Address
	// Duration of the challenge period.
	ChallengePeriodSeconds(ctx context.Context) (time.Duration, error)
	// Calculates the unique identifier for a challenge given an claim ID and a challenge type.
	// An claim could be an assertion or a vertex that originated the challenge.
	CalculateChallengeHash(ctx context.Context, claimId common.Hash, challengeType ChallengeType) (ChallengeHash, error)
	// Calculates an edge hash given its challenge id, start history, and end history.
	CalculateEdgeHash(
		ctx context.Context,
		challengeId ChallengeHash,
		startHistory util.HistoryCommitment,
		endHistory util.HistoryCommitment,
	) (EdgeHash, error)
	// Gets an edge by its hash.
	GetEdge(ctx context.Context, edgeId EdgeHash) (util.Option[SpecEdge], error)
	// Gets a challenge by its hash.
	GetChallenge(ctx context.Context, challengeId ChallengeHash) (util.Option[SpecChallenge], error)
}

// Height if defined as the height of a history commitment in the specification.
// Heights are 0-indexed.
type Height uint64

// ChallengeStatus represents the enum with the same name
// in the protocol smart contracts.
type ChallengeStatus uint8

const (
	ChallengePending ChallengeStatus = iota
	ChallengeConfirmed
)

// SpecChallenge implements the research specification.
type SpecChallenge interface {
	// The unique identifier of a challenge.
	Id() ChallengeHash
	// The type of challenge.
	GetType() ChallengeType
	// The start timestamp of the challenge.
	StartTime() (uint64, error)
	RootCommitment() (Height, common.Hash, error)
	Status(ctx context.Context) (ChallengeStatus, error)
	// The root assertion the challenge is made upon.
	RootAssertion(ctx context.Context) (Assertion, error)
	// The history commitment for the top-level edge a challenge is made upon.
	// This is used at subchallenge creation boundaries.
	TopLevelClaimCommitment(ctx context.Context) (Height, common.Hash, error)
	// The winner level-zero edge for a challenge.
	WinningEdge(ctx context.Context) (util.Option[SpecEdge], error)
	// Checks if two edges are at a one-step-fork.
	AreAtOneStepFork(a, b SpecEdge) (bool, error)
	CreateSubChallenge(ctx context.Context) (SpecChallenge, error)
	// Adds a level-zero edge to a block challenge given an assertion and a history commitment.
	AddBlockChallengeLevelZeroEdge(
		ctx context.Context,
		assertion Assertion,
		history util.HistoryCommitment,
	) (SpecEdge, error)
	// Adds a level-zero edge to sub block challenge given a source edge and a history commitment.
	AddSubChallengeLevelZeroEdge(
		ctx context.Context,
		challengedEdge SpecEdge,
		history util.HistoryCommitment,
	) (SpecEdge, error)
}

// EdgeStatus of an edge in the protocol.
type EdgeStatus uint8

const (
	EdgePending EdgeStatus = iota
	EdgeConfirmed
)

// The two direct children of an edge.
// nolint:unused
type EdgeChildren struct {
	// nolint:unused
	a SpecEdge
	// nolint:unused
	b SpecEdge
}

type SpecEdge interface {
	Id() [32]byte
	MiniStaker() (common.Address, error)
	StartCommitment() (Height, common.Hash)
	TargetCommitment() (Height, common.Hash)
	PresumptiveTimer(ctx context.Context) (uint64, error)
	IsPresumptive(ctx context.Context) (bool, error)
	Status(ctx context.Context) (EdgeStatus, error)
	HasConfirmedRival(ctx context.Context) (bool, error)
	// Gets the two direct children of an edge, if any.
	DirectChildren(ctx context.Context) (util.Option[EdgeChildren], error)
	GetSubChallenge(ctx context.Context) (util.Option[SpecChallenge], error)
	// Challenge moves
	Bisect(
		ctx context.Context,
		history util.HistoryCommitment,
		proof []byte,
	) (SpecEdge, SpecEdge, error)
	// Confirms an edge for having a presumptive timer >= a challenge period.
	ConfirmForTimer(ctx context.Context) error
	// Confirms an edge for having a subchallenge winner of a one-step-proof.
	ConfirmForSubChallengeWin(ctx context.Context) error
}

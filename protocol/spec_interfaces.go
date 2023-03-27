package protocol

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

type EdgeHash common.Hash

type SpecChallengeManager interface {
	Address() common.Address
	ChallengePeriodSeconds(ctx context.Context) (time.Duration, error)
	CalculateChallengeHash(ctx context.Context, itemId common.Hash, challengeType ChallengeType) (ChallengeHash, error)
	CalculateEdgeId(ctx context.Context, challengeId ChallengeHash, history util.HistoryCommitment) (EdgeHash, error)
	GetEdge(ctx context.Context, edgeId EdgeHash) (util.Option[SpecEdge], error)
	GetChallenge(ctx context.Context, challengeId ChallengeHash) (util.Option[SpecChallenge], error)
}

type Height uint64

// ChallengeStatus represents the enum with the same name
// in the protocol smart contracts.
type ChallengeStatus uint8

const (
	ChallengePending ChallengeStatus = iota
	ChallengeConfirmed
)

type SpecChallenge interface {
	// Basic attributes.
	Id() ChallengeHash
	GetType() ChallengeType
	StartTime() (uint64, error)
	RootCommitment() (Height, common.Hash, error)
	Status(ctx context.Context) (ChallengeStatus, error)
	RootAssertion(ctx context.Context) (Assertion, error)
	TopLevelClaimCommitment(ctx context.Context) (Height, common.Hash, error)
	// The winner level-zero edge for a challenge.
	WinningEdge(ctx context.Context) (util.Option[SpecEdge], error)
	AreAtOneStepFork(a, b SpecEdge) (bool, error)
	CreateSubChallenge(ctx context.Context) (SpecChallenge, error)
	AddBlockChallengeLevelZeroEdge(
		ctx context.Context,
		assertion Assertion,
		history util.HistoryCommitment,
	) (SpecEdge, error)
	AddSubChallengeLevelZeroEdge(
		ctx context.Context,
		challengedEdge SpecEdge,
		history util.HistoryCommitment,
	) (SpecEdge, error)
}

type EdgeStatus uint8

const (
	EdgePending EdgeStatus = iota
	EdgeConfirmed
)

// The two direct children of an edge.
type EdgeChildren struct {
	a SpecEdge
	b SpecEdge
}

type SpecEdge interface {
	// Basic attributes.
	Id() [32]byte
	MiniStaker() (common.Address, error)
	StartCommitment() (Height, common.Hash, error)
	TargetCommitment() (Height, common.Hash, error)
	PresumptiveTimer(ctx context.Context) (uint64, error)
	IsPresumptive(ctx context.Context) (bool, error)
	Status(ctx context.Context) (EdgeStatus, error)
	HasConfirmedRival(ctx context.Context) (bool, error)
	DirectChildren(ctx context.Context) (util.Option[EdgeChildren], error)
	GetSubChallenge(ctx context.Context) (util.Option[SpecChallenge], error)
	// Challenge moves
	Bisect(
		ctx context.Context,
		history util.HistoryCommitment,
		proof []byte,
	) (SpecEdge, SpecEdge, error)
	ConfirmForTimer(ctx context.Context) error
	ConfirmForSubChallengeWin(ctx context.Context) error
}

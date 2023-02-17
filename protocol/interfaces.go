package main

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// AssertionManager allows the creation of new leaves for a Staker with a State Commitment
// and a previous assertion.
//
//	type	AssertionManager interface {
//			Inbox() *Inbox
//			NumAssertions(tx *ActiveTx) uint64
//			AssertionBySequenceNum(tx *ActiveTx, seqNum AssertionSequenceNumber) (*Assertion, error)
//			ChallengeByCommitHash(tx *ActiveTx, commitHash ChallengeCommitHash) (ChallengeInterface, error)
//			ChallengeVertexByCommitHash(tx *ActiveTx, challenge ChallengeCommitHash, vertex VertexCommitHash) (*ChallengeVertex, error)
//			IsAtOneStepFork(
//				ctx context.Context,
//				tx *ActiveTx,
//				challengeCommitHash ChallengeCommitHash,
//				vertexCommit util.HistoryCommitment,
//				vertexParentCommit util.HistoryCommitment,
//			) (bool, error)
//			ChallengePeriodLength(tx *ActiveTx) time.Duration
//			LatestConfirmed(*ActiveTx) *Assertion
//			CreateLeaf(tx *ActiveTx, prev *Assertion, commitment util.StateCommitment, staker common.Address) (*Assertion, error)
//			TimeReference() util.TimeReference
//		}

type ActiveTx struct{}

type AssertionSequenceNumber uint64
type VertexSequenceNumber uint64

type AssertionChain interface {
	AssertionBySequenceNum(
		ctx context.Context,
		tx *ActiveTx,
		seqNum AssertionSequenceNumber,
	) (*Assertion, error)
	ChallengePeriodLength(tx *ActiveTx) time.Duration
	LatestConfirmed(tx *ActiveTx) *Assertion
}

type ChallengeManager interface {
}

type Assertion interface{}

type ChallengeType uint8
type AssertionState uint8

type Challenge interface {
	RootAssertion(ctx context.Context, tx *ActiveTx) (Assertion, error)
	Completed(ctx context.Context, tx *ActiveTx) (bool, error)
	HasConfirmedSibling(ctx context.Context, tx *ActiveTx, vertex ChallengeVertex) (bool, error)
	RootVertex(ctx context.Context, tx *ActiveTx) (ChallengeVertex, error)
	ParentStateCommitment(ctx context.Context, tx *ActiveTx) (util.StateCommitment, error)
	AddLeaf(
		ctx context.Context,
		tx *ActiveTx,
		assertion Assertion,
		history util.HistoryCommitment,
		validator common.Address,
	) (ChallengeVertex, error)
	AddSubchallengeLeaf(
		ctx context.Context,
		tx *ActiveTx,
		vertex ChallengeVertex,
		history util.HistoryCommitment,
		validator common.Address,
	) (ChallengeVertex, error)
	WinnerVertex(ctx context.Context, tx *ActiveTx) (util.Option[ChallengeVertex], error)
	HasEnded(
		ctx context.Context,
		tx *ActiveTx,
		challengeManager ChallengeManager,
	) (bool, error)
	GetType(ctx context.Context, tx *ActiveTx) (ChallengeType, error)
	GetCreationTime(ctx context.Context, tx *ActiveTx) (time.Time, error)
}

type ChallengeVertex interface {
	ConfirmForPsTimer(ctx context.Context, tx *ActiveTx) error
	ConfirmForChallengeDeadline(ctx context.Context, tx *ActiveTx) error
	ConfirmForSubChallengeWin(ctx context.Context, tx *ActiveTx) error
	IsPresumptiveSuccessor(ctx context.Context, tx *ActiveTx) (bool, error)
	Bisect(
		ctx context.Context,
		tx *ActiveTx,
		history util.HistoryCommitment,
		proof []common.Hash,
		validator common.Address,
	) (ChallengeVertex, error)
	Merge(
		ctx context.Context,
		tx *ActiveTx,
		mergingTo ChallengeVertex,
		proof []common.Hash,
		validator common.Address,
	) error
	EligibleForNewSuccessor(ctx context.Context, tx *ActiveTx) (bool, error)
	Prev(ctx context.Context, tx *ActiveTx) (util.Option[ChallengeVertex], error)
	Status(ctx context.Context, tx *ActiveTx) (AssertionState, error)
	GetSubChallenge(ctx context.Context, tx *ActiveTx) (util.Option[Challenge], error)
	PsTimer(ctx context.Context, tx *ActiveTx) (*util.CountUpTimer, error)
	HistoryCommitment(ctx context.Context, tx *ActiveTx) (util.HistoryCommitment, error)
	MiniStaker(ctx context.Context, tx *ActiveTx) (common.Address, error)
	SequenceNum(ctx context.Context, tx *ActiveTx) (VertexSequenceNumber, error)
	PresumptiveSuccessor(ctx context.Context, tx *ActiveTx) (util.Option[ChallengeVertex], error)
	ChessClockExpired(ctx context.Context, tx *ActiveTx, challengePeriod time.Duration) (bool, error)
}

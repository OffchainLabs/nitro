package protocol

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// EdgeType corresponds to the three different challenge
// levels in the protocol: block challenges, big step challenges,
// and small step challenges.
type EdgeType uint8

const (
	BlockChallengeEdge EdgeType = iota
	BigStepChallengeEdge
	SmallStepChallengeEdge
)

func (et EdgeType) String() string {
	switch et {
	case BlockChallengeEdge:
		return "block_challenge_edge"
	case BigStepChallengeEdge:
		return "big_step_challenge_edge"
	case SmallStepChallengeEdge:
		return "small_step_challenge_edge"
	default:
		return "unknown"
	}
}

// OriginId is the id of the item that originated a challenge an edge
// is a part of. In a block challenge, the origin id is the id of the assertion
// being challenged. In a big step challenge, it is the mutual id of the edge at the block challenge
// level that was the source of the one step fork leading to the big step challenge.
// In a small step challenge, it is the mutual id of the edge at the big step level that was
// the source of the one step fork leading to the small step challenge.
type OriginId common.Hash

// MutualId is a unique identifier for an edge's start commitment and edge type.
// Rival edges share a mutual id. For example, an edge going A --> B, and another
// going from A --> C would share A, and we define the mutual id as the unique identifier
// for A.
type MutualId common.Hash

// EdgeId is a unique identifier for an edge. Edge IDs encompass the edge type
// along with the start and end height + commitment for an edge.
type EdgeId common.Hash

// ClaimId is the unique identifier of the commitment of a level zero edge corresponds to.
// For example, if assertion A has two children, B and C, and a block challenge is initiated
// on A, the level zero edges will have claim ids corresponding to assertions B and C when opened.
// The same occurs in the subchallenge layers, where claim ids are the edges at the higher challenge
// level corresponding to the level zero edges in the respective subchallenge.
type ClaimId common.Hash

// SpecChallengeManager implements the research specification.
type SpecChallengeManager interface {
	// Address of the challenge manager contract.
	Address() common.Address
	// Duration of the challenge period.
	ChallengePeriodSeconds(ctx context.Context) (time.Duration, error)
	// Gets an edge by its id.
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
	) (EdgeId, error)
	// Adds a level-zero edge to a block challenge given an assertion and a history commitments.
	AddBlockChallengeLevelZeroEdge(
		ctx context.Context,
		assertion Assertion,
		startCommit util.HistoryCommitment,
		endCommit util.HistoryCommitment,
	) (SpecEdge, error)
	// Adds a level-zero edge to subchallenge given a source edge and history commitments.
	AddSubChallengeLevelZeroEdge(
		ctx context.Context,
		challengedEdge SpecEdge,
		startCommit util.HistoryCommitment,
		endCommit util.HistoryCommitment,
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

// SpecEdge according to the protocol specification.
type SpecEdge interface {
	// The unique identifier for an edge.
	Id() EdgeId
	// The type of challenge the edge is a part of.
	GetType() EdgeType
	// The ministaker of an edge. Only existing for level zero edges.
	MiniStaker() util.Option[common.Address]
	// The start height and history commitment for an edge.
	StartCommitment() (Height, common.Hash)
	// The end height and history commitment for an edge.
	EndCommitment() (Height, common.Hash)
	// The presumptive timer in seconds for an edge.
	PresumptiveTimer(ctx context.Context) (uint64, error)
	// Whether or not an edge is presumptive.
	IsPresumptive(ctx context.Context) (bool, error)
	// The status of an edge.
	Status(ctx context.Context) (EdgeStatus, error)
	// Checks the start commitment of an edge is the source of a one-step fork.
	IsOneStepForkSource(ctx context.Context) (bool, error)
	// Bisection capabilities for an edge. Returns the two child
	// edges that are created as a result.
	Bisect(
		ctx context.Context,
		prefixHistoryRoot common.Hash,
		prefixProof []byte,
	) (SpecEdge, SpecEdge, error)
	// Confirms an edge for having a presumptive timer >= one challenge period.
	ConfirmByTimer(ctx context.Context, ancestorIds []EdgeId) error
	// Confirms an edge with the specified claim id.
	ConfirmByClaim(ctx context.Context, claimId ClaimId) error
	ConfirmByOneStepProof(ctx context.Context) error
	// The history commitment for the top-level edge the current edge's challenge is made upon.
	// This is used at subchallenge creation boundaries.
	OriginCommitment(ctx context.Context) (Height, common.Hash, error)
}

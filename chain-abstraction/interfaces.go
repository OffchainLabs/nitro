// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package protocol

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"strconv"

	"github.com/OffchainLabs/bold/containers/option"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	commitments "github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// AssertionHash represents a unique identifier for an assertion
// constructed as a keccak256 hash of some of its internals.
type AssertionHash struct {
	common.Hash
}

// Protocol --
type Protocol interface {
	AssertionChain
}

type AssertionStatus uint8

const (
	NoAssertion AssertionStatus = iota
	AssertionPending
	AssertionConfirmed
)

// Assertion represents a top-level claim in the protocol about the
// chain state created by a validator that stakes on their claim.
// Assertions can be challenged.
type Assertion interface {
	Id() AssertionHash
	PrevId(ctx context.Context) (AssertionHash, error)
	HasSecondChild() (bool, error)
	CreatedAtBlock() (uint64, error)
}

// AssertionCreatedInfo from an event creation.
type AssertionCreatedInfo struct {
	ConfirmPeriodBlocks uint64
	RequiredStake       *big.Int
	ParentAssertionHash common.Hash
	BeforeState         rollupgen.ExecutionState
	AfterState          rollupgen.ExecutionState
	InboxMaxCount       *big.Int
	AfterInboxBatchAcc  common.Hash
	AssertionHash       common.Hash
	WasmModuleRoot      common.Hash
	ChallengeManager    common.Address
	TransactionHash     common.Hash
	CreationBlock       uint64
}

func (i AssertionCreatedInfo) ExecutionHash() common.Hash {
	afterGlobalStateHash := GoGlobalStateFromSolidity(i.AfterState.GlobalState).Hash()
	return crypto.Keccak256Hash(append([]byte{i.AfterState.MachineStatus}, afterGlobalStateHash.Bytes()...))
}

// AssertionChain can manage assertions in the protocol and retrieve
// information about them. It also has an associated challenge manager
// which is used for all challenges in the protocol.
type AssertionChain interface {
	// Read-only methods.
	IsStaked(ctx context.Context) (bool, error)
	GetAssertion(ctx context.Context, id AssertionHash) (Assertion, error)
	LatestConfirmed(ctx context.Context) (Assertion, error)
	LatestCreatedAssertion(ctx context.Context) (Assertion, error)
	LatestCreatedAssertionHashes(ctx context.Context) ([]AssertionHash, error)
	ReadAssertionCreationInfo(
		ctx context.Context, id AssertionHash,
	) (*AssertionCreatedInfo, error)

	AssertionUnrivaledBlocks(ctx context.Context, assertionHash AssertionHash) (uint64, error)
	TopLevelAssertion(ctx context.Context, edgeId EdgeId) (AssertionHash, error)
	TopLevelClaimHeights(ctx context.Context, edgeId EdgeId) (OriginHeights, error)

	// Mutating methods.
	NewStakeOnNewAssertion(
		ctx context.Context,
		assertionCreationInfo *AssertionCreatedInfo,
		postState *ExecutionState,
	) (Assertion, error)
	StakeOnNewAssertion(
		ctx context.Context,
		assertionCreationInfo *AssertionCreatedInfo,
		postState *ExecutionState,
	) (Assertion, error)
	ConfirmAssertionByChallengeWinner(
		ctx context.Context,
		assertionHash AssertionHash,
		winningEdgeId EdgeId,
	) error

	// Spec-based implementation methods.
	SpecChallengeManager(ctx context.Context) (SpecChallengeManager, error)
}

// ChallengeLevel corresponds to the different challenge levels in the protocol.
// 0 is for block challenges and the last level is for small step challenges.
// Everything else is a big step challenge of level i where 0 < i < last.
type ChallengeLevel uint64

func NewBlockChallengeLevel() ChallengeLevel {
	return 0
}

func (et ChallengeLevel) Big() *big.Int {
	return new(big.Int).SetUint64(uint64(et))
}
func (et ChallengeLevel) IsBlockChallengeLevel() bool {
	return et == 0
}

func (et ChallengeLevel) Next() ChallengeLevel {
	return et + 1
}

func (et ChallengeLevel) String() string {
	if et == 0 {
		return "block_challenge_edge"
	}
	return fmt.Sprintf("challenge_level_%d_edge", et)
}

func ChallengeLevelFromString(s string) (ChallengeLevel, error) {
	switch s {
	case "block_challenge_edge":
		return 0, nil
	default:
		re := regexp.MustCompile("[0-9]+")
		challengeLevel, err := strconv.Atoi(re.FindString(s))
		if err != nil {
			return 0, err
		}
		return ChallengeLevel(challengeLevel), nil
	}
}

type Agreement struct {
	AgreesWithStartCommit bool
	IsHonestEdge          bool
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
type EdgeId struct {
	common.Hash
}

// ClaimId is the unique identifier of the commitment of a level zero edge corresponds to.
// For example, if assertion A has two children, B and C, and a block challenge is initiated
// on A, the level zero edges will have claim ids corresponding to assertions B and C when opened.
// The same occurs in the subchallenge layers, where claim ids are the edges at the higher challenge
// level corresponding to the level zero edges in the respective subchallenge.
type ClaimId common.Hash

// OneStepData used for confirming edges by one step proofs.
type OneStepData struct {
	BeforeHash common.Hash
	Proof      []byte
}

// SpecChallengeManager implements the research specification.
type SpecChallengeManager interface {
	// Address of the challenge manager contract.
	Address() common.Address
	// Heights for level zero edge creation.
	LevelZeroBlockEdgeHeight(ctx context.Context) (uint64, error)
	// Duration of the challenge period in blocks.
	ChallengePeriodBlocks(ctx context.Context) (uint64, error)
	// Gets an edge by its id.
	GetEdge(ctx context.Context, edgeId EdgeId) (option.Option[SpecEdge], error)
	// Calculates an edge id for an edge.
	CalculateEdgeId(
		ctx context.Context,
		edgeType ChallengeLevel,
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
		startCommit,
		endCommit commitments.History,
		startEndPrefixProof []byte,
	) (VerifiedHonestEdge, error)
	// Adds a level-zero edge to subchallenge given a source edge and history commitments.
	AddSubChallengeLevelZeroEdge(
		ctx context.Context,
		challengedEdge SpecEdge,
		startCommit,
		endCommit commitments.History,
		startParentInclusionProof []common.Hash,
		endParentInclusionProof []common.Hash,
		startEndPrefixProof []byte,
	) (VerifiedHonestEdge, error)
	ConfirmEdgeByOneStepProof(
		ctx context.Context,
		tentativeWinnerId EdgeId,
		oneStepData *OneStepData,
		preHistoryInclusionProof []common.Hash,
		postHistoryInclusionProof []common.Hash,
	) error
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

func (e EdgeStatus) String() string {
	switch e {
	case EdgePending:
		return "pending"
	case EdgeConfirmed:
		return "confirmed"
	default:
		return "unknown"
	}
}

type OriginHeights struct {
	ChallengeOriginHeights []Height `json:"challengeOriginHeights"`
}

// ReadOnlyEdge defines methods that only retrieve data from the chain
// regarding for a given edge.
type ReadOnlyEdge interface {
	// The unique identifier for an edge.
	Id() EdgeId
	// The challenge level the edge is a part of.
	GetChallengeLevel() (ChallengeLevel, error)
	// GetReversedChallengeLevel obtains the challenge level for the edge. The lowest level starts at 0, and goes all way
	// up to the max number of levels. The reason we go from the lowest challenge level being 0 instead of 2
	// is to make our code a lot more readable. If we flipped the order, we would need to do
	// a lot of backwards for loops instead of simple range loops over slices.
	GetReversedChallengeLevel() (ChallengeLevel, error)
	// Total number possible challenge levels.
	GetTotalChallengeLevels(ctx context.Context) (uint64, error)
	// The start height and history commitment for an edge.
	StartCommitment() (Height, common.Hash)
	// The end height and history commitment for an edge.
	EndCommitment() (Height, common.Hash)
	// The block number the edge was created at.
	CreatedAtBlock() (uint64, error)
	// The mutual id of the edge.
	MutualId() MutualId
	// The origin id of the edge.
	OriginId() OriginId
	// The claim id of the edge, if any
	ClaimId() option.Option[ClaimId]
	// Checks if the edge has children.
	HasChildren(ctx context.Context) (bool, error)
	// The lower child of the edge, if any.
	LowerChild(ctx context.Context) (option.Option[EdgeId], error)
	// The upper child of the edge, if any.
	UpperChild(ctx context.Context) (option.Option[EdgeId], error)
	// The ministaker of an edge. Only existing for level zero edges.
	MiniStaker() option.Option[common.Address]
	// The assertion hash of the parent assertion that originated the challenge
	// at the top-level.
	AssertionHash(ctx context.Context) (AssertionHash, error)
	// The time in seconds an edge has been unrivaled.
	TimeUnrivaled(ctx context.Context) (uint64, error)
	// Whether or not an edge has rivals.
	HasRival(ctx context.Context) (bool, error)
	// The status of an edge.
	Status(ctx context.Context) (EdgeStatus, error)
	// Checks if an edge has a length one rival.
	HasLengthOneRival(ctx context.Context) (bool, error)
	// The history commitment for the top-level edge the current edge's challenge is made upon.
	// This is used at subchallenge creation boundaries.
	TopLevelClaimHeight(ctx context.Context) (OriginHeights, error)
}

// VerifiedHonestEdge marks edges that are known to be honest. For example,
// when a local validator creates an edge, it is known to be honest and several types
// expensive or duplicate computation can be avoided in methods that take in this type.
// A sentinel method `Honest()` is used to mark an edge as satisfying this interface.
type VerifiedHonestEdge interface {
	SpecEdge
	Honest()
}

// SpecEdge according to the protocol specification.
type SpecEdge interface {
	ReadOnlyEdge
	// Bisection capabilities for an edge. Returns the two child
	// edges that are created as a result.
	Bisect(
		ctx context.Context,
		prefixHistoryRoot common.Hash,
		prefixProof []byte,
	) (VerifiedHonestEdge, VerifiedHonestEdge, error)
	// Confirms an edge for having a presumptive timer >= one challenge period.
	ConfirmByTimer(ctx context.Context, ancestorIds []EdgeId) error
	// Confirms an edge with the specified claim id.
	ConfirmByClaim(ctx context.Context, claimId ClaimId) error
	ConfirmByChildren(ctx context.Context) error
}

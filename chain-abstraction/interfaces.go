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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ChainBackend to interact with the underlying blockchain.
type ChainBackend interface {
	bind.ContractBackend
	ReceiptFetcher
}

// ReceiptFetcher defines the ability to retrieve transactions receipts from the chain.
type ReceiptFetcher interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

// LayerZeroHeights for edges configured as parameters in the challenge manager contract.
type LayerZeroHeights struct {
	BlockChallengeHeight     uint64
	BigStepChallengeHeight   uint64
	SmallStepChallengeHeight uint64
}

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

func (a AssertionStatus) String() string {
	switch a {
	case NoAssertion:
		return "no_assertion"
	case AssertionPending:
		return "pending"
	case AssertionConfirmed:
		return "confirmed"
	default:
		return "unknown_status"
	}
}

const BeforeDeadlineAssertionConfirmationError = "BEFORE_DEADLINE"
const ChallengeGracePeriodNotPassedAssertionConfirmationError = "CHALLENGE_GRACE_PERIOD_NOT_PASSED"

// Assertion represents a top-level claim in the protocol about the
// chain state created by a validator that stakes on their claim.
// Assertions can be challenged.
type Assertion interface {
	Id() AssertionHash
	PrevId(ctx context.Context) (AssertionHash, error)
	HasSecondChild() (bool, error)
	FirstChildCreationBlock() (uint64, error)
	SecondChildCreationBlock() (uint64, error)
	IsFirstChild() (bool, error)
	CreatedAtBlock() uint64
	Status(ctx context.Context) (AssertionStatus, error)
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
	RollupUserLogic() *rollupgen.RollupUserLogic
	GetAssertion(ctx context.Context, id AssertionHash) (Assertion, error)
	IsChallengeComplete(ctx context.Context, challengeParentAssertionHash AssertionHash) (bool, error)
	Backend() ChainBackend
	AssertionStatus(
		ctx context.Context,
		assertionHash AssertionHash,
	) (AssertionStatus, error)
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
	ConfirmAssertionByTime(
		ctx context.Context,
		assertionHash AssertionHash,
	) error
	ConfirmAssertionByChallengeWinner(
		ctx context.Context,
		assertionHash AssertionHash,
		winningEdgeId EdgeId,
	) error

	// Spec-based implementation methods.
	SpecChallengeManager(ctx context.Context) (SpecChallengeManager, error)
}

// InheritedTimer for an edge from its children or claiming edges.
type InheritedTimer uint64

// ChallengeLevel corresponds to the different challenge levels in the protocol.
// 0 is for block challenges and the last level is for small step challenges.
// Everything else is a big step challenge of level i where 0 < i < last.
type ChallengeLevel uint8

func NewBlockChallengeLevel() ChallengeLevel {
	return 0
}

func (et ChallengeLevel) Uint8() uint8 {
	return uint8(et)
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
	AfterHash  common.Hash
	Proof      []byte
}

// SpecChallengeManager implements the research specification.
type SpecChallengeManager interface {
	// Address of the challenge manager contract.
	Address() common.Address
	// Layer zero edge heights defined the challenge manager contract.
	LayerZeroHeights(ctx context.Context) (*LayerZeroHeights, error)
	// Number of big step challenge levels defined in the challenge manager contract.
	NumBigSteps(ctx context.Context) (uint8, error)
	// Duration of the challenge period in blocks.
	ChallengePeriodBlocks(ctx context.Context) (uint64, error)
	// Gets an edge by its id.
	GetEdge(ctx context.Context, edgeId EdgeId) (option.Option[SpecEdge], error)
	MultiUpdateInheritedTimers(
		ctx context.Context,
		challengeBranch []ReadOnlyEdge,
	) error
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
	) (VerifiedRoyalEdge, error)
	// Adds a level-zero edge to subchallenge given a source edge and history commitments.
	AddSubChallengeLevelZeroEdge(
		ctx context.Context,
		challengedEdge SpecEdge,
		startCommit,
		endCommit commitments.History,
		startParentInclusionProof []common.Hash,
		endParentInclusionProof []common.Hash,
		startEndPrefixProof []byte,
	) (VerifiedRoyalEdge, error)
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
	GetChallengeLevel() ChallengeLevel
	// GetReversedChallengeLevel obtains the challenge level for the edge. The lowest level starts at 0, and goes all way
	// up to the max number of levels. The reason we go from the lowest challenge level being 0 instead of 2
	// is to make our code a lot more readable. If we flipped the order, we would need to do
	// a lot of backwards for loops instead of simple range loops over slices.
	GetReversedChallengeLevel() ChallengeLevel
	// Total number possible challenge levels.
	GetTotalChallengeLevels(ctx context.Context) uint8
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
	// The inherited timer from the edge's children or claiming edges. Needs to be refreshed
	// onchain over time.
	InheritedTimer(ctx context.Context) (InheritedTimer, error)
	// Whether or not an edge has rivals.
	HasRival(ctx context.Context) (bool, error)
	// The status of an edge.
	Status(ctx context.Context) (EdgeStatus, error)
	// The block at which the edge was confirmed.
	ConfirmedAtBlock(ctx context.Context) (uint64, error)
	// Checks if an edge has a length one rival.
	HasLengthOneRival(ctx context.Context) (bool, error)
	// The history commitment for the top-level edge the current edge's challenge is made upon.
	// This is used at subchallenge creation boundaries.
	TopLevelClaimHeight(ctx context.Context) (OriginHeights, error)
}

// VerifiedRoyalEdge marks edges that are known to be royal. For example,
// when a local validator creates an edge, it is known to be royal and several types
// expensive or duplicate computation can be avoided in methods that take in this type.
// A sentinel method `Honest()` is used to mark an edge as satisfying this interface.
type VerifiedRoyalEdge interface {
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
	) (VerifiedRoyalEdge, VerifiedRoyalEdge, error)
	// Confirms an edge for having a total timer >= one challenge period.
	ConfirmByTimer(ctx context.Context) error
}

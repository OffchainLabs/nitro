package protocol

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"
)

// AssertionSequenceNumber is a monotonically increasing ID
// for each assertion in the chain.
type AssertionSequenceNumber uint64

// VertexSequenceNumber is a monotonically increasing ID
// for each vertex in the chain.
type VertexSequenceNumber uint64

// AssertionHash represents a unique identifier for an assertion
// constructed as a keccak256 hash of some of its internals.
type AssertionHash common.Hash

// ChallengeHash represents a unique identifier for a challenge
// constructed as a keccak256 hash of some of its internals.
type ChallengeHash common.Hash

// VertexHash represents a unique identifier for a challenge vertex
// constructed as a keccak256 hash of some of its internals.
type VertexHash common.Hash

// ActiveTx is a transaction that is currently being processed.
type ActiveTx interface {
	FinalizedBlockNumber() *big.Int // Finalized block number.
	HeadBlockNumber() *big.Int      // If nil, uses the latest block in the chain.
	ReadOnly() bool                 // Checks if a transaction is read-only.
	Sender() common.Address
}

// ChainReader can only make non-mutating calls to a backing blockchain.
// It executes a callback and feeds it an ActiveTx type which includes relevant
// data about the chain, such as the finalized block number and head block number.
type ChainReader interface {
	Call(callback func(ActiveTx) error) error
}

// ChainReadWriter can make mutating and non-mutating calls to a backing blockchain.
// It can executes a callbacks and feed them an ActiveTx type which includes relevant
// data about the chain, such as the finalized block number and head block number.
type ChainReadWriter interface {
	ChainReader
	Tx(callback func(ActiveTx) error) error
}

// Protocol --
type Protocol interface {
	ChainReadWriter
	AssertionChain
}

// AssertionChain can manage assertions in the protocol and retrieve
// information about them. It also has an associated challenge manager
// which is used for all challenges in the protocol.
type AssertionChain interface {
	// Read-only methods.
	NumAssertions(
		ctx context.Context,
		tx ActiveTx,
	) (uint64, error)
	AssertionBySequenceNum(
		ctx context.Context,
		tx ActiveTx,
		seqNum AssertionSequenceNumber,
	) (Assertion, error)
	LatestConfirmed(ctx context.Context, tx ActiveTx) (Assertion, error)
	CurrentChallengeManager(ctx context.Context, tx ActiveTx) (ChallengeManager, error)
	GetAssertionId(
		ctx context.Context,
		tx ActiveTx,
		seqNum AssertionSequenceNumber,
	) (AssertionHash, error)

	// Mutating methods.
	CreateAssertion(
		ctx context.Context,
		tx ActiveTx,
		height uint64,
		prevSeqNum AssertionSequenceNumber,
		prevAssertionState *ExecutionState,
		postState *ExecutionState,
		prevInboxMaxCount *big.Int,
	) (Assertion, error)
	CreateSuccessionChallenge(
		ctx context.Context, tx ActiveTx, seqNum AssertionSequenceNumber,
	) (Challenge, error)
	Confirm(
		ctx context.Context, tx ActiveTx, blockHash, sendRoot common.Hash,
	) error
	Reject(
		ctx context.Context, tx ActiveTx, staker common.Address,
	) error
}

// ChallengeManager allows for retrieving details of challenges such
// as challenges themselves, vertices, or constants such as the challenge period seconds.
type ChallengeManager interface {
	ChallengePeriodSeconds(
		ctx context.Context, tx ActiveTx,
	) (time.Duration, error)
	CalculateChallengeHash(
		ctx context.Context,
		tx ActiveTx,
		itemId common.Hash,
		challengeType ChallengeType,
	) (ChallengeHash, error)
	CalculateChallengeVertexId(
		ctx context.Context,
		tx ActiveTx,
		challengeId ChallengeHash,
		history util.HistoryCommitment,
	) (VertexHash, error)
	GetVertex(
		ctx context.Context,
		tx ActiveTx,
		vertexId VertexHash,
	) (util.Option[ChallengeVertex], error)
	GetChallenge(
		ctx context.Context,
		tx ActiveTx,
		challengeId ChallengeHash,
	) (util.Option[Challenge], error)
}

// Assertion represents a top-level claim in the protocol about the
// chain state created by a validator that stakes on their claim.
// Assertions can be challenged.
type Assertion interface {
	Height() uint64
	SeqNum() AssertionSequenceNumber
	PrevSeqNum() AssertionSequenceNumber
	StateHash() common.Hash
}

// ChallengeType represents the enum with the same name
// in the protocol smart contracts.
type ChallengeType uint8

const (
	BlockChallenge ChallengeType = iota
	BigStepChallenge
	SmallStepChallenge
)

// AssertionState represents the enum with the same name
// in the protocol smart contracts.
type AssertionState uint8

const (
	AssertionPending AssertionState = iota
	AssertionConfirmed
	AssertionRejected
)

// Challenge represents a challenge instance in the protocol, with associated
// methods about its lifecycle, getters of its internals, and methods that
// make mutating calls to the chain for adding leaves and confirming / rejecting
// a challenge.
type Challenge interface {
	// Getters.
	GetType() ChallengeType
	WinningClaim() util.Option[AssertionHash]
	RootAssertion(ctx context.Context, tx ActiveTx) (Assertion, error)
	RootVertex(ctx context.Context, tx ActiveTx) (ChallengeVertex, error)
	GetCreationTime(ctx context.Context, tx ActiveTx) (time.Time, error)
	ParentStateCommitment(ctx context.Context, tx ActiveTx) (util.StateCommitment, error)
	WinnerVertex(ctx context.Context, tx ActiveTx) (util.Option[ChallengeVertex], error)
	Completed(ctx context.Context, tx ActiveTx) (bool, error)

	// Mutating calls.
	AddBlockChallengeLeaf(
		ctx context.Context,
		tx ActiveTx,
		assertion Assertion,
		history util.HistoryCommitment,
	) (ChallengeVertex, error)
	AddBigStepChallengeLeaf(
		ctx context.Context,
		tx ActiveTx,
		vertex ChallengeVertex,
		history util.HistoryCommitment,
	) (ChallengeVertex, error)
}

// ChallengeVertex represents a challenge vertex instance in the protocol, with associated
// methods about its lifecycle, getters of its internals, and methods that
// make mutating calls to the chain for making challenge moves.
type ChallengeVertex interface {
	// Getters.
	Id() [32]byte
	SequenceNum() VertexSequenceNumber
	Status() AssertionState
	HistoryCommitment() util.HistoryCommitment
	MiniStaker() common.Address
	Prev(ctx context.Context, tx ActiveTx) (util.Option[ChallengeVertex], error)
	GetSubChallenge(ctx context.Context, tx ActiveTx) (util.Option[Challenge], error)
	HasConfirmedSibling(
		ctx context.Context,
		tx ActiveTx,
	) (bool, error)

	// Presumptive status / timer readers.
	EligibleForNewSuccessor(ctx context.Context, tx ActiveTx) (bool, error)
	IsPresumptiveSuccessor(ctx context.Context, tx ActiveTx) (bool, error)
	PresumptiveSuccessor(
		ctx context.Context, tx ActiveTx,
	) (util.Option[ChallengeVertex], error)
	PsTimer(ctx context.Context, tx ActiveTx) (uint64, error)
	ChessClockExpired(
		ctx context.Context,
		tx ActiveTx,
		challengePeriodSeconds time.Duration,
	) (bool, error)
	ChildrenAreAtOneStepFork(ctx context.Context, tx ActiveTx) (bool, error)

	// Mutating calls for challenge moves.
	CreateSubChallenge(
		ctx context.Context,
		tx ActiveTx,
	) (Challenge, error)
	Bisect(
		ctx context.Context,
		tx ActiveTx,
		history util.HistoryCommitment,
		proof []common.Hash,
	) (ChallengeVertex, error)
	Merge(
		ctx context.Context,
		tx ActiveTx,
		mergingToHistory util.HistoryCommitment,
		proof []common.Hash,
	) (ChallengeVertex, error)

	// Mutating calls for confirmations.
	ConfirmForPsTimer(ctx context.Context, tx ActiveTx) error
	ConfirmForChallengeDeadline(ctx context.Context, tx ActiveTx) error
	ConfirmForSubChallengeWin(ctx context.Context, tx ActiveTx) error
}

type AssertionChainEvent interface {
	IsAssertionChainEvent() bool // this method is just a marker that the type intends to be an AssertionChainEvent
}

type genericAssertionChainEvent struct{}

func (ev *genericAssertionChainEvent) IsAssertionChainEvent() bool { return true }

type CreateLeafEvent struct {
	genericAssertionChainEvent
	PrevSeqNum    AssertionSequenceNumber
	SeqNum        AssertionSequenceNumber
	PrevHeight    uint64
	PrevStateHash common.Hash
	Height        uint64
	StateHash     common.Hash
	Validator     common.Address
}

type ConfirmEvent struct {
	genericAssertionChainEvent
	SeqNum AssertionSequenceNumber
}

type RejectEvent struct {
	genericAssertionChainEvent
	SeqNum AssertionSequenceNumber
}

type StartChallengeEvent struct {
	genericAssertionChainEvent
	ParentSeqNum    AssertionSequenceNumber
	ParentStateHash common.Hash
	ParentStaker    common.Address
	Validator       common.Address
}

type SetBalanceEvent struct {
	genericAssertionChainEvent
	Addr       common.Address
	OldBalance *big.Int
	NewBalance *big.Int
}

type ChallengeEvent interface {
	IsChallengeEvent() bool // this method is just a marker that the type intends to be a ChallengeEvent
	ParentStateCommitmentHash() common.Hash
	ValidatorAddress() common.Address
}

type genericChallengeEvent struct{}

func (ev *genericChallengeEvent) IsChallengeEvent() bool { return true }

type ChallengeLeafEvent struct {
	genericChallengeEvent
	ParentSeqNum      VertexSequenceNumber
	SequenceNum       VertexSequenceNumber
	WinnerIfConfirmed AssertionSequenceNumber
	ParentStateCommit util.StateCommitment
	History           util.HistoryCommitment
	BecomesPS         bool
	Validator         common.Address
}

type ChallengeBisectEvent struct {
	genericChallengeEvent
	FromSequenceNum   VertexSequenceNumber // previously existing vertex
	SequenceNum       VertexSequenceNumber // newly created vertex
	ParentStateCommit util.StateCommitment
	ToHistory         util.HistoryCommitment
	FromHistory       util.HistoryCommitment
	BecomesPS         bool
	Validator         common.Address
}

type ChallengeMergeEvent struct {
	genericChallengeEvent
	ToHistory            util.HistoryCommitment
	FromHistory          util.HistoryCommitment
	ParentStateCommit    util.StateCommitment
	DeeperSequenceNum    VertexSequenceNumber
	ShallowerSequenceNum VertexSequenceNumber
	BecomesPS            bool
	Validator            common.Address
}

func (c *ChallengeLeafEvent) ParentStateCommitmentHash() common.Hash {
	return c.ParentStateCommit.Hash()
}

func (c *ChallengeBisectEvent) ParentStateCommitmentHash() common.Hash {
	return c.ParentStateCommit.Hash()
}

func (c *ChallengeMergeEvent) ParentStateCommitmentHash() common.Hash {
	return c.ParentStateCommit.Hash()
}

func (c *ChallengeLeafEvent) ValidatorAddress() common.Address {
	return c.Validator
}

func (c *ChallengeBisectEvent) ValidatorAddress() common.Address {
	return c.Validator
}

func (c *ChallengeMergeEvent) ValidatorAddress() common.Address {
	return c.Validator
}

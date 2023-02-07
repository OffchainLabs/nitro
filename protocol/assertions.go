package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
)

var (
	AssertionStake       = big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(100))
	ChallengeVertexStake = big.NewInt(params.Ether)

	ErrWrongChain             = errors.New("wrong chain")
	ErrParentDoesNotExist     = errors.New("assertion's parent does not exist on-chain")
	ErrInvalidOp              = errors.New("invalid operation")
	ErrChallengeAlreadyExists = errors.New("challenge already exists on leaf")
	ErrCannotChallengeOwnLeaf = errors.New("cannot challenge own leaf")
	ErrInvalidHeight          = errors.New("invalid block height")
	ErrVertexAlreadyExists    = errors.New("vertex already exists")
	ErrWrongState             = errors.New("vertex state does not allow this operation")
	ErrWrongPredecessorState  = errors.New("predecessor state does not allow this operation")
	ErrNotYet                 = errors.New("deadline has not yet passed")
	ErrNoWinnerYet            = errors.New("challenges does not yet have a winner assertion")
	ErrPastDeadline           = errors.New("deadline has passed")
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrNotImplemented         = errors.New("not yet implemented")
	ErrNoLastLeafProof        = errors.New("history commitment must provide a last leaf proof")
	ErrWrongFirstLeaf         = errors.New("first leaf of history does not match required state root")
	ErrWrongLastLeaf          = errors.New("last leaf of history does not match required state root")
	ErrProofFailsToVerify     = errors.New("Merkle proof fails to verify for last state of history commitment")
)

// ChallengeCommitHash returns the hash of the state commitment of the challenge.
type ChallengeCommitHash common.Hash

// VertexCommitHash returns the hash of the history commitment of the vertex.
type VertexCommitHash common.Hash

// AssertionSequenceNumber is a monotonically increasing index, starting from 0, for the creation
// of in a collection such as assertions.
type AssertionSequenceNumber uint64

// VertexSequenceNumber is a monotonically increasing index, starting from 0, for the creation
// of in a collection such as challenge vertexes.
type VertexSequenceNumber uint64

// OnChainProtocol defines an interface for interacting with the smart contract implementation
// of the assertion protocol, with methods to issue mutating transactions, make eth calls, create
// leafs in the protocol, issue challenges, and subscribe to chain events wrapped in simple abstractions.
type OnChainProtocol interface {
	ChainReadWriter
	AssertionManager
}

// ChainReadWriter can make mutating and non-mutating calls to the blockchain.
type ChainReadWriter interface {
	ChainReader
	ChainWriter
	EventProvider
}

// ChainReader can make non-mutating calls to the on-chain protocol.
type ChainReader interface {
	Call(clo func(*ActiveTx, OnChainProtocol) error) error
}

// ChainWriter can make mutating calls to the on-chain protocol.
type ChainWriter interface {
	Tx(clo func(*ActiveTx, OnChainProtocol) error) error
}

// EventProvider allows subscribing to chain events for the on-chain protocol.
type EventProvider interface {
	SubscribeChainEvents(ctx context.Context, ch chan<- AssertionChainEvent)
	SubscribeChallengeEvents(ctx context.Context, ch chan<- ChallengeEvent)
}
type (
	// AssertionManager allows the creation of new leaves for a Staker with a State Commitment
	// and a previous assertion.
	AssertionManager interface {
		Inbox() *Inbox
		NumAssertions(tx *ActiveTx) uint64
		AssertionBySequenceNum(tx *ActiveTx, seqNum AssertionSequenceNumber) (*Assertion, error)
		ChallengeByCommitHash(tx *ActiveTx, commitHash ChallengeCommitHash) (*Challenge, error)
		ChallengeVertexByCommitHash(tx *ActiveTx, challenge ChallengeCommitHash, vertex VertexCommitHash) (*ChallengeVertex, error)
		IsAtOneStepFork(
			tx *ActiveTx,
			challengeCommitHash ChallengeCommitHash,
			vertexCommit util.HistoryCommitment,
			vertexParentCommit util.HistoryCommitment,
		) (bool, error)
		ChallengePeriodLength(tx *ActiveTx) time.Duration
		LatestConfirmed(*ActiveTx) *Assertion
		CreateLeaf(tx *ActiveTx, prev *Assertion, commitment util.StateCommitment, staker common.Address) (*Assertion, error)
		TimeReference() util.TimeReference
	}
)

type ChallengeInterface interface {
	RootAssertion() *Assertion
	Completed(tx *ActiveTx) bool
	HasConfirmedSibling(tx *ActiveTx, vertex ChallengeVertexInterface) bool
	RootVertex() ChallengeVertexInterface
	ParentStateCommitment() util.StateCommitment
	Winner(tx *ActiveTx) (*Assertion, error)
	AddLeaf(tx *ActiveTx, assertion *Assertion, history util.HistoryCommitment, validator common.Address, ) (ChallengeVertexInterface, error)
	GetWinnerVertex() util.Option[ChallengeVertexInterface]
	HasEnded(chain *AssertionChain) bool
	GetChallengeType() ChallengeType
	GetCreationTime() time.Time
}

type ChallengeVertexInterface interface {
	ConfirmForPsTimer(tx *ActiveTx) error
	ConfirmForChallengeDeadline(tx *ActiveTx) error
	ConfirmForSubChallengeWin(tx *ActiveTx) error
	IsPresumptiveSuccessor() bool
	Bisect(tx *ActiveTx, history util.HistoryCommitment, proof []common.Hash, validator common.Address) (ChallengeVertexInterface, error)
	Merge(tx *ActiveTx, mergingTo ChallengeVertexInterface, proof []common.Hash, validator common.Address) error
	ChessClockExpired(challengePeriod time.Duration) bool
	EligibleForNewSuccessor() bool
	GetPrev() util.Option[ChallengeVertexInterface]
	GetStatus() AssertionState
	GetSubChallenge() util.Option[ChallengeInterface]
	GetPsTimer() *util.CountUpTimer
	GetCommitment() util.HistoryCommitment
	GetValidator() common.Address
	GetSequenceNum() VertexSequenceNumber
	GetChallenge() util.Option[ChallengeInterface]
	GetPresumptiveSuccessor() util.Option[ChallengeVertexInterface]
}
type AssertionChain struct {
	mutex                         sync.RWMutex
	timeReference                 util.TimeReference
	challengePeriod               time.Duration
	latestConfirmed               AssertionSequenceNumber
	assertions                    []*Assertion
	assertionsBySeqNum            map[common.Hash]AssertionSequenceNumber
	challengeVerticesByCommitHash map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex
	challengesByCommitHash        map[ChallengeCommitHash]*Challenge
	balances                      *util.MapWithDefault[common.Address, *big.Int]
	feed                          *EventFeed[AssertionChainEvent]
	challengesFeed                *EventFeed[ChallengeEvent]
	inbox                         *Inbox
}

const (
	DeadTxStatus = iota
	ReadOnlyTxStatus
	ReadWriteTxStatus
)

// ActiveTx is a transaction that is currently being processed.
type ActiveTx struct {
	TxStatus int
}

// verifyRead is a helper function to verify that the transaction is read-only.
func (tx *ActiveTx) verifyRead() {
	if tx.TxStatus == DeadTxStatus {
		panic("tried to read chain after call ended")
	}
}

// verifyReadWrite is a helper function to verify that the transaction is read-write.
func (tx *ActiveTx) verifyReadWrite() {
	if tx.TxStatus != ReadWriteTxStatus {
		panic("tried to modify chain in read-only call")
	}
}

// Tx enables a mutating call to the on-chain protocol.
func (chain *AssertionChain) Tx(clo func(tx *ActiveTx, p OnChainProtocol) error) error {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	err := clo(tx, chain)
	tx.TxStatus = DeadTxStatus
	return err
}

// Call enables a non-mutating call to the on-chain protocol.
func (chain *AssertionChain) Call(clo func(tx *ActiveTx, p OnChainProtocol) error) error {
	chain.mutex.RLock()
	defer chain.mutex.RUnlock()
	tx := &ActiveTx{TxStatus: ReadOnlyTxStatus}
	err := clo(tx, chain)
	tx.TxStatus = DeadTxStatus
	return err
}

const (
	PendingAssertionState = iota
	ConfirmedAssertionState
	RejectedAssertionState
)

// AssertionState is a type used to represent the state of an assertion.
type AssertionState int

// Assertion represents an assertion in the protocol.
type Assertion struct {
	SequenceNum             AssertionSequenceNumber `json:"sequence_num"`
	StateCommitment         util.StateCommitment    `json:"state_commitment"`
	Staker                  util.Option[common.Address]
	Prev                    util.Option[*Assertion]
	chain                   *AssertionChain
	status                  AssertionState
	isFirstChild            bool
	firstChildCreationTime  util.Option[time.Time]
	secondChildCreationTime util.Option[time.Time]
	challenge               util.Option[*Challenge]
}

// NewAssertionChain creates a new AssertionChain.
func NewAssertionChain(ctx context.Context, timeRef util.TimeReference, challengePeriod time.Duration) *AssertionChain {
	genesis := &Assertion{
		chain:       nil,
		status:      ConfirmedAssertionState,
		SequenceNum: 0,
		StateCommitment: util.StateCommitment{
			Height:    0,
			StateRoot: common.Hash{},
		},
		Prev:                    util.None[*Assertion](),
		isFirstChild:            false,
		firstChildCreationTime:  util.None[time.Time](),
		secondChildCreationTime: util.None[time.Time](),
		challenge:               util.None[*Challenge](),
		Staker:                  util.None[common.Address](),
	}

	genesisKey := crypto.Keccak256Hash(
		binary.BigEndian.AppendUint64(
			genesis.StateCommitment.Hash().Bytes(),
			math.MaxUint64,
		),
	)
	assertionsSeen := map[common.Hash]AssertionSequenceNumber{
		genesisKey: 0,
	}
	chain := &AssertionChain{
		mutex:                         sync.RWMutex{},
		timeReference:                 timeRef,
		challengePeriod:               challengePeriod,
		challengesByCommitHash:        make(map[ChallengeCommitHash]*Challenge),
		challengeVerticesByCommitHash: make(map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex),
		latestConfirmed:               0,
		assertions:                    []*Assertion{genesis},
		balances:                      util.NewMapWithDefaultAdvanced[common.Address, *big.Int](common.Big0, func(x *big.Int) bool { return x.Sign() == 0 }),
		feed:                          NewEventFeed[AssertionChainEvent](ctx),
		challengesFeed:                NewEventFeed[ChallengeEvent](ctx),
		inbox:                         NewInbox(ctx),
		assertionsBySeqNum:            assertionsSeen,
	}
	genesis.chain = chain
	return chain
}

// TimeReference returns the time reference used by the chain.
func (chain *AssertionChain) TimeReference() util.TimeReference {
	return chain.timeReference
}

// Inbox returns the inbox used by the chain.
func (chain *AssertionChain) Inbox() *Inbox {
	return chain.inbox
}

// GetBalance returns the balance of the given address.
func (chain *AssertionChain) GetBalance(tx *ActiveTx, addr common.Address) *big.Int {
	tx.verifyRead()
	return chain.balances.Get(addr)
}

// SetBalance sets the balance of the given address.
func (chain *AssertionChain) SetBalance(tx *ActiveTx, addr common.Address, balance *big.Int) {
	tx.verifyReadWrite()
	oldBalance := chain.balances.Get(addr)
	chain.balances.Set(addr, balance)
	chain.feed.Append(&SetBalanceEvent{Addr: addr, OldBalance: oldBalance, NewBalance: balance})
}

// AddToBalance adds the given amount to the balance of the given address.
func (chain *AssertionChain) AddToBalance(tx *ActiveTx, addr common.Address, amount *big.Int) {
	tx.verifyReadWrite()
	chain.SetBalance(tx, addr, new(big.Int).Add(chain.GetBalance(tx, addr), amount))
}

// DeductFromBalance deducts the given amount from the balance of the given address.
func (chain *AssertionChain) DeductFromBalance(tx *ActiveTx, addr common.Address, amount *big.Int) error {
	tx.verifyReadWrite()
	balance := chain.GetBalance(tx, addr)
	if balance.Cmp(amount) < 0 {
		return errors.Wrapf(ErrInsufficientBalance, "%s < %s", balance.String(), amount.String())
	}
	chain.SetBalance(tx, addr, new(big.Int).Sub(balance, amount))
	return nil
}

// ChallengePeriodLength returns the length of the challenge period.
func (chain *AssertionChain) ChallengePeriodLength(tx *ActiveTx) time.Duration {
	tx.verifyRead()
	return chain.challengePeriod
}

// LatestConfirmed returns the latest confirmed assertion.
func (chain *AssertionChain) LatestConfirmed(tx *ActiveTx) *Assertion {
	tx.verifyRead()
	return chain.assertions[chain.latestConfirmed]
}

// NumAssertions returns the number of assertions in the chain.
func (chain *AssertionChain) NumAssertions(tx *ActiveTx) uint64 {
	tx.verifyRead()
	return uint64(len(chain.assertions))
}

// AssertionBySequenceNum returns the assertion with the given sequence number.
func (chain *AssertionChain) AssertionBySequenceNum(tx *ActiveTx, seqNum AssertionSequenceNumber) (*Assertion, error) {
	tx.verifyRead()
	if seqNum >= AssertionSequenceNumber(len(chain.assertions)) {
		return nil, fmt.Errorf("assertion sequence out of range %d >= %d", seqNum, len(chain.assertions))
	}
	return chain.assertions[seqNum], nil
}

// IsAtOneStepFork when given a challenge vertex's history commitment
// along with its parent's, will check other challenge vertices in that challenge
// to verify there are > 1 vertices that are one height away from their parent.
func (chain *AssertionChain) IsAtOneStepFork(
	tx *ActiveTx,
	challengeCommitHash ChallengeCommitHash,
	vertexCommit util.HistoryCommitment,
	vertexParentCommit util.HistoryCommitment,
) (bool, error) {
	tx.verifyRead()
	if vertexCommit.Height != vertexParentCommit.Height+1 {
		return false, nil
	}
	vertices, ok := chain.challengeVerticesByCommitHash[challengeCommitHash]
	if !ok {
		return false, fmt.Errorf("challenge vertices not found for assertion with state commit hash %#x", challengeCommitHash)
	}
	parentCommitHash := VertexCommitHash(vertexParentCommit.Hash())
	return verticesContainOneStepFork(vertices, parentCommitHash), nil
}

// Check if a vertices with a matching parent commitment hash are at a one-step-fork from their parent.
// First, we filter out vertices with the specified parent commit hash, then check that all of the
// matching vertices are one-step away from their parent.
func verticesContainOneStepFork(vertices map[VertexCommitHash]*ChallengeVertex, parentCommitHash VertexCommitHash) bool {
	if len(vertices) < 2 {
		return false
	}
	childVertices := make([]*ChallengeVertex, 0)
	for _, v := range vertices {
		if v.Prev.IsNone() {
			continue
		}
		// We only check vertices that have a matching parent commit hash.
		vParentHash := VertexCommitHash(v.Prev.Unwrap().GetCommitment().Hash())
		if vParentHash == parentCommitHash {
			childVertices = append(childVertices, v)
		}
	}
	if len(childVertices) < 2 {
		return false
	}
	for _, vertex := range childVertices {
		if !isOneStepAwayFromParent(vertex) {
			return false
		}
	}
	return true
}

func isOneStepAwayFromParent(vertex *ChallengeVertex) bool {
	if vertex.Prev.IsNone() {
		return false
	}
	return vertex.Commitment.Height == vertex.Prev.Unwrap().GetCommitment().Height+1
}

// ChallengeVertexByCommitHash returns the challenge vertex with the given commit hash.
func (chain *AssertionChain) ChallengeVertexByCommitHash(
	tx *ActiveTx, challengeHash ChallengeCommitHash, vertexHash VertexCommitHash,
) (*ChallengeVertex, error) {
	tx.verifyRead()
	vertices, ok := chain.challengeVerticesByCommitHash[challengeHash]
	if !ok {
		return nil, fmt.Errorf("challenge vertices not found for assertion with state commit hash %#x", challengeHash)
	}
	vertex, ok := vertices[vertexHash]
	if !ok {
		return nil, fmt.Errorf("challenge vertex with sequence number not found %#x", vertexHash)
	}
	return vertex, nil
}

// ChallengeByCommitHash returns the challenge with the given commit hash.
func (chain *AssertionChain) ChallengeByCommitHash(tx *ActiveTx, commitHash ChallengeCommitHash) (*Challenge, error) {
	tx.verifyRead()
	chal, ok := chain.challengesByCommitHash[commitHash]
	if !ok {
		return nil, errors.Wrapf(ErrVertexAlreadyExists, fmt.Sprintf("Hash: %s", commitHash))
	}
	return chal, nil
}

// SubscribeChainEvents subscribes to chain events.
func (chain *AssertionChain) SubscribeChainEvents(ctx context.Context, ch chan<- AssertionChainEvent) {
	chain.feed.Subscribe(ctx, ch)
}

// SubscribeChallengeEvents subscribes to challenge events.
func (chain *AssertionChain) SubscribeChallengeEvents(ctx context.Context, ch chan<- ChallengeEvent) {
	chain.challengesFeed.Subscribe(ctx, ch)
}

// CreateLeaf creates a new leaf assertion.
func (chain *AssertionChain) CreateLeaf(tx *ActiveTx, prev *Assertion, commitment util.StateCommitment, staker common.Address) (*Assertion, error) {
	tx.verifyReadWrite()
	if prev.chain != chain {
		return nil, ErrWrongChain
	}
	if prev.StateCommitment.Height >= commitment.Height {
		return nil, ErrInvalidOp
	}

	// Ensure the assertion being created is not a duplicate.
	if _, ok := chain.assertionsBySeqNum[commitment.Hash()]; ok {
		return nil, ErrVertexAlreadyExists
	}

	if !prev.Prev.IsNone() {
		// The parent must exist on-chain.
		prevSeqNum, ok := chain.assertionsBySeqNum[prev.StateCommitment.Hash()]
		if !ok {
			return nil, ErrParentDoesNotExist
		}
		// Parent sequence number must be < the new assertion's assigned sequence number.
		if prevSeqNum >= AssertionSequenceNumber(uint64(len(chain.assertions))) {
			return nil, ErrParentDoesNotExist
		}
	}

	if err := prev.Staker.IfLet(
		func(oldStaker common.Address) error {
			if staker != oldStaker {
				if err := chain.DeductFromBalance(tx, staker, AssertionStake); err != nil {
					return err
				}
				chain.AddToBalance(tx, oldStaker, AssertionStake)
				prev.Staker = util.None[common.Address]()
			}
			return nil
		},
		func() error {
			if err := chain.DeductFromBalance(tx, staker, AssertionStake); err != nil {
				return err
			}
			return nil

		},
	); err != nil {
		return nil, err
	}

	leaf := &Assertion{
		chain:                   chain,
		status:                  PendingAssertionState,
		SequenceNum:             AssertionSequenceNumber(len(chain.assertions)),
		StateCommitment:         commitment,
		Prev:                    util.Some(prev),
		isFirstChild:            prev.firstChildCreationTime.IsNone(),
		firstChildCreationTime:  util.None[time.Time](),
		secondChildCreationTime: util.None[time.Time](),
		challenge:               util.None[*Challenge](),
		Staker:                  util.Some(staker),
	}
	if prev.firstChildCreationTime.IsNone() {
		prev.firstChildCreationTime = util.Some(chain.timeReference.Get())
	} else if prev.secondChildCreationTime.IsNone() {
		prev.secondChildCreationTime = util.Some(chain.timeReference.Get())
	}
	chain.assertions = append(chain.assertions, leaf)
	chain.assertionsBySeqNum[commitment.Hash()] = leaf.SequenceNum
	chain.feed.Append(&CreateLeafEvent{
		PrevStateCommitment: prev.StateCommitment,
		PrevSeqNum:          prev.SequenceNum,
		SeqNum:              leaf.SequenceNum,
		StateCommitment:     leaf.StateCommitment,
		Validator:           staker,
	})
	return leaf, nil
}

// RejectForPrev rejects the assertion and emits the information through feed. It moves assertion to `RejectedAssertionState` state.
func (a *Assertion) RejectForPrev(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("State: %d", a.status))
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	if a.Prev.Unwrap().status != RejectedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", a.Prev.Unwrap().status))
	}
	a.status = RejectedAssertionState
	a.chain.feed.Append(&RejectEvent{
		SeqNum: a.SequenceNum,
	})
	return nil
}

// RejectForLoss rejects the assertion and emits the information through feed. It moves assertion to `RejectedAssertionState` state.
func (a *Assertion) RejectForLoss(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("State: %d", a.status))
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	chal := a.Prev.Unwrap().challenge
	if chal.IsNone() {
		return util.ErrOptionIsEmpty
	}
	winner, err := chal.Unwrap().Winner(tx)
	if err != nil {
		return err
	}
	if winner == a {
		return ErrInvalidOp
	}
	a.status = RejectedAssertionState
	a.chain.feed.Append(&RejectEvent{
		SeqNum: a.SequenceNum,
	})
	return nil
}

// ConfirmNoRival confirms that there is no rival for the assertion and moves the assertion to `ConfirmedAssertionState` state.
func (a *Assertion) ConfirmNoRival(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("State: %d", a.status))
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	prev := a.Prev.Unwrap()
	if prev.status != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", a.Prev.Unwrap().status))
	}
	if !prev.secondChildCreationTime.IsNone() {
		return ErrInvalidOp
	}
	if !a.chain.timeReference.Get().After(prev.firstChildCreationTime.Unwrap().Add(a.chain.challengePeriod)) {
		return errors.Wrapf(ErrNotYet, fmt.Sprintf("%d > %d", a.chain.timeReference.Get().Unix(), prev.firstChildCreationTime.Unwrap().Add(a.chain.challengePeriod).Unix()))
	}
	a.status = ConfirmedAssertionState
	a.chain.latestConfirmed = a.SequenceNum
	a.chain.feed.Append(&ConfirmEvent{
		SeqNum: a.SequenceNum,
	})

	if !a.Staker.IsNone() && a.firstChildCreationTime.IsNone() {
		a.chain.AddToBalance(tx, a.Staker.Unwrap(), AssertionStake)
		a.Staker = util.None[common.Address]()
	}
	return nil
}

// ConfirmForWin confirms that the assertion is the WinnerAssertion of the challenge and moves the assertion to `ConfirmedAssertionState` state.
func (a *Assertion) ConfirmForWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("State: %d", a.status))
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	prev := a.Prev.Unwrap()
	if prev.status != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", a.Prev.Unwrap().status))
	}
	if prev.challenge.IsNone() {
		return ErrWrongPredecessorState
	}
	winner, err := prev.challenge.Unwrap().Winner(tx)
	if err != nil {
		return err
	}
	if winner != a {
		return ErrInvalidOp
	}
	a.status = ConfirmedAssertionState
	a.chain.latestConfirmed = a.SequenceNum
	a.chain.feed.Append(&ConfirmEvent{
		SeqNum: a.SequenceNum,
	})
	return nil
}

type ChallengeType uint

const (
	NoChallengeType    ChallengeType = iota
	BlockChallenge                   = 1
	BigStepChallenge                 = 2
	SmallStepChallenge               = 3
)

// Challenge created by an assertion.
type Challenge struct {
	rootAssertion          util.Option[*Assertion]
	WinnerAssertion        util.Option[*Assertion]
	WinnerVertex           util.Option[ChallengeVertexInterface]
	rootVertex             util.Option[ChallengeVertexInterface]
	leafVertexCount        uint64
	creationTime           time.Time
	includedHistories      map[common.Hash]bool
	currentVertexSeqNumber VertexSequenceNumber
	challengePeriod        time.Duration
	challengeType          ChallengeType
}

// CreateChallenge creates a challenge for the assertion and moves the assertion to `ChallengedAssertionState` state.
func (a *Assertion) CreateChallenge(tx *ActiveTx, ctx context.Context, validator common.Address) (ChallengeInterface, error) {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState && a.chain.LatestConfirmed(tx) != a {
		return nil, errors.Wrapf(ErrWrongState, fmt.Sprintf("State: %d, Confirmed status: %v", a.status, a.chain.LatestConfirmed(tx) != a))
	}
	if !a.challenge.IsNone() {
		return nil, ErrChallengeAlreadyExists
	}
	if a.secondChildCreationTime.IsNone() {
		return nil, ErrInvalidOp
	}
	currSeqNumber := VertexSequenceNumber(0)
	rootVertex := &ChallengeVertex{
		Challenge:   util.None[ChallengeInterface](),
		SequenceNum: currSeqNumber,
		isLeaf:      false,
		Status:      ConfirmedAssertionState,
		Commitment: util.HistoryCommitment{
			Height: 0,
			Merkle: common.Hash{},
		},
		Prev:                 util.None[ChallengeVertexInterface](),
		PresumptiveSuccessor: util.None[ChallengeVertexInterface](),
		PsTimer:              util.NewCountUpTimer(a.chain.timeReference),
		SubChallenge:         util.None[ChallengeInterface](),
	}

	chal := &Challenge{
		rootAssertion:     util.Some(a),
		WinnerAssertion:   util.None[*Assertion](),
		WinnerVertex:      util.None[ChallengeVertexInterface](),
		rootVertex:        util.Some(ChallengeVertexInterface(rootVertex)),
		creationTime:      a.chain.timeReference.Get(),
		includedHistories: make(map[common.Hash]bool),
		challengePeriod:   a.chain.challengePeriod,
		challengeType:     BlockChallenge,
	}
	rootVertex.Challenge = util.Some(ChallengeInterface(chal))
	chal.includedHistories[rootVertex.Commitment.Hash()] = true
	a.challenge = util.Some(chal)
	parentStaker := common.Address{}
	if !a.Staker.IsNone() {
		parentStaker = a.Staker.Unwrap()
	}
	a.chain.feed.Append(&StartChallengeEvent{
		ParentSeqNum:          a.SequenceNum,
		ParentStateCommitment: a.StateCommitment,
		ParentStaker:          parentStaker,
		Validator:             validator,
	})

	challengeID := ChallengeCommitHash(a.StateCommitment.Hash())
	a.chain.challengesByCommitHash[challengeID] = chal
	a.chain.challengeVerticesByCommitHash[challengeID] = map[VertexCommitHash]*ChallengeVertex{VertexCommitHash(rootVertex.Commitment.Hash()): rootVertex}

	return chal, nil
}

// Parentutil.StateCommitment returns the state commitment of the parent assertion.
func (c *Challenge) ParentStateCommitment() util.StateCommitment {
	if c.rootAssertion.IsNone() {
		return util.StateCommitment{}
	}
	return c.rootAssertion.Unwrap().StateCommitment
}

// AssertionSeqNumber returns the sequence number of the assertion that created the challenge.
func (c *Challenge) AssertionSeqNumber() AssertionSequenceNumber {
	return c.rootAssertion.Unwrap().SequenceNum
}

// RootAssertion returns the root assertion of challenge
func (c *Challenge) RootAssertion() *Assertion {
	return c.rootAssertion.Unwrap()
}

func (c *Challenge) GetWinnerVertex() util.Option[ChallengeVertexInterface] {
	return c.WinnerVertex
}

func (c *Challenge) GetChallengeType() ChallengeType {
	return c.challengeType
}

func (c *Challenge) GetCreationTime() time.Time {
	return c.creationTime
}

// RootVertex returns the root vertex of challenge
func (c *Challenge) RootVertex() ChallengeVertexInterface {
	return c.rootVertex.Unwrap()
}

// AddLeaf adds a new leaf to the challenge.
func (c *Challenge) AddLeaf(
	tx *ActiveTx,
	assertion *Assertion,
	history util.HistoryCommitment,
	validator common.Address,
) (ChallengeVertexInterface, error) {
	tx.verifyReadWrite()
	if assertion.Prev.IsNone() {
		return nil, ErrInvalidOp
	}
	prev := assertion.Prev.Unwrap()
	if prev != c.rootAssertion.Unwrap() {
		return nil, ErrInvalidOp
	}
	if c.Completed(tx) {
		return nil, ErrWrongState
	}
	if !c.rootVertex.Unwrap().EligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if c.includedHistories[history.Hash()] {
		return nil, errors.Wrapf(ErrVertexAlreadyExists, fmt.Sprintf("Hash: %s", history.Hash().String()))
	}
	if err := c.rootAssertion.Unwrap().chain.DeductFromBalance(tx, validator, ChallengeVertexStake); err != nil {
		return nil, errors.Wrapf(ErrInsufficientBalance, err.Error())
	}

	if !historyProvidesLastLeafProof(history) {
		return nil, ErrNoLastLeafProof
	}

	// The first leaf in the history commitment must be the
	// same as the previous vertex's history state root.
	if prev.StateCommitment.Height != 0 && prev.StateCommitment.StateRoot != history.FirstLeaf {
		return nil, ErrWrongFirstLeaf
	}

	// The last leaf claimed in the history commitment must be the
	// state root of the assertion we are adding a leaf for.
	if assertion.StateCommitment.StateRoot != history.LastLeaf {
		return nil, ErrWrongLastLeaf
	}

	// Assert the history commitment's height is equal to the
	// assertion.height - assertion.prev.height
	if prev.StateCommitment.Height >= assertion.StateCommitment.Height {
		return nil, errors.Wrapf(
			ErrInvalidHeight,
			"previous assertion's height %d, must be less than %d",
			prev.StateCommitment.Height,
			assertion.StateCommitment.Height,
		)
	}
	expectedHeight := assertion.StateCommitment.Height - prev.StateCommitment.Height
	if history.Height != expectedHeight {
		return nil, errors.Wrapf(
			ErrInvalidHeight,
			"history height does not match expected value %d != %d",
			history.Height,
			expectedHeight,
		)
	}

	// The validator must provide a history commitment over
	// a series of states where the last state must be proven to be
	// one corresponding to the assertion specified.
	if err := util.VerifyPrefixProof(
		history.LastLeafPrefix.Unwrap(),
		history.Normalized().Unwrap(),
		history.LastLeafProof,
	); err != nil {
		return nil, ErrProofFailsToVerify
	}

	chain := assertion.chain
	timer := util.NewCountUpTimer(chain.timeReference)
	if assertion.isFirstChild {
		delta := prev.secondChildCreationTime.Unwrap().Sub(prev.firstChildCreationTime.Unwrap())
		timer.Set(delta)
	}
	nextSeqNumber := c.currentVertexSeqNumber + 1
	leaf := &ChallengeVertex{
		Challenge:            util.Some(ChallengeInterface(c)),
		SequenceNum:          nextSeqNumber,
		Validator:            validator,
		isLeaf:               true,
		Status:               PendingAssertionState,
		Commitment:           history,
		Prev:                 c.rootVertex,
		PresumptiveSuccessor: util.None[ChallengeVertexInterface](),
		PsTimer:              timer,
		SubChallenge:         util.None[ChallengeInterface](),
		winnerIfConfirmed:    util.Some(assertion),
	}
	c.currentVertexSeqNumber = nextSeqNumber
	c.rootVertex.Unwrap().(*ChallengeVertex).maybeNewPresumptiveSuccessor(leaf)
	c.rootAssertion.Unwrap().chain.challengesFeed.Append(&ChallengeLeafEvent{
		ParentSeqNum:      leaf.Prev.Unwrap().GetSequenceNum(),
		SequenceNum:       leaf.SequenceNum,
		WinnerIfConfirmed: assertion.SequenceNum,
		History:           history,
		BecomesPS:         leaf.Prev.Unwrap().(*ChallengeVertex).PresumptiveSuccessor.Unwrap() == leaf,
		Validator:         validator,
	})
	c.includedHistories[history.Hash()] = true
	h := ChallengeCommitHash(c.rootAssertion.Unwrap().StateCommitment.Hash())
	c.rootAssertion.Unwrap().chain.challengesByCommitHash[h] = c
	c.rootAssertion.Unwrap().chain.challengeVerticesByCommitHash[h][VertexCommitHash(leaf.Commitment.Hash())] = leaf
	c.leafVertexCount++

	return leaf, nil
}

// Completed returns true if the challenge is completed.
func (c *Challenge) Completed(tx *ActiveTx) bool {
	tx.verifyRead()
	return !c.WinnerAssertion.IsNone()
}

// CreationTime returns the time the challenge was created.
func (c *Challenge) CreationTime(tx *ActiveTx) time.Time {
	tx.verifyRead()
	return c.creationTime
}

// Winner returns the winning assertion if the challenge is completed.
func (c *Challenge) Winner(tx *ActiveTx) (*Assertion, error) {
	tx.verifyRead()
	if c.WinnerAssertion.IsNone() {
		return nil, ErrNoWinnerYet
	}
	return c.WinnerAssertion.Unwrap(), nil
}

// HasConfirmedSibling returns true if another sibling vertex is confirmed.
func (c *Challenge) HasConfirmedSibling(tx *ActiveTx, vertex ChallengeVertexInterface) bool {
	tx.verifyRead()

	if c.rootAssertion.IsNone() {
		return false
	}
	vertices, ok := c.rootAssertion.Unwrap().chain.challengeVerticesByCommitHash[ChallengeCommitHash(c.ParentStateCommitment().Hash())]
	if !ok {
		return false
	}

	if vertex.GetPrev().IsNone() {
		return false
	}
	parentHash := vertex.GetPrev().Unwrap().GetCommitment().Hash()
	for _, v := range vertices {
		if v.Prev.IsNone() {
			continue
		}
		// We only check vertices that have a matching parent commit hash.
		if v.Prev.Unwrap().GetCommitment().Hash() == parentHash {
			if vertex != v && v.Status == ConfirmedAssertionState {
				return true
			}
		}
	}

	return false
}

type ChallengeVertex struct {
	Commitment           util.HistoryCommitment
	Challenge            util.Option[ChallengeInterface]
	SequenceNum          VertexSequenceNumber // unique within the challenge
	Validator            common.Address
	isLeaf               bool
	Status               AssertionState
	Prev                 util.Option[ChallengeVertexInterface]
	PresumptiveSuccessor util.Option[ChallengeVertexInterface]
	PsTimer              *util.CountUpTimer
	SubChallenge         util.Option[ChallengeInterface]
	winnerIfConfirmed    util.Option[*Assertion]
}

// EligibleForNewSuccessor returns true if the vertex is eligible to have a new successor.
func (v *ChallengeVertex) EligibleForNewSuccessor() bool {
	return v.PresumptiveSuccessor.IsNone() ||
		v.PresumptiveSuccessor.Unwrap().GetPsTimer().Get() <= v.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.challengePeriod
}

// maybeNewPresumptiveSuccessor updates the presumptive successor if the given vertex is eligible.
func (v *ChallengeVertex) maybeNewPresumptiveSuccessor(succ ChallengeVertexInterface) {
	if !v.PresumptiveSuccessor.IsNone() &&
		succ.GetCommitment().Height < v.PresumptiveSuccessor.Unwrap().GetCommitment().Height {
		v.PresumptiveSuccessor.Unwrap().GetPsTimer().Stop()
		v.PresumptiveSuccessor = util.None[ChallengeVertexInterface]()
	}

	if v.PresumptiveSuccessor.IsNone() {
		v.PresumptiveSuccessor = util.Some(succ)
		succ.GetPsTimer().Start()
	}
}

// IsPresumptiveSuccessor returns true if the vertex is the presumptive successor of its parent.
func (v *ChallengeVertex) IsPresumptiveSuccessor() bool {
	return v.Prev.IsNone() || v.Prev.Unwrap().GetPresumptiveSuccessor().Unwrap() == v
}

// requiredBisectionHeight returns the height of the history commitment that must be bisectioned to prove the vertex.
func (v *ChallengeVertex) requiredBisectionHeight() (uint64, error) {
	return util.BisectionPoint(v.Prev.Unwrap().GetCommitment().Height, v.Commitment.Height)
}

// Bisect returns the bisection proof for the vertex.
func (v *ChallengeVertex) Bisect(tx *ActiveTx, history util.HistoryCommitment, proof []common.Hash, validator common.Address) (ChallengeVertexInterface, error) {
	tx.verifyReadWrite()
	if v.IsPresumptiveSuccessor() {
		return nil, ErrWrongState
	}
	if !v.Prev.Unwrap().EligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if v.Challenge.Unwrap().(*Challenge).includedHistories[history.Hash()] {
		return nil, errors.Wrapf(ErrVertexAlreadyExists, fmt.Sprintf("Hash: %s", history.Hash().String()))
	}
	bisectionHeight, err := v.requiredBisectionHeight()
	if err != nil {
		return nil, err
	}
	if bisectionHeight != history.Height {
		return nil, errors.Wrapf(ErrInvalidHeight, fmt.Sprintf("%d != %v", bisectionHeight, history))
	}
	if err := util.VerifyPrefixProof(history, v.Commitment, proof); err != nil {
		return nil, err
	}

	v.PsTimer.Stop()
	nextSeqNum := v.SequenceNum + 1
	newVertex := &ChallengeVertex{
		Challenge:            v.Challenge,
		SequenceNum:          nextSeqNum,
		Validator:            validator,
		isLeaf:               false,
		Commitment:           history,
		Prev:                 v.Prev,
		PresumptiveSuccessor: util.None[ChallengeVertexInterface](),
		PsTimer:              v.PsTimer.Clone(),
	}
	v.SequenceNum = nextSeqNum
	newVertex.maybeNewPresumptiveSuccessor(v)
	newVertex.Prev.Unwrap().(*ChallengeVertex).maybeNewPresumptiveSuccessor(newVertex)
	newVertex.Challenge.Unwrap().(*Challenge).includedHistories[history.Hash()] = true

	v.Prev = util.Some[ChallengeVertexInterface](newVertex)

	newVertex.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.challengesFeed.Append(&ChallengeBisectEvent{
		FromSequenceNum: v.SequenceNum,
		SequenceNum:     newVertex.SequenceNum,
		ToHistory:       newVertex.Commitment,
		FromHistory:     v.Commitment,
		BecomesPS:       newVertex.Prev.Unwrap().(*ChallengeVertex).PresumptiveSuccessor.Unwrap() == newVertex,
		Validator:       validator,
	})
	commitHash := ChallengeCommitHash(newVertex.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().StateCommitment.Hash())
	newVertex.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.challengeVerticesByCommitHash[commitHash][VertexCommitHash(newVertex.Commitment.Hash())] = newVertex

	return newVertex, nil
}

// Merge merges the vertex with its presumptive successor.
func (v *ChallengeVertex) Merge(tx *ActiveTx, mergingTo ChallengeVertexInterface, proof []common.Hash, validator common.Address) error {
	tx.verifyReadWrite()
	if !mergingTo.EligibleForNewSuccessor() {
		return ErrPastDeadline
	}
	// The vertex we are merging to should be the mandatory bisection point
	// of the current vertex's height and its parent's height.
	bisectionPoint, err := util.BisectionPoint(v.Prev.Unwrap().GetCommitment().Height, v.Commitment.Height)
	if err != nil {
		return err
	}
	if mergingTo.GetCommitment().Height != bisectionPoint {
		return errors.Wrapf(ErrInvalidHeight, "%d != %d", mergingTo.GetCommitment().Height, bisectionPoint)
	}
	if err := util.VerifyPrefixProof(mergingTo.GetCommitment(), v.Commitment, proof); err != nil {
		return err
	}

	v.Prev = util.Some(mergingTo)
	mergingTo.GetPsTimer().Add(v.PsTimer.Get())
	mergingTo.(*ChallengeVertex).maybeNewPresumptiveSuccessor(v)
	v.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.challengesFeed.Append(&ChallengeMergeEvent{
		DeeperSequenceNum:    v.SequenceNum,
		ShallowerSequenceNum: mergingTo.(*ChallengeVertex).SequenceNum,
		BecomesPS:            mergingTo.(*ChallengeVertex).PresumptiveSuccessor.Unwrap() == v,
		ToHistory:            mergingTo.(*ChallengeVertex).Commitment,
		FromHistory:          v.Commitment,
		Validator:            validator,
	})
	return nil
}

// ConfirmForSubChallengeWin confirms the vertex as the winner of a sub-challenge.
func (v *ChallengeVertex) ConfirmForSubChallengeWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.Status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.Status))
	}
	if v.Prev.Unwrap().GetStatus() != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", v.Prev.Unwrap().GetStatus()))
	}
	subChal := v.Prev.Unwrap().GetSubChallenge()
	if subChal.IsNone() || subChal.Unwrap().GetWinnerVertex().IsNone() {
		return ErrInvalidOp
	}
	winnerVertex := subChal.Unwrap().GetWinnerVertex().Unwrap()
	if winnerVertex != v {
		return ErrInvalidOp
	}
	v._confirm(tx)
	return nil
}

// ConfirmForPsTimer confirms the vertex as the Winner of a PsTimer.
func (v *ChallengeVertex) ConfirmForPsTimer(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.Status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.Status))
	}
	if !v.Prev.Unwrap().GetSubChallenge().IsNone() {
		return errors.Wrap(ErrInvalidOp, "predecessor contains sub-challenge")
	}
	if v.Prev.Unwrap().GetStatus() != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", v.Prev.Unwrap().GetStatus()))
	}
	if v.PsTimer.Get() <= v.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.challengePeriod {
		return errors.Wrapf(
			ErrNotYet,
			fmt.Sprintf(
				"%d <= %d",
				v.PsTimer.Get(),
				v.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.challengePeriod),
		)
	}
	v._confirm(tx)
	return nil
}

// ConfirmForChallengeDeadline confirms the vertex as the winner of a challenge deadline.
func (v *ChallengeVertex) ConfirmForChallengeDeadline(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.Status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.Status))
	}
	if !v.Prev.Unwrap().(*ChallengeVertex).SubChallenge.IsNone() {
		return errors.Wrap(ErrInvalidOp, "predecessor contains sub-challenge")
	}
	if v.Prev.Unwrap().(*ChallengeVertex).Status != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", v.Prev.Unwrap().(*ChallengeVertex).Status))
	}
	if v != v.Prev.Unwrap().(*ChallengeVertex).PresumptiveSuccessor.Unwrap() {
		return errors.Wrap(ErrInvalidOp, "Vertex is not the presumptive successor")
	}
	chain := v.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain
	chalPeriod := chain.challengePeriod
	if !chain.timeReference.Get().After(v.Challenge.Unwrap().(*Challenge).creationTime.Add(2 * chalPeriod)) {
		return errors.Wrapf(
			ErrNotYet, fmt.Sprintf(
				"%d <= %d",
				chain.timeReference.Get().Unix(),
				v.Challenge.Unwrap().(*Challenge).creationTime.Add(2*chalPeriod).Unix(),
			),
		)
	}
	v._confirm(tx)
	return nil
}

func (v *ChallengeVertex) _confirm(tx *ActiveTx) {
	v.Status = ConfirmedAssertionState
	if v.isLeaf {
		refund := big.NewInt(0)
		leafCount := int64(v.Challenge.Unwrap().(*Challenge).leafVertexCount)
		refund.Mul(ChallengeVertexStake, big.NewInt(leafCount+1))
		refund.Div(refund, big.NewInt(2))
		v.Challenge.Unwrap().(*Challenge).rootAssertion.Unwrap().chain.AddToBalance(tx, v.Validator, refund)
		v.Challenge.Unwrap().(*Challenge).WinnerAssertion = v.winnerIfConfirmed
	}
}

func (v *ChallengeVertex) GetPrev() util.Option[ChallengeVertexInterface] {
	return v.Prev
}
func (v *ChallengeVertex) GetStatus() AssertionState {
	return v.Status
}
func (v *ChallengeVertex) GetSubChallenge() util.Option[ChallengeInterface] {
	return v.SubChallenge
}
func (v *ChallengeVertex) GetPsTimer() *util.CountUpTimer {
	return v.PsTimer
}
func (v *ChallengeVertex) GetCommitment() util.HistoryCommitment {
	return v.Commitment
}
func (v *ChallengeVertex) GetValidator() common.Address {
	return v.Validator
}
func (v *ChallengeVertex) GetSequenceNum() VertexSequenceNumber {
	return v.SequenceNum
}
func (v *ChallengeVertex) GetChallenge() util.Option[ChallengeInterface] {
	return v.Challenge
}
func (v *ChallengeVertex) GetPresumptiveSuccessor() util.Option[ChallengeVertexInterface] {
	return v.PresumptiveSuccessor
}

func historyProvidesLastLeafProof(history util.HistoryCommitment) bool {
	return history.LastLeaf != (common.Hash{}) &&
		len(history.LastLeafProof) != 0 &&
		!history.LastLeafPrefix.IsNone() &&
		!history.Normalized().IsNone()
}

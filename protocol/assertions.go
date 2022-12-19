package protocol

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	Gwei              = big.NewInt(1000000000)
	AssertionStakeWei = Gwei

	ErrWrongChain             = errors.New("wrong chain")
	ErrInvalidOp              = errors.New("invalid operation")
	ErrChallengeAlreadyExists = errors.New("challenge already exists on leaf")
	ErrCannotChallengeOwnLeaf = errors.New("cannot challenge own leaf")
	ErrInvalidHeight          = errors.New("invalid block height")
	ErrVertexAlreadyExists    = errors.New("vertex already exists")
	ErrWrongState             = errors.New("vertex state does not allow this operation")
	ErrWrongPredecessorState  = errors.New("predecessor state does not allow this operation")
	ErrNotYet                 = errors.New("deadline has not yet passed")
	ErrNoWinnerYet            = errors.New("challenges does not yet have a winnerAssertion")
	ErrPastDeadline           = errors.New("deadline has passed")
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrNotImplemented         = errors.New("not yet implemented")
)

// CommitHash uses the hash of an assertion's state commitment
// as a type used throughout the protocol.
type CommitHash common.Hash

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

// AssertionManager allows the creation of new leaves for a Staker with a State Commitment
// and a previous assertion.
type AssertionManager interface {
	Inbox() *Inbox
	NumAssertions(tx *ActiveTx) uint64
	AssertionBySequenceNum(tx *ActiveTx, seqNum AssertionSequenceNumber) (*Assertion, error)
	ChallengeByCommitHash(tx *ActiveTx, commitHash CommitHash) (*Challenge, error)
	ChallengeVertexBySequenceNum(tx *ActiveTx, commitHash CommitHash, seqNum VertexSequenceNumber) (*ChallengeVertex, error)
	ChallengePeriodLength(tx *ActiveTx) time.Duration
	LatestConfirmed(*ActiveTx) *Assertion
	CreateLeaf(tx *ActiveTx, prev *Assertion, commitment StateCommitment, staker common.Address) (*Assertion, error)
}

type AssertionChain struct {
	mutex                               sync.RWMutex
	timeReference                       util.TimeReference
	challengePeriod                     time.Duration
	latestConfirmed                     AssertionSequenceNumber
	assertions                          []*Assertion
	hasSeenAssertions                   map[common.Hash]bool
	challengeVerticesByCommitHashSeqNum map[CommitHash]map[VertexSequenceNumber]*ChallengeVertex
	challengesByCommitHash              map[CommitHash]*Challenge
	balances                            *util.MapWithDefault[common.Address, *big.Int]
	feed                                *EventFeed[AssertionChainEvent]
	challengesFeed                      *EventFeed[ChallengeEvent]
	inbox                               *Inbox
}

const (
	deadTxStatus = iota
	readOnlyTxStatus
	readWriteTxStatus
)

// ActiveTx is a transaction that is currently being processed.
type ActiveTx struct {
	txStatus int
}

// verifyRead is a helper function to verify that the transaction is read-only.
func (tx *ActiveTx) verifyRead() {
	if tx.txStatus == deadTxStatus {
		panic("tried to read chain after call ended")
	}
}

// verifyReadWrite is a helper function to verify that the transaction is read-write.
func (tx *ActiveTx) verifyReadWrite() {
	if tx.txStatus != readWriteTxStatus {
		panic("tried to modify chain in read-only call")
	}
}

// Tx enables a mutating call to the on-chain protocol.
func (chain *AssertionChain) Tx(clo func(tx *ActiveTx, p OnChainProtocol) error) error {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	err := clo(tx, chain)
	tx.txStatus = deadTxStatus
	return err
}

// Call enables a non-mutating call to the on-chain protocol.
func (chain *AssertionChain) Call(clo func(tx *ActiveTx, p OnChainProtocol) error) error {
	chain.mutex.RLock()
	defer chain.mutex.RUnlock()
	tx := &ActiveTx{txStatus: readOnlyTxStatus}
	err := clo(tx, chain)
	tx.txStatus = deadTxStatus
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
	SequenceNum             AssertionSequenceNumber
	StateCommitment         StateCommitment
	Staker                  util.Option[common.Address]
	Prev                    util.Option[*Assertion]
	chain                   *AssertionChain
	status                  AssertionState
	isFirstChild            bool
	firstChildCreationTime  util.Option[time.Time]
	secondChildCreationTime util.Option[time.Time]
	challenge               util.Option[*Challenge]
}

// StateCommitment is a type used to represent the state commitment of an assertion.
type StateCommitment struct {
	Height    uint64
	StateRoot common.Hash
}

// Hash returns the hash of the state commitment.
func (comm StateCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, comm.Height), comm.StateRoot.Bytes())
}

// NewAssertionChain creates a new AssertionChain.
func NewAssertionChain(ctx context.Context, timeRef util.TimeReference, challengePeriod time.Duration) *AssertionChain {
	genesis := &Assertion{
		chain:       nil,
		status:      ConfirmedAssertionState,
		SequenceNum: 0,
		StateCommitment: StateCommitment{
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
	chain := &AssertionChain{
		mutex:                               sync.RWMutex{},
		timeReference:                       timeRef,
		challengePeriod:                     challengePeriod,
		challengesByCommitHash:              make(map[CommitHash]*Challenge),
		challengeVerticesByCommitHashSeqNum: make(map[CommitHash]map[VertexSequenceNumber]*ChallengeVertex),
		latestConfirmed:                     0,
		assertions:                          []*Assertion{genesis},
		balances:                            util.NewMapWithDefaultAdvanced[common.Address, *big.Int](common.Big0, func(x *big.Int) bool { return x.Sign() == 0 }),
		feed:                                NewEventFeed[AssertionChainEvent](ctx),
		challengesFeed:                      NewEventFeed[ChallengeEvent](ctx),
		inbox:                               NewInbox(ctx),
		hasSeenAssertions:                   make(map[common.Hash]bool),
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
		return ErrInsufficientBalance
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

// ChallengeVertexBySequenceNum returns the challenge vertex with the given sequence number.
func (chain *AssertionChain) ChallengeVertexBySequenceNum(
	tx *ActiveTx, commitHash CommitHash, seqNum VertexSequenceNumber,
) (*ChallengeVertex, error) {
	tx.verifyRead()
	vertices, ok := chain.challengeVerticesByCommitHashSeqNum[commitHash]
	if !ok {
		return nil, fmt.Errorf("challenge vertices not found for assertion with state commit hash %#x", commitHash)
	}
	if seqNum >= VertexSequenceNumber(len(vertices)) {
		return nil, fmt.Errorf("challenge vertex sequence out of range %d >= %d", seqNum, len(vertices))
	}
	vertex, ok := vertices[seqNum]
	if !ok {
		return nil, fmt.Errorf("challenge vertex with sequence number not found %d", seqNum)
	}
	return vertex, nil
}

// ChallengeByCommitHash returns the challenge with the given commit hash.
func (chain *AssertionChain) ChallengeByCommitHash(tx *ActiveTx, commitHash CommitHash) (*Challenge, error) {
	tx.verifyRead()
	chal, ok := chain.challengesByCommitHash[commitHash]
	if !ok {
		return nil, fmt.Errorf("challenge not found for assertion with state commit hash %#x", commitHash)
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
func (chain *AssertionChain) CreateLeaf(tx *ActiveTx, prev *Assertion, commitment StateCommitment, staker common.Address) (*Assertion, error) {
	tx.verifyReadWrite()
	if prev.chain != chain {
		return nil, ErrWrongChain
	}
	if prev.StateCommitment.Height >= commitment.Height {
		return nil, ErrInvalidOp
	}
	seenKey := crypto.Keccak256Hash(binary.BigEndian.AppendUint64(commitment.Hash().Bytes(), uint64(prev.SequenceNum)))
	if chain.hasSeenAssertions[seenKey] {
		return nil, ErrVertexAlreadyExists
	}

	if err := prev.Staker.IfLet(
		func(oldStaker common.Address) error {
			if staker != oldStaker {
				if err := chain.DeductFromBalance(tx, staker, AssertionStakeWei); err != nil {
					return err
				}
				chain.AddToBalance(tx, oldStaker, AssertionStakeWei)
				prev.Staker = util.None[common.Address]()
			}
			return nil
		},
		func() error {
			if err := chain.DeductFromBalance(tx, staker, AssertionStakeWei); err != nil {
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
		Prev:                    util.Some[*Assertion](prev),
		isFirstChild:            prev.firstChildCreationTime.IsNone(),
		firstChildCreationTime:  util.None[time.Time](),
		secondChildCreationTime: util.None[time.Time](),
		challenge:               util.None[*Challenge](),
		Staker:                  util.Some[common.Address](staker),
	}
	if prev.firstChildCreationTime.IsNone() {
		prev.firstChildCreationTime = util.Some[time.Time](chain.timeReference.Get())
	} else if prev.secondChildCreationTime.IsNone() {
		prev.secondChildCreationTime = util.Some[time.Time](chain.timeReference.Get())
	}
	chain.assertions = append(chain.assertions, leaf)
	chain.hasSeenAssertions[seenKey] = true
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
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	if a.Prev.Unwrap().status != RejectedAssertionState {
		return ErrWrongPredecessorState
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
		return ErrWrongState
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
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	prev := a.Prev.Unwrap()
	if prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	if !prev.secondChildCreationTime.IsNone() {
		return ErrInvalidOp
	}
	if !a.chain.timeReference.Get().After(prev.firstChildCreationTime.Unwrap().Add(a.chain.challengePeriod)) {
		return ErrNotYet
	}
	a.status = ConfirmedAssertionState
	a.chain.latestConfirmed = a.SequenceNum
	a.chain.feed.Append(&ConfirmEvent{
		SeqNum: a.SequenceNum,
	})
	if !a.Staker.IsNone() {
		a.chain.AddToBalance(tx, a.Staker.Unwrap(), AssertionStakeWei)
		a.Staker = util.None[common.Address]()
	}
	return nil
}

// ConfirmForWin confirms that the assertion is the winnerAssertion of the challenge and moves the assertion to `ConfirmedAssertionState` state.
func (a *Assertion) ConfirmForWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalidOp
	}
	prev := a.Prev.Unwrap()
	if prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
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

// Challenge created by an assertion.
type Challenge struct {
	rootAssertion         *Assertion
	winnerAssertion       *Assertion
	rootVertex            *ChallengeVertex
	latestConfirmedVertex *ChallengeVertex
	creationTime          time.Time
	includedHistories     map[common.Hash]bool
	nextSequenceNum       VertexSequenceNumber
}

// CreateChallenge creates a challenge for the assertion and moves the assertion to `ChallengedAssertionState` state.
func (a *Assertion) CreateChallenge(tx *ActiveTx, ctx context.Context, challenger common.Address) (*Challenge, error) {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState && a.chain.LatestConfirmed(tx) != a {
		return nil, ErrWrongState
	}
	if !a.challenge.IsNone() {
		return nil, ErrChallengeAlreadyExists
	}
	if a.secondChildCreationTime.IsNone() {
		return nil, ErrInvalidOp
	}
	currSeqNumber := VertexSequenceNumber(0)
	rootVertex := &ChallengeVertex{
		challenge:   nil,
		SequenceNum: currSeqNumber,
		isLeaf:      false,
		status:      ConfirmedAssertionState,
		Commitment: util.HistoryCommitment{
			Height: 0,
			Merkle: common.Hash{},
		},
		Prev:                 nil,
		presumptiveSuccessor: nil,
		psTimer:              util.NewCountUpTimer(a.chain.timeReference),
		subChallenge:         nil,
	}
	chal := &Challenge{
		rootAssertion:         a,
		winnerAssertion:       nil,
		rootVertex:            rootVertex,
		latestConfirmedVertex: rootVertex,
		creationTime:          a.chain.timeReference.Get(),
		includedHistories:     make(map[common.Hash]bool),
		nextSequenceNum:       currSeqNumber + 1,
	}
	rootVertex.challenge = chal
	chal.includedHistories[rootVertex.Commitment.Hash()] = true
	a.challenge = util.Some[*Challenge](chal)
	parentStaker := common.Address{}
	if !a.Staker.IsNone() {
		parentStaker = a.Staker.Unwrap()
	}
	a.chain.feed.Append(&StartChallengeEvent{
		ParentSeqNum:          a.SequenceNum,
		ParentStateCommitment: a.StateCommitment,
		ParentStaker:          parentStaker,
		Validator:             challenger,
	})

	challengeID := CommitHash(a.StateCommitment.Hash())
	a.chain.challengesByCommitHash[challengeID] = chal
	a.chain.challengeVerticesByCommitHashSeqNum[challengeID] = map[VertexSequenceNumber]*ChallengeVertex{currSeqNumber: rootVertex}

	return chal, nil
}

// ParentStateCommitment returns the state commitment of the parent assertion.
func (c *Challenge) ParentStateCommitment() StateCommitment {
	return c.rootAssertion.StateCommitment
}

// AssertionSeqNumber returns the sequence number of the assertion that created the challenge.
func (c *Challenge) AssertionSeqNumber() AssertionSequenceNumber {
	return c.rootAssertion.SequenceNum
}

// AddLeaf adds a new leaf to the challenge.
func (c *Challenge) AddLeaf(tx *ActiveTx, assertion *Assertion, history util.HistoryCommitment, challenger common.Address) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if assertion.Prev.IsNone() {
		return nil, ErrInvalidOp
	}
	prev := assertion.Prev.Unwrap()
	if prev != c.rootAssertion {
		return nil, ErrInvalidOp
	}
	if c.Completed(tx) {
		return nil, ErrWrongState
	}
	chain := assertion.chain
	if !c.rootVertex.eligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if c.includedHistories[history.Hash()] {
		return nil, ErrVertexAlreadyExists
	}

	timer := util.NewCountUpTimer(chain.timeReference)
	if assertion.isFirstChild {
		delta := prev.secondChildCreationTime.Unwrap().Sub(prev.firstChildCreationTime.Unwrap())
		timer.Set(delta)
	}
	leaf := &ChallengeVertex{
		challenge:            c,
		SequenceNum:          c.nextSequenceNum,
		Challenger:           challenger,
		isLeaf:               true,
		status:               PendingAssertionState,
		Commitment:           history,
		Prev:                 c.rootVertex,
		presumptiveSuccessor: nil,
		psTimer:              timer,
		subChallenge:         nil,
		winnerIfConfirmed:    assertion,
	}
	c.nextSequenceNum++
	c.rootVertex.maybeNewPresumptiveSuccessor(leaf)
	c.rootAssertion.chain.challengesFeed.Append(&ChallengeLeafEvent{
		ParentSeqNum:      leaf.Prev.SequenceNum,
		SequenceNum:       leaf.SequenceNum,
		WinnerIfConfirmed: assertion.SequenceNum,
		History:           history,
		BecomesPS:         leaf.Prev.presumptiveSuccessor == leaf,
		Validator:         challenger,
	})
	c.includedHistories[history.Hash()] = true
	h := CommitHash(c.rootAssertion.StateCommitment.Hash())
	c.rootAssertion.chain.challengesByCommitHash[h] = c
	c.rootAssertion.chain.challengeVerticesByCommitHashSeqNum[h][leaf.SequenceNum] = leaf
	return leaf, nil
}

// Completed returns true if the challenge is completed.
func (c *Challenge) Completed(tx *ActiveTx) bool {
	tx.verifyRead()
	return c.winnerAssertion != nil
}

// Winner returns the winning assertion if the challenge is completed.
func (c *Challenge) Winner(tx *ActiveTx) (*Assertion, error) {
	tx.verifyRead()
	if c.winnerAssertion == nil {
		return nil, ErrNoWinnerYet
	}
	return c.winnerAssertion, nil
}

type ChallengeVertex struct {
	Commitment           util.HistoryCommitment
	challenge            *Challenge
	SequenceNum          VertexSequenceNumber // unique within the challenge
	Challenger           common.Address
	isLeaf               bool
	status               AssertionState
	Prev                 *ChallengeVertex
	presumptiveSuccessor *ChallengeVertex
	psTimer              *util.CountUpTimer
	subChallenge         *SubChallenge
	winnerIfConfirmed    *Assertion
}

// eligibleForNewSuccessor returns true if the vertex is eligible to have a new successor.
func (v *ChallengeVertex) eligibleForNewSuccessor() bool {
	return v.presumptiveSuccessor == nil || v.presumptiveSuccessor.psTimer.Get() <= v.challenge.rootAssertion.chain.challengePeriod
}

// maybeNewPresumptiveSuccessor updates the presumptive successor if the given vertex is eligible.
func (v *ChallengeVertex) maybeNewPresumptiveSuccessor(succ *ChallengeVertex) {
	if v.presumptiveSuccessor != nil && succ.Commitment.Height < v.presumptiveSuccessor.Commitment.Height {
		v.presumptiveSuccessor.psTimer.Stop()
		v.presumptiveSuccessor = nil
	}
	if v.presumptiveSuccessor == nil {
		v.presumptiveSuccessor = succ
		succ.psTimer.Start()
	}
}

// IsPresumptiveSuccessor returns true if the vertex is the presumptive successor of its parent.
func (v *ChallengeVertex) IsPresumptiveSuccessor() bool {
	return v.Prev == nil || v.Prev.presumptiveSuccessor == v
}

// requiredBisectionHeight returns the height of the history commitment that must be bisectioned to prove the vertex.
func (v *ChallengeVertex) requiredBisectionHeight() (uint64, error) {
	return util.BisectionPoint(v.Prev.Commitment.Height, v.Commitment.Height)
}

// Bisect returns the bisection proof for the vertex.
func (v *ChallengeVertex) Bisect(tx *ActiveTx, history util.HistoryCommitment, proof []common.Hash, challenger common.Address) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if v.IsPresumptiveSuccessor() {
		return nil, ErrWrongState
	}
	if !v.Prev.eligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if v.challenge.includedHistories[history.Hash()] {
		return nil, ErrVertexAlreadyExists
	}
	bisectionHeight, err := v.requiredBisectionHeight()
	if err != nil {
		return nil, err
	}
	if bisectionHeight != history.Height {
		return nil, ErrInvalidHeight
	}
	if err := util.VerifyPrefixProof(history, v.Commitment, proof); err != nil {
		return nil, err
	}

	v.psTimer.Stop()
	newVertex := &ChallengeVertex{
		challenge:            v.challenge,
		SequenceNum:          v.challenge.nextSequenceNum,
		Challenger:           challenger,
		isLeaf:               false,
		Commitment:           history,
		Prev:                 v.Prev,
		presumptiveSuccessor: nil,
		psTimer:              v.psTimer.Clone(),
	}
	newVertex.challenge.nextSequenceNum++
	newVertex.maybeNewPresumptiveSuccessor(v)
	newVertex.Prev.maybeNewPresumptiveSuccessor(newVertex)
	newVertex.challenge.includedHistories[history.Hash()] = true

	v.Prev = newVertex

	newVertex.challenge.rootAssertion.chain.challengesFeed.Append(&ChallengeBisectEvent{
		FromSequenceNum: v.SequenceNum,
		SequenceNum:     newVertex.SequenceNum,
		History:         newVertex.Commitment,
		BecomesPS:       newVertex.Prev.presumptiveSuccessor == newVertex,
		Validator:       challenger,
	})
	commitHash := CommitHash(newVertex.challenge.rootAssertion.StateCommitment.Hash())
	newVertex.challenge.rootAssertion.chain.challengeVerticesByCommitHashSeqNum[commitHash][newVertex.SequenceNum] = newVertex

	return newVertex, nil
}

// Merge merges the vertex with its presumptive successor.
func (v *ChallengeVertex) Merge(tx *ActiveTx, mergingTo *ChallengeVertex, proof []common.Hash, challenger common.Address) error {
	tx.verifyReadWrite()
	if !mergingTo.eligibleForNewSuccessor() {
		return ErrPastDeadline
	}
	// The vertex we are merging to should be the mandatory bisection point
	// of the current vertex's height and its parent's height.
	bisectionPoint, err := util.BisectionPoint(v.Prev.Commitment.Height, v.Commitment.Height)
	if err != nil {
		return err
	}
	if mergingTo.Commitment.Height != bisectionPoint {
		return ErrInvalidHeight
	}
	if err := util.VerifyPrefixProof(mergingTo.Commitment, v.Commitment, proof); err != nil {
		return err
	}

	v.Prev = mergingTo
	mergingTo.psTimer.Add(v.psTimer.Get())
	mergingTo.maybeNewPresumptiveSuccessor(v)
	v.challenge.rootAssertion.chain.challengesFeed.Append(&ChallengeMergeEvent{
		DeeperSequenceNum:    v.SequenceNum,
		ShallowerSequenceNum: mergingTo.SequenceNum,
		BecomesPS:            mergingTo.presumptiveSuccessor == v,
		History:              mergingTo.Commitment,
		Validator:            challenger,
	})
	return nil
}

// ConfirmForSubChallengeWin confirms the vertex as the winner of a sub-challenge.
func (v *ChallengeVertex) ConfirmForSubChallengeWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.status != PendingAssertionState {
		return ErrWrongState
	}
	if v.Prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	subChal := v.Prev.subChallenge
	if subChal == nil || subChal.winner != v {
		return ErrInvalidOp
	}
	v._confirm()
	return nil
}

// ConfirmForPsTimer confirms the vertex as the winner of a psTimer.
func (v *ChallengeVertex) ConfirmForPsTimer(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.status != PendingAssertionState {
		return ErrWrongState
	}
	if v.Prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	if v.psTimer.Get() <= v.challenge.rootAssertion.chain.challengePeriod {
		return ErrNotYet
	}
	v._confirm()
	return nil
}

// ConfirmForChallengeDeadline confirms the vertex as the winner of a challenge deadline.
func (v *ChallengeVertex) ConfirmForChallengeDeadline(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.status != PendingAssertionState {
		return ErrWrongState
	}
	if v.Prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	chain := v.challenge.rootAssertion.chain
	chalPeriod := chain.challengePeriod
	if !chain.timeReference.Get().After(v.challenge.creationTime.Add(2 * chalPeriod)) {
		return ErrNotYet
	}
	v._confirm()
	return nil
}

func (v *ChallengeVertex) _confirm() {
	v.status = ConfirmedAssertionState
	if v.isLeaf {
		v.challenge.winnerAssertion = v.winnerIfConfirmed
	}
}

// CreateSubChallenge creates a sub-challenge for the vertex.
func (v *ChallengeVertex) CreateSubChallenge(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.subChallenge != nil {
		return ErrVertexAlreadyExists
	}
	if v.status == ConfirmedAssertionState {
		return ErrWrongState
	}
	v.subChallenge = &SubChallenge{
		parent: v,
		winner: nil,
	}
	return nil
}

type SubChallenge struct {
	parent *ChallengeVertex
	winner *ChallengeVertex
}

// SetWinner sets the winner of the sub-challenge.
func (sc *SubChallenge) SetWinner(tx *ActiveTx, winner *ChallengeVertex) error {
	tx.verifyReadWrite()
	if sc.winner != nil {
		return ErrInvalidOp
	}
	if winner.Prev != sc.parent {
		return ErrInvalidOp
	}
	sc.winner = winner
	return nil
}

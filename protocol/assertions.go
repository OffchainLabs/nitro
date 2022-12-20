package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
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

// AssertionManager allows the creation of new leaves for a Staker with a State Commitment
// and a previous assertion.
type AssertionManager interface {
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
	CreateLeaf(tx *ActiveTx, prev *Assertion, commitment StateCommitment, staker common.Address) (*Assertion, error)
}

type AssertionChain struct {
	mutex                         sync.RWMutex
	timeReference                 util.TimeReference
	challengePeriod               time.Duration
	latestConfirmed               AssertionSequenceNumber
	assertions                    []*Assertion
	hasSeenAssertions             map[common.Hash]bool
	challengeVerticesByCommitHash map[ChallengeCommitHash]map[VertexCommitHash]*ChallengeVertex
	challengesByCommitHash        map[ChallengeCommitHash]*Challenge
	balances                      *util.MapWithDefault[common.Address, *big.Int]
	feed                          *EventFeed[AssertionChainEvent]
	challengesFeed                *EventFeed[ChallengeEvent]
	inbox                         *Inbox
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
		hasSeenAssertions:             make(map[common.Hash]bool),
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
		vParentHash := VertexCommitHash(v.Prev.Unwrap().Commitment.Hash())
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
	return vertex.Commitment.Height == vertex.Prev.Unwrap().Commitment.Height+1
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

// Challenge created by an assertion.
type Challenge struct {
	rootAssertion         util.Option[*Assertion]
	winnerAssertion       util.Option[*Assertion]
	rootVertex            util.Option[*ChallengeVertex]
	latestConfirmedVertex util.Option[*ChallengeVertex]
	creationTime          time.Time
	includedHistories     map[common.Hash]bool
	nextSequenceNum       VertexSequenceNumber
}

// CreateChallenge creates a challenge for the assertion and moves the assertion to `ChallengedAssertionState` state.
func (a *Assertion) CreateChallenge(tx *ActiveTx, ctx context.Context, validator common.Address) (*Challenge, error) {
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
		challenge:   util.None[*Challenge](),
		SequenceNum: currSeqNumber,
		isLeaf:      false,
		status:      ConfirmedAssertionState,
		Commitment: util.HistoryCommitment{
			Height: 0,
			Merkle: common.Hash{},
		},
		Prev:                 util.None[*ChallengeVertex](),
		PresumptiveSuccessor: util.None[*ChallengeVertex](),
		psTimer:              util.NewCountUpTimer(a.chain.timeReference),
		subChallenge:         util.None[*SubChallenge](),
	}

	chal := &Challenge{
		rootAssertion:         util.Some[*Assertion](a),
		winnerAssertion:       util.None[*Assertion](),
		rootVertex:            util.Some[*ChallengeVertex](rootVertex),
		latestConfirmedVertex: util.Some[*ChallengeVertex](rootVertex),
		creationTime:          a.chain.timeReference.Get(),
		includedHistories:     make(map[common.Hash]bool),
		nextSequenceNum:       currSeqNumber + 1,
	}
	rootVertex.challenge = util.Some[*Challenge](chal)
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
		Validator:             validator,
	})

	challengeID := ChallengeCommitHash(a.StateCommitment.Hash())
	a.chain.challengesByCommitHash[challengeID] = chal
	a.chain.challengeVerticesByCommitHash[challengeID] = map[VertexCommitHash]*ChallengeVertex{VertexCommitHash(rootVertex.Commitment.Merkle): rootVertex}

	return chal, nil
}

// ParentStateCommitment returns the state commitment of the parent assertion.
func (c *Challenge) ParentStateCommitment() StateCommitment {
	return c.rootAssertion.Unwrap().StateCommitment
}

// AssertionSeqNumber returns the sequence number of the assertion that created the challenge.
func (c *Challenge) AssertionSeqNumber() AssertionSequenceNumber {
	return c.rootAssertion.Unwrap().SequenceNum
}

// AddLeaf adds a new leaf to the challenge.
func (c *Challenge) AddLeaf(tx *ActiveTx, assertion *Assertion, history util.HistoryCommitment, validator common.Address) (*ChallengeVertex, error) {
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
	chain := assertion.chain
	if !c.rootVertex.Unwrap().eligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if c.includedHistories[history.Hash()] {
		return nil, errors.Wrapf(ErrVertexAlreadyExists, fmt.Sprintf("Hash: %s", history.Hash().String()))
	}

	timer := util.NewCountUpTimer(chain.timeReference)
	if assertion.isFirstChild {
		delta := prev.secondChildCreationTime.Unwrap().Sub(prev.firstChildCreationTime.Unwrap())
		timer.Set(delta)
	}
	leaf := &ChallengeVertex{
		challenge:            util.Some[*Challenge](c),
		SequenceNum:          c.nextSequenceNum,
		Validator:            validator,
		isLeaf:               true,
		status:               PendingAssertionState,
		Commitment:           history,
		Prev:                 c.rootVertex,
		PresumptiveSuccessor: util.None[*ChallengeVertex](),
		psTimer:              timer,
		subChallenge:         util.None[*SubChallenge](),
		winnerIfConfirmed:    util.Some[*Assertion](assertion),
	}
	c.nextSequenceNum++
	c.rootVertex.Unwrap().maybeNewPresumptiveSuccessor(leaf)
	c.rootAssertion.Unwrap().chain.challengesFeed.Append(&ChallengeLeafEvent{
		ParentSeqNum:      leaf.Prev.Unwrap().SequenceNum,
		SequenceNum:       leaf.SequenceNum,
		WinnerIfConfirmed: assertion.SequenceNum,
		History:           history,
		BecomesPS:         leaf.Prev.Unwrap().PresumptiveSuccessor.Unwrap() == leaf,
		Validator:         validator,
	})
	c.includedHistories[history.Hash()] = true
	h := ChallengeCommitHash(c.rootAssertion.Unwrap().StateCommitment.Hash())
	c.rootAssertion.Unwrap().chain.challengesByCommitHash[h] = c
	c.rootAssertion.Unwrap().chain.challengeVerticesByCommitHash[h][VertexCommitHash(leaf.Commitment.Merkle)] = leaf
	return leaf, nil
}

// Completed returns true if the challenge is completed.
func (c *Challenge) Completed(tx *ActiveTx) bool {
	tx.verifyRead()
	return !c.winnerAssertion.IsNone()
}

// Winner returns the winning assertion if the challenge is completed.
func (c *Challenge) Winner(tx *ActiveTx) (*Assertion, error) {
	tx.verifyRead()
	if c.winnerAssertion.IsNone() {
		return nil, ErrNoWinnerYet
	}
	return c.winnerAssertion.Unwrap(), nil
}

type ChallengeVertex struct {
	Commitment           util.HistoryCommitment
	challenge            util.Option[*Challenge]
	SequenceNum          VertexSequenceNumber // unique within the challenge
	Validator            common.Address
	isLeaf               bool
	status               AssertionState
	Prev                 util.Option[*ChallengeVertex]
	PresumptiveSuccessor util.Option[*ChallengeVertex]
	psTimer              *util.CountUpTimer
	subChallenge         util.Option[*SubChallenge]
	winnerIfConfirmed    util.Option[*Assertion]
}

// eligibleForNewSuccessor returns true if the vertex is eligible to have a new successor.
func (v *ChallengeVertex) eligibleForNewSuccessor() bool {
	return v.PresumptiveSuccessor.IsNone() ||
		v.PresumptiveSuccessor.Unwrap().psTimer.Get() <= v.challenge.Unwrap().rootAssertion.Unwrap().chain.challengePeriod
}

// maybeNewPresumptiveSuccessor updates the presumptive successor if the given vertex is eligible.
func (v *ChallengeVertex) maybeNewPresumptiveSuccessor(succ *ChallengeVertex) {
	if !v.PresumptiveSuccessor.IsNone() &&
		succ.Commitment.Height < v.PresumptiveSuccessor.Unwrap().Commitment.Height {
		v.PresumptiveSuccessor.Unwrap().psTimer.Stop()
		v.PresumptiveSuccessor = util.None[*ChallengeVertex]()
	}

	if v.PresumptiveSuccessor.IsNone() {
		v.PresumptiveSuccessor = util.Some(succ)
		succ.psTimer.Start()
	}
}

// IsPresumptiveSuccessor returns true if the vertex is the presumptive successor of its parent.
func (v *ChallengeVertex) IsPresumptiveSuccessor() bool {
	return v.Prev.IsNone() || v.Prev.Unwrap().PresumptiveSuccessor.Unwrap() == v
}

// requiredBisectionHeight returns the height of the history commitment that must be bisectioned to prove the vertex.
func (v *ChallengeVertex) requiredBisectionHeight() (uint64, error) {
	return util.BisectionPoint(v.Prev.Unwrap().Commitment.Height, v.Commitment.Height)
}

// Bisect returns the bisection proof for the vertex.
func (v *ChallengeVertex) Bisect(tx *ActiveTx, history util.HistoryCommitment, proof []common.Hash, validator common.Address) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if v.IsPresumptiveSuccessor() {
		return nil, ErrWrongState
	}
	if !v.Prev.Unwrap().eligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if v.challenge.Unwrap().includedHistories[history.Hash()] {
		return nil, errors.Wrapf(ErrVertexAlreadyExists, fmt.Sprintf("Hash: %s", history.Hash().String()))
	}
	bisectionHeight, err := v.requiredBisectionHeight()
	if err != nil {
		return nil, err
	}
	if bisectionHeight != history.Height {
		return nil, errors.Wrapf(ErrInvalidHeight, fmt.Sprintf("%d != %d", bisectionHeight, history))
	}
	if err := util.VerifyPrefixProof(history, v.Commitment, proof); err != nil {
		return nil, err
	}

	v.psTimer.Stop()
	newVertex := &ChallengeVertex{
		challenge:            v.challenge,
		SequenceNum:          v.challenge.Unwrap().nextSequenceNum,
		Validator:            validator,
		isLeaf:               false,
		Commitment:           history,
		Prev:                 v.Prev,
		PresumptiveSuccessor: util.None[*ChallengeVertex](),
		psTimer:              v.psTimer.Clone(),
	}
	newVertex.challenge.Unwrap().nextSequenceNum++
	newVertex.maybeNewPresumptiveSuccessor(v)
	newVertex.Prev.Unwrap().maybeNewPresumptiveSuccessor(newVertex)
	newVertex.challenge.Unwrap().includedHistories[history.Hash()] = true

	v.Prev = util.Some[*ChallengeVertex](newVertex)

	newVertex.challenge.Unwrap().rootAssertion.Unwrap().chain.challengesFeed.Append(&ChallengeBisectEvent{
		FromSequenceNum: v.SequenceNum,
		SequenceNum:     newVertex.SequenceNum,
		History:         newVertex.Commitment,
		BecomesPS:       newVertex.Prev.Unwrap().PresumptiveSuccessor.Unwrap() == newVertex,
		Validator:       validator,
	})
	commitHash := ChallengeCommitHash(newVertex.challenge.Unwrap().rootAssertion.Unwrap().StateCommitment.Hash())
	newVertex.challenge.Unwrap().rootAssertion.Unwrap().chain.challengeVerticesByCommitHash[commitHash][VertexCommitHash(newVertex.Commitment.Merkle)] = newVertex

	return newVertex, nil
}

// Merge merges the vertex with its presumptive successor.
func (v *ChallengeVertex) Merge(tx *ActiveTx, mergingTo *ChallengeVertex, proof []common.Hash, validator common.Address) error {
	tx.verifyReadWrite()
	if !mergingTo.eligibleForNewSuccessor() {
		return ErrPastDeadline
	}
	// The vertex we are merging to should be the mandatory bisection point
	// of the current vertex's height and its parent's height.
	bisectionPoint, err := util.BisectionPoint(v.Prev.Unwrap().Commitment.Height, v.Commitment.Height)
	if err != nil {
		return err
	}
	if mergingTo.Commitment.Height != bisectionPoint {
		return errors.Wrapf(ErrInvalidHeight, "%d != %d", mergingTo.Commitment.Height, bisectionPoint)
	}
	if err := util.VerifyPrefixProof(mergingTo.Commitment, v.Commitment, proof); err != nil {
		return err
	}

	v.Prev = util.Some(mergingTo)
	mergingTo.psTimer.Add(v.psTimer.Get())
	mergingTo.maybeNewPresumptiveSuccessor(v)
	v.challenge.Unwrap().rootAssertion.Unwrap().chain.challengesFeed.Append(&ChallengeMergeEvent{
		DeeperSequenceNum:    v.SequenceNum,
		ShallowerSequenceNum: mergingTo.SequenceNum,
		BecomesPS:            mergingTo.PresumptiveSuccessor.Unwrap() == v,
		History:              mergingTo.Commitment,
		Validator:            validator,
	})
	return nil
}

// ConfirmForSubChallengeWin confirms the vertex as the winner of a sub-challenge.
func (v *ChallengeVertex) ConfirmForSubChallengeWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.status))
	}
	if v.Prev.Unwrap().status != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", v.Prev.Unwrap().status))
	}
	subChal := v.Prev.Unwrap().subChallenge
	if subChal.IsNone() || subChal.Unwrap().winner != v {
		return ErrInvalidOp
	}
	v._confirm()
	return nil
}

// ConfirmForPsTimer confirms the vertex as the winner of a psTimer.
func (v *ChallengeVertex) ConfirmForPsTimer(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.status))
	}
	if v.Prev.Unwrap().status != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", v.Prev.Unwrap().status))
	}
	if v.psTimer.Get() <= v.challenge.Unwrap().rootAssertion.Unwrap().chain.challengePeriod {
		return errors.Wrapf(
			ErrNotYet,
			fmt.Sprintf(
				"%d <= %d",
				v.psTimer.Get(),
				v.challenge.Unwrap().rootAssertion.Unwrap().chain.challengePeriod),
		)
	}
	v._confirm()
	return nil
}

// ConfirmForChallengeDeadline confirms the vertex as the winner of a challenge deadline.
func (v *ChallengeVertex) ConfirmForChallengeDeadline(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if v.status != PendingAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.status))
	}
	if v.Prev.Unwrap().status != ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongPredecessorState, fmt.Sprintf("State: %d", v.Prev.Unwrap().status))
	}
	chain := v.challenge.Unwrap().rootAssertion.Unwrap().chain
	chalPeriod := chain.challengePeriod
	if !chain.timeReference.Get().After(v.challenge.Unwrap().creationTime.Add(2 * chalPeriod)) {
		return errors.Wrapf(
			ErrNotYet, fmt.Sprintf(
				"%d <= %d",
				chain.timeReference.Get().Unix(),
				v.challenge.Unwrap().creationTime.Add(2*chalPeriod).Unix(),
			),
		)
	}
	v._confirm()
	return nil
}

func (v *ChallengeVertex) _confirm() {
	v.status = ConfirmedAssertionState
	if v.isLeaf {
		v.challenge.Unwrap().winnerAssertion = v.winnerIfConfirmed
	}
}

// CreateSubChallenge creates a sub-challenge for the vertex.
func (v *ChallengeVertex) CreateSubChallenge(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if !v.subChallenge.IsNone() {
		return ErrVertexAlreadyExists
	}
	if v.status == ConfirmedAssertionState {
		return errors.Wrapf(ErrWrongState, fmt.Sprintf("Status: %d", v.status))
	}
	v.subChallenge = util.Some[*SubChallenge](&SubChallenge{
		parent: v,
		winner: nil,
	})
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
	if winner.Prev.Unwrap() != sc.parent {
		return ErrInvalidOp
	}
	sc.winner = winner
	return nil
}

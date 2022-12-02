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
	ErrInvalid                = errors.New("invalid operation")
	ErrChallengeAlreadyExists = errors.New("challenge already exists on leaf")
	ErrCannotChallengeOwnLeaf = errors.New("cannot challenge own leaf")
	ErrInvalidHeight          = errors.New("invalid block height")
	ErrVertexAlreadyExists    = errors.New("vertex already exists")
	ErrWrongState             = errors.New("vertex state does not allow this operation")
	ErrWrongPredecessorState  = errors.New("predecessor state does not allow this operation")
	ErrNotYet                 = errors.New("deadline has not yet passed")
	ErrNoWinnerYet            = errors.New("challenges does not yet have a winner")
	ErrPastDeadline           = errors.New("deadline has passed")
	ErrInsufficientBalance    = errors.New("insufficient balance")
	ErrNotImplemented         = errors.New("not yet implemented")
)

// ChallengeID is the challenge's parent assertion state commitment hash.
type ChallengeID common.Hash

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
	AssertionBySequenceNum(tx *ActiveTx, seqNum uint64) (*Assertion, error)
	ChallengeByID(tx *ActiveTx, challengeID ChallengeID) (*Challenge, error)
	ChallengeVertexBySequenceNum(tx *ActiveTx, challengeID ChallengeID, seqNum uint64) (*ChallengeVertex, error)
	ChallengePeriodLength(tx *ActiveTx) time.Duration
	LatestConfirmed(*ActiveTx) *Assertion
	CreateLeaf(tx *ActiveTx, prev *Assertion, commitment StateCommitment, staker common.Address) (*Assertion, error)
}

type AssertionChain struct {
	mutex                          sync.RWMutex
	timeReference                  util.TimeReference
	challengePeriod                time.Duration
	confirmedLatest                uint64
	assertions                     []*Assertion
	challengeVerticesByChallengeID map[ChallengeID][]*ChallengeVertex
	challengesByID                 map[ChallengeID]*Challenge
	dedupe                         map[common.Hash]bool
	balances                       *util.MapWithDefault[common.Address, *big.Int]
	feed                           *EventFeed[AssertionChainEvent]
	challengesFeed                 *EventFeed[ChallengeEvent]
	inbox                          *Inbox
}

const (
	deadTxStatus = iota
	readOnlyTxStatus
	readWriteTxStatus
)

type ActiveTx struct {
	txStatus int
}

func (tx *ActiveTx) verifyRead() {
	if tx.txStatus == deadTxStatus {
		panic("tried to read chain after call ended")
	}
}

func (tx *ActiveTx) verifyReadWrite() {
	if tx.txStatus != readWriteTxStatus {
		panic("tried to modify chain in read-only call")
	}
}

func (chain *AssertionChain) Tx(clo func(tx *ActiveTx, p OnChainProtocol) error) error {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	err := clo(tx, chain)
	tx.txStatus = deadTxStatus
	return err
}

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

type AssertionState int

type Assertion struct {
	SequenceNum             uint64
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

type StateCommitment struct {
	Height    uint64
	StateRoot common.Hash
}

func (comm StateCommitment) Hash() common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, comm.Height), comm.StateRoot.Bytes())
}

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
		mutex:                          sync.RWMutex{},
		timeReference:                  timeRef,
		challengePeriod:                challengePeriod,
		challengeVerticesByChallengeID: make(map[ChallengeID][]*ChallengeVertex),
		challengesByID:                 make(map[ChallengeID]*Challenge),
		confirmedLatest:                0,
		assertions:                     []*Assertion{genesis},
		dedupe:                         make(map[common.Hash]bool), // no need to insert genesis assertion here
		balances:                       util.NewMapWithDefaultAdvanced[common.Address, *big.Int](common.Big0, func(x *big.Int) bool { return x.Sign() == 0 }),
		feed:                           NewEventFeed[AssertionChainEvent](ctx),
		challengesFeed:                 NewEventFeed[ChallengeEvent](ctx),
		inbox:                          NewInbox(ctx),
	}
	genesis.chain = chain
	return chain
}

func (chain *AssertionChain) TimeReference() util.TimeReference {
	return chain.timeReference
}

func (chain *AssertionChain) Inbox() *Inbox {
	return chain.inbox
}

func (chain *AssertionChain) GetBalance(tx *ActiveTx, addr common.Address) *big.Int {
	tx.verifyRead()
	return chain.balances.Get(addr)
}

func (chain *AssertionChain) SetBalance(tx *ActiveTx, addr common.Address, balance *big.Int) {
	tx.verifyReadWrite()
	oldBalance := chain.balances.Get(addr)
	chain.balances.Set(addr, balance)
	chain.feed.Append(&SetBalanceEvent{Addr: addr, OldBalance: oldBalance, NewBalance: balance})
}

func (chain *AssertionChain) AddToBalance(tx *ActiveTx, addr common.Address, amount *big.Int) {
	tx.verifyReadWrite()
	chain.SetBalance(tx, addr, new(big.Int).Add(chain.GetBalance(tx, addr), amount))
}

func (chain *AssertionChain) DeductFromBalance(tx *ActiveTx, addr common.Address, amount *big.Int) error {
	tx.verifyReadWrite()
	balance := chain.GetBalance(tx, addr)
	if balance.Cmp(amount) < 0 {
		return ErrInsufficientBalance
	}
	chain.SetBalance(tx, addr, new(big.Int).Sub(balance, amount))
	return nil
}

func (chain *AssertionChain) ChallengePeriodLength(tx *ActiveTx) time.Duration {
	tx.verifyRead()
	return chain.challengePeriod
}

func (chain *AssertionChain) LatestConfirmed(tx *ActiveTx) *Assertion {
	tx.verifyRead()
	return chain.assertions[chain.confirmedLatest]
}

func (chain *AssertionChain) NumAssertions(tx *ActiveTx) uint64 {
	tx.verifyRead()
	return uint64(len(chain.assertions))
}

func (chain *AssertionChain) AssertionBySequenceNum(tx *ActiveTx, seqNum uint64) (*Assertion, error) {
	tx.verifyRead()
	if seqNum >= uint64(len(chain.assertions)) {
		return nil, fmt.Errorf("assertion sequence out of range %d >= %d", seqNum, len(chain.assertions))
	}
	return chain.assertions[seqNum], nil
}

func (chain *AssertionChain) ChallengeVertexBySequenceNum(tx *ActiveTx, id ChallengeID, seqNum uint64) (*ChallengeVertex, error) {
	tx.verifyRead()
	vertices, ok := chain.challengeVerticesByChallengeID[id]
	if !ok {
		return nil, fmt.Errorf("challenge vertices not found for challenge ID %#x", id)
	}
	if seqNum >= uint64(len(vertices)) {
		return nil, fmt.Errorf("challenge vertex sequence out of range %d >= %d", seqNum, len(vertices))
	}
	return vertices[seqNum], nil
}

func (chain *AssertionChain) ChallengeByID(tx *ActiveTx, id ChallengeID) (*Challenge, error) {
	tx.verifyRead()
	chal, ok := chain.challengesByID[id]
	if !ok {
		return nil, fmt.Errorf("challenge not found for challenge ID %#x", id)
	}
	return chal, nil
}

func (chain *AssertionChain) SubscribeChainEvents(ctx context.Context, ch chan<- AssertionChainEvent) {
	chain.feed.Subscribe(ctx, ch)
}

func (chain *AssertionChain) SubscribeChallengeEvents(ctx context.Context, ch chan<- ChallengeEvent) {
	chain.challengesFeed.Subscribe(ctx, ch)
}

func (chain *AssertionChain) CreateLeaf(tx *ActiveTx, prev *Assertion, commitment StateCommitment, staker common.Address) (*Assertion, error) {
	tx.verifyReadWrite()
	if prev.chain != chain {
		return nil, ErrWrongChain
	}
	if prev.StateCommitment.Height >= commitment.Height {
		return nil, ErrInvalid
	}
	dedupeCode := crypto.Keccak256Hash(binary.BigEndian.AppendUint64(commitment.Hash().Bytes(), prev.SequenceNum))
	if chain.dedupe[dedupeCode] {
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
		SequenceNum:             uint64(len(chain.assertions)),
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
	chain.dedupe[dedupeCode] = true
	chain.feed.Append(&CreateLeafEvent{
		PrevStateCommitment: prev.StateCommitment,
		PrevSeqNum:          prev.SequenceNum,
		SeqNum:              leaf.SequenceNum,
		StateCommitment:     leaf.StateCommitment,
		Staker:              staker,
	})
	return leaf, nil
}

func (a *Assertion) RejectForPrev(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalid
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

func (a *Assertion) RejectForLoss(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalid
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
		return ErrInvalid
	}
	a.status = RejectedAssertionState
	a.chain.feed.Append(&RejectEvent{
		SeqNum: a.SequenceNum,
	})
	return nil
}

func (a *Assertion) ConfirmNoRival(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalid
	}
	prev := a.Prev.Unwrap()
	if prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	if !prev.secondChildCreationTime.IsNone() {
		return ErrInvalid
	}
	if !a.chain.timeReference.Get().After(prev.firstChildCreationTime.Unwrap().Add(a.chain.challengePeriod)) {
		return ErrNotYet
	}
	a.status = ConfirmedAssertionState
	a.chain.confirmedLatest = a.SequenceNum
	a.chain.feed.Append(&ConfirmEvent{
		SeqNum: a.SequenceNum,
	})
	if !a.Staker.IsNone() {
		a.chain.AddToBalance(tx, a.Staker.Unwrap(), AssertionStakeWei)
		a.Staker = util.None[common.Address]()
	}
	return nil
}

func (a *Assertion) ConfirmForWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.Prev.IsNone() {
		return ErrInvalid
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
		return ErrInvalid
	}
	a.status = ConfirmedAssertionState
	a.chain.confirmedLatest = a.SequenceNum
	a.chain.feed.Append(&ConfirmEvent{
		SeqNum: a.SequenceNum,
	})
	return nil
}

type Challenge struct {
	parent            *Assertion
	winner            *Assertion
	root              *ChallengeVertex
	latestConfirmed   *ChallengeVertex
	creationTime      time.Time
	includedHistories map[common.Hash]bool
	nextSequenceNum   uint64
}

func (parent *Assertion) CreateChallenge(tx *ActiveTx, ctx context.Context, challenger common.Address) (*Challenge, error) {
	tx.verifyReadWrite()
	if parent.status != PendingAssertionState && parent.chain.LatestConfirmed(tx) != parent {
		return nil, ErrWrongState
	}
	if !parent.challenge.IsNone() {
		return nil, ErrChallengeAlreadyExists
	}
	if parent.secondChildCreationTime.IsNone() {
		return nil, ErrInvalid
	}
	if !parent.Staker.IsNone() {
		if parent.Staker.Unwrap() == challenger {
			return nil, ErrCannotChallengeOwnLeaf
		}
	}
	root := &ChallengeVertex{
		challenge:   nil,
		SequenceNum: 0,
		isLeaf:      false,
		status:      ConfirmedAssertionState,
		Commitment: util.HistoryCommitment{
			Height: 0,
			Merkle: common.Hash{},
		},
		Prev:                 nil,
		presumptiveSuccessor: nil,
		psTimer:              util.NewCountUpTimer(parent.chain.timeReference),
		subChallenge:         nil,
	}
	ret := &Challenge{
		parent:            parent,
		winner:            nil,
		root:              root,
		latestConfirmed:   root,
		creationTime:      parent.chain.timeReference.Get(),
		includedHistories: make(map[common.Hash]bool),
		nextSequenceNum:   1,
	}
	root.challenge = ret
	ret.includedHistories[root.Commitment.Hash()] = true
	parent.challenge = util.Some[*Challenge](ret)
	parentStaker := common.Address{}
	if !parent.Staker.IsNone() {
		parentStaker = parent.Staker.Unwrap()
	}
	parent.chain.feed.Append(&StartChallengeEvent{
		ParentSeqNum:          parent.SequenceNum,
		ParentStateCommitment: parent.StateCommitment,
		ParentStaker:          parentStaker,
		Challenger:            challenger,
	})

	parent.chain.challengeVerticesByChallengeID[ChallengeID(parent.StateCommitment.Hash())] = []*ChallengeVertex{root}

	return ret, nil
}

func (chal *Challenge) ParentStateCommitment() StateCommitment {
	return chal.parent.StateCommitment
}

func (chal *Challenge) AddLeaf(tx *ActiveTx, assertion *Assertion, history util.HistoryCommitment, challenger common.Address) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if assertion.Prev.IsNone() {
		return nil, ErrInvalid
	}
	prev := assertion.Prev.Unwrap()
	if prev != chal.parent {
		return nil, ErrInvalid
	}
	if chal.Completed(tx) {
		return nil, ErrWrongState
	}
	chain := assertion.chain
	if !chal.root.eligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}

	timer := util.NewCountUpTimer(chain.timeReference)
	if assertion.isFirstChild {
		delta := prev.secondChildCreationTime.Unwrap().Sub(prev.firstChildCreationTime.Unwrap())
		timer.Set(delta)
	}
	leaf := &ChallengeVertex{
		challenge:            chal,
		SequenceNum:          chal.nextSequenceNum,
		Challenger:           challenger,
		isLeaf:               true,
		status:               PendingAssertionState,
		Commitment:           history,
		Prev:                 chal.root,
		presumptiveSuccessor: nil,
		psTimer:              timer,
		subChallenge:         nil,
		winnerIfConfirmed:    assertion,
	}
	chal.nextSequenceNum++
	chal.root.maybeNewPresumptiveSuccessor(leaf)
	chal.parent.chain.challengesFeed.Append(&ChallengeLeafEvent{
		ParentSeqNum:      leaf.Prev.SequenceNum,
		SequenceNum:       leaf.SequenceNum,
		WinnerIfConfirmed: assertion.SequenceNum,
		History:           history,
		BecomesPS:         leaf.Prev.presumptiveSuccessor == leaf,
		Actor:             challenger,
	})
	id := ChallengeID(chal.parent.StateCommitment.Hash())
	chal.parent.chain.challengesByID[id] = chal
	chal.parent.chain.challengeVerticesByChallengeID[id] = append(
		chal.parent.chain.challengeVerticesByChallengeID[id],
		leaf,
	)
	return leaf, nil
}

func (chal *Challenge) Completed(tx *ActiveTx) bool {
	tx.verifyRead()
	return chal.winner != nil
}

func (chal *Challenge) Winner(tx *ActiveTx) (*Assertion, error) {
	tx.verifyRead()
	if chal.winner == nil {
		return nil, ErrNoWinnerYet
	}
	return chal.winner, nil
}

type ChallengeVertex struct {
	Commitment           util.HistoryCommitment
	challenge            *Challenge
	SequenceNum          uint64 // unique within the challenge
	Challenger           common.Address
	isLeaf               bool
	status               AssertionState
	Prev                 *ChallengeVertex
	presumptiveSuccessor *ChallengeVertex
	psTimer              *util.CountUpTimer
	subChallenge         *SubChallenge
	winnerIfConfirmed    *Assertion
}

func (vertex *ChallengeVertex) eligibleForNewSuccessor() bool {
	return vertex.presumptiveSuccessor == nil || vertex.presumptiveSuccessor.psTimer.Get() <= vertex.challenge.parent.chain.challengePeriod
}

func (vertex *ChallengeVertex) maybeNewPresumptiveSuccessor(succ *ChallengeVertex) {
	if vertex.presumptiveSuccessor != nil && succ.Commitment.Height < vertex.presumptiveSuccessor.Commitment.Height {
		vertex.presumptiveSuccessor.psTimer.Stop()
		vertex.presumptiveSuccessor = nil
	}
	if vertex.presumptiveSuccessor == nil {
		vertex.presumptiveSuccessor = succ
		succ.psTimer.Start()
	}
}

func (vertex *ChallengeVertex) IsPresumptiveSuccessor() bool {
	return vertex.Prev == nil || vertex.Prev.presumptiveSuccessor == vertex
}

func (vertex *ChallengeVertex) requiredBisectionHeight() (uint64, error) {
	return util.BisectionPoint(vertex.Prev.Commitment.Height, vertex.Commitment.Height)
}

func (vertex *ChallengeVertex) Bisect(tx *ActiveTx, history util.HistoryCommitment, proof []common.Hash, challenger common.Address) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if vertex.IsPresumptiveSuccessor() {
		return nil, ErrWrongState
	}
	if !vertex.Prev.eligibleForNewSuccessor() {
		return nil, ErrPastDeadline
	}
	if vertex.challenge.includedHistories[history.Hash()] {
		return nil, ErrVertexAlreadyExists
	}
	bisectionHeight, err := vertex.requiredBisectionHeight()
	if err != nil {
		return nil, err
	}
	if bisectionHeight != history.Height {
		return nil, ErrInvalidHeight
	}
	if err := util.VerifyPrefixProof(history, vertex.Commitment, proof); err != nil {
		return nil, err
	}

	vertex.psTimer.Stop()
	newVertex := &ChallengeVertex{
		challenge:            vertex.challenge,
		SequenceNum:          vertex.challenge.nextSequenceNum,
		Challenger:           challenger,
		isLeaf:               false,
		Commitment:           history,
		Prev:                 vertex.Prev,
		presumptiveSuccessor: nil,
		psTimer:              vertex.psTimer.Clone(),
	}
	newVertex.challenge.nextSequenceNum++
	newVertex.maybeNewPresumptiveSuccessor(vertex)
	newVertex.Prev.maybeNewPresumptiveSuccessor(newVertex)
	newVertex.challenge.includedHistories[history.Hash()] = true

	vertex.Prev = newVertex

	newVertex.challenge.parent.chain.challengesFeed.Append(&ChallengeBisectEvent{
		FromSequenceNum: vertex.SequenceNum,
		SequenceNum:     newVertex.SequenceNum,
		History:         newVertex.Commitment,
		BecomesPS:       newVertex.Prev.presumptiveSuccessor == newVertex,
		Actor:           challenger,
	})
	id := ChallengeID(newVertex.challenge.parent.StateCommitment.Hash())
	newVertex.challenge.parent.chain.challengeVerticesByChallengeID[id] = append(
		newVertex.challenge.parent.chain.challengeVerticesByChallengeID[id],
		newVertex,
	)
	return newVertex, nil
}

func (vertex *ChallengeVertex) Merge(tx *ActiveTx, newPrev *ChallengeVertex, proof []common.Hash, challenger common.Address) error {
	tx.verifyReadWrite()
	if !newPrev.eligibleForNewSuccessor() {
		return ErrPastDeadline
	}
	if vertex.Prev != newPrev.Prev {
		return ErrInvalid
	}
	if vertex.Commitment.Height <= newPrev.Commitment.Height {
		return ErrInvalidHeight
	}
	if err := util.VerifyPrefixProof(newPrev.Commitment, vertex.Commitment, proof); err != nil {
		return err
	}

	vertex.Prev = newPrev
	newPrev.psTimer.Add(vertex.psTimer.Get())
	newPrev.maybeNewPresumptiveSuccessor(vertex)
	vertex.challenge.parent.chain.challengesFeed.Append(&ChallengeMergeEvent{
		DeeperSequenceNum:    vertex.SequenceNum,
		ShallowerSequenceNum: newPrev.SequenceNum,
		BecomesPS:            newPrev.presumptiveSuccessor == vertex,
		History:              newPrev.Commitment,
		Actor:                challenger,
	})
	return nil
}

func (vertex *ChallengeVertex) ConfirmForSubChallengeWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if vertex.status != PendingAssertionState {
		return ErrWrongState
	}
	if vertex.Prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	subChal := vertex.Prev.subChallenge
	if subChal == nil || subChal.winner != vertex {
		return ErrInvalid
	}
	vertex._confirm()
	return nil
}

func (vertex *ChallengeVertex) ConfirmForPsTimer(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if vertex.status != PendingAssertionState {
		return ErrWrongState
	}
	if vertex.Prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	if vertex.psTimer.Get() <= vertex.challenge.parent.chain.challengePeriod {
		return ErrNotYet
	}
	vertex._confirm()
	return nil
}

func (vertex *ChallengeVertex) ConfirmForChallengeDeadline(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if vertex.status != PendingAssertionState {
		return ErrWrongState
	}
	if vertex.Prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	chain := vertex.challenge.parent.chain
	chalPeriod := chain.challengePeriod
	if !chain.timeReference.Get().After(vertex.challenge.creationTime.Add(2 * chalPeriod)) {
		return ErrNotYet
	}
	vertex._confirm()
	return nil
}

func (vertex *ChallengeVertex) _confirm() {
	vertex.status = ConfirmedAssertionState
	if vertex.isLeaf {
		vertex.challenge.winner = vertex.winnerIfConfirmed
	}
}

func (vertex *ChallengeVertex) CreateSubChallenge(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if vertex.subChallenge != nil {
		return ErrVertexAlreadyExists
	}
	if vertex.status == ConfirmedAssertionState {
		return ErrWrongState
	}
	vertex.subChallenge = &SubChallenge{
		parent: vertex,
		winner: nil,
	}
	return nil
}

type SubChallenge struct {
	parent *ChallengeVertex
	winner *ChallengeVertex
}

func (sc *SubChallenge) SetWinner(tx *ActiveTx, winner *ChallengeVertex) error {
	tx.verifyReadWrite()
	if sc.winner != nil {
		return ErrInvalid
	}
	if winner.Prev != sc.parent {
		return ErrInvalid
	}
	sc.winner = winner
	return nil
}

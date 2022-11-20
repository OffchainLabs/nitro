package protocol

import (
	"context"
	"encoding/binary"
	"errors"
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

	ErrWrongChain            = errors.New("wrong chain")
	ErrInvalid               = errors.New("invalid operation")
	ErrInvalidHeight         = errors.New("invalid block height")
	ErrVertexAlreadyExists   = errors.New("vertex already exists")
	ErrWrongState            = errors.New("vertex state does not allow this operation")
	ErrWrongPredecessorState = errors.New("predecessor state does not allow this operation")
	ErrNotYet                = errors.New("deadline has not yet passed")
	ErrNoWinnerYet           = errors.New("challenges does not yet have a winner")
	ErrPastDeadline          = errors.New("deadline has passed")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrNotImplemented        = errors.New("not yet implemented")
)

// OnChainProtocol defines an interface for interacting with the smart contract implementation
// of the assertion protocol, with methods to issue mutating transactions, make eth calls, create
// leafs in the protocol, issue challenges, and subscribe to chain events wrapped in simple abstractions.
type OnChainProtocol interface {
	ChainReader
	ChainWriter
	EventProvider
	AssertionManager
}

// ChainReader can make non-mutating calls to the on-chain protocol.
type ChainReader interface {
	ChallengePeriodLength(tx *ActiveTx) time.Duration
	Call(clo func(*ActiveTx, *AssertionChain) error) error
}

// ChainWriter can make mutating calls to the on-chain protocol.
type ChainWriter interface {
	Tx(clo func(*ActiveTx, *AssertionChain) error) error
}

// EventProvider allows subscribing to chain events for the on-chain protocol.
type EventProvider interface {
	SubscribeChainEvents(ctx context.Context, ch chan<- AssertionChainEvent)
}

// AssertionManager allows the creation of new leaves for a Staker with a State Commitment
// and a previous assertion.
type AssertionManager interface {
	LatestConfirmed(*ActiveTx) *Assertion
	CreateLeaf(tx *ActiveTx, prev *Assertion, commitment StateCommitment, staker common.Address) (*Assertion, error)
}

type AssertionChain struct {
	mutex           sync.RWMutex
	timeReference   util.TimeReference
	challengePeriod time.Duration
	confirmedLatest uint64
	assertions      []*Assertion
	dedupe          map[common.Hash]bool
	balances        *util.MapWithDefault[common.Address, *big.Int]
	feed            *EventFeed[AssertionChainEvent]
	inbox           *Inbox
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

func (chain *AssertionChain) Tx(clo func(tx *ActiveTx, chain *AssertionChain) error) error {
	chain.mutex.Lock()
	defer chain.mutex.Unlock()
	tx := &ActiveTx{txStatus: readWriteTxStatus}
	err := clo(tx, chain)
	tx.txStatus = deadTxStatus
	return err
}

func (chain *AssertionChain) Call(clo func(tx *ActiveTx, chain *AssertionChain) error) error {
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
	chain                   *AssertionChain
	status                  AssertionState
	SequenceNum             uint64
	StateCommitment         StateCommitment
	prev                    util.Option[*Assertion]
	isFirstChild            bool
	firstChildCreationTime  util.Option[time.Time]
	secondChildCreationTime util.Option[time.Time]
	challenge               util.Option[*Challenge]
	staker                  util.Option[common.Address]
}

type StateCommitment struct {
	Height    uint64
	StateRoot common.Hash
}

func (comm *StateCommitment) Hash() common.Hash {
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
		prev:                    util.EmptyOption[*Assertion](),
		isFirstChild:            false,
		firstChildCreationTime:  util.EmptyOption[time.Time](),
		secondChildCreationTime: util.EmptyOption[time.Time](),
		challenge:               util.EmptyOption[*Challenge](),
		staker:                  util.EmptyOption[common.Address](),
	}
	chain := &AssertionChain{
		mutex:           sync.RWMutex{},
		timeReference:   timeRef,
		challengePeriod: challengePeriod,
		confirmedLatest: 0,
		assertions:      []*Assertion{genesis},
		dedupe:          make(map[common.Hash]bool), // no need to insert genesis assertion here
		balances:        util.NewMapWithDefaultAdvanced[common.Address, *big.Int](common.Big0, func(x *big.Int) bool { return x.Sign() == 0 }),
		feed:            NewEventFeed[AssertionChainEvent](ctx),
		inbox:           NewInbox(ctx),
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

func (chain *AssertionChain) SubscribeChainEvents(ctx context.Context, ch chan<- AssertionChainEvent) {
	chain.feed.Subscribe(ctx, ch)
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

	if err := prev.staker.IfLet(
		func(oldStaker common.Address) error {
			if staker != oldStaker {
				if err := chain.DeductFromBalance(tx, staker, AssertionStakeWei); err != nil {
					return err
				}
				chain.AddToBalance(tx, oldStaker, AssertionStakeWei)
				prev.staker = util.EmptyOption[common.Address]()
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
		prev:                    util.FullOption[*Assertion](prev),
		isFirstChild:            prev.firstChildCreationTime.IsEmpty(),
		firstChildCreationTime:  util.EmptyOption[time.Time](),
		secondChildCreationTime: util.EmptyOption[time.Time](),
		challenge:               util.EmptyOption[*Challenge](),
		staker:                  util.FullOption[common.Address](staker),
	}
	if prev.firstChildCreationTime.IsEmpty() {
		prev.firstChildCreationTime = util.FullOption[time.Time](chain.timeReference.Get())
	} else if prev.secondChildCreationTime.IsEmpty() {
		prev.secondChildCreationTime = util.FullOption[time.Time](chain.timeReference.Get())
	}
	chain.assertions = append(chain.assertions, leaf)
	chain.dedupe[dedupeCode] = true
	chain.feed.Append(&CreateLeafEvent{
		PrevSeqNum:      prev.SequenceNum,
		SeqNum:          leaf.SequenceNum,
		StateCommitment: leaf.StateCommitment,
		Staker:          staker,
	})
	return leaf, nil
}

func (a *Assertion) RejectForPrev(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.prev.IsEmpty() {
		return ErrInvalid
	}
	if a.prev.OpenKnownFull().status != RejectedAssertionState {
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
	if a.prev.IsEmpty() {
		return ErrInvalid
	}
	chal := a.prev.OpenKnownFull().challenge
	if chal.IsEmpty() {
		return util.ErrOptionIsEmpty
	}
	winner, err := chal.OpenKnownFull().Winner(tx)
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
	if a.prev.IsEmpty() {
		return ErrInvalid
	}
	prev := a.prev.OpenKnownFull()
	if prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	if !prev.secondChildCreationTime.IsEmpty() {
		return ErrInvalid
	}
	if !a.chain.timeReference.Get().After(prev.firstChildCreationTime.OpenKnownFull().Add(a.chain.challengePeriod)) {
		return ErrNotYet
	}
	a.status = ConfirmedAssertionState
	a.chain.confirmedLatest = a.SequenceNum
	a.chain.feed.Append(&ConfirmEvent{
		SeqNum: a.SequenceNum,
	})
	if !a.staker.IsEmpty() {
		a.chain.AddToBalance(tx, a.staker.OpenKnownFull(), AssertionStakeWei)
		a.staker = util.EmptyOption[common.Address]()
	}
	return nil
}

func (a *Assertion) ConfirmForWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if a.status != PendingAssertionState {
		return ErrWrongState
	}
	if a.prev.IsEmpty() {
		return ErrInvalid
	}
	prev := a.prev.OpenKnownFull()
	if prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	if prev.challenge.IsEmpty() {
		return ErrWrongPredecessorState
	}
	winner, err := prev.challenge.OpenKnownFull().Winner(tx)
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
	feed              *EventFeed[ChallengeEvent]
}

func (parent *Assertion) CreateChallenge(tx *ActiveTx, ctx context.Context) (*Challenge, error) {
	tx.verifyReadWrite()
	if parent.status != PendingAssertionState && parent.chain.LatestConfirmed(tx) != parent {
		return nil, ErrWrongState
	}
	if !parent.challenge.IsEmpty() {
		return nil, ErrInvalid
	}
	if parent.secondChildCreationTime.IsEmpty() {
		return nil, ErrInvalid
	}
	root := &ChallengeVertex{
		challenge:   nil,
		sequenceNum: 0,
		isLeaf:      false,
		status:      ConfirmedAssertionState,
		commitment: util.HistoryCommitment{
			Height: 0,
			Merkle: common.Hash{},
		},
		prev:                 nil,
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
		feed:              NewEventFeed[ChallengeEvent](ctx),
	}
	root.challenge = ret
	ret.includedHistories[root.commitment.Hash()] = true
	parent.challenge = util.FullOption[*Challenge](ret)
	parent.chain.feed.Append(&StartChallengeEvent{
		ParentSeqNum: parent.SequenceNum,
	})
	return ret, nil
}

func (chal *Challenge) AddLeaf(tx *ActiveTx, assertion *Assertion, history util.HistoryCommitment) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if assertion.prev.IsEmpty() {
		return nil, ErrInvalid
	}
	prev := assertion.prev.OpenKnownFull()
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
		delta := prev.secondChildCreationTime.OpenKnownFull().Sub(prev.firstChildCreationTime.OpenKnownFull())
		timer.Set(delta)
	}
	leaf := &ChallengeVertex{
		challenge:            chal,
		sequenceNum:          chal.nextSequenceNum,
		isLeaf:               true,
		status:               PendingAssertionState,
		commitment:           history,
		prev:                 chal.root,
		presumptiveSuccessor: nil,
		psTimer:              timer,
		subChallenge:         nil,
		winnerIfConfirmed:    assertion,
	}
	chal.nextSequenceNum++
	chal.root.maybeNewPresumptiveSuccessor(leaf)
	chal.feed.Append(&ChallengeLeafEvent{
		SequenceNum:       leaf.sequenceNum,
		WinnerIfConfirmed: assertion.SequenceNum,
		History:           history,
		BecomesPS:         leaf.prev.presumptiveSuccessor == leaf,
	})
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
	challenge            *Challenge
	sequenceNum          uint64 // unique within the challenge
	isLeaf               bool
	status               AssertionState
	commitment           util.HistoryCommitment
	prev                 *ChallengeVertex
	presumptiveSuccessor *ChallengeVertex
	psTimer              *util.CountUpTimer
	subChallenge         *SubChallenge
	winnerIfConfirmed    *Assertion
}

func (vertex *ChallengeVertex) eligibleForNewSuccessor() bool {
	return vertex.presumptiveSuccessor == nil || vertex.presumptiveSuccessor.psTimer.Get() <= vertex.challenge.parent.chain.challengePeriod
}

func (vertex *ChallengeVertex) maybeNewPresumptiveSuccessor(succ *ChallengeVertex) {
	if vertex.presumptiveSuccessor != nil && succ.commitment.Height < vertex.presumptiveSuccessor.commitment.Height {
		vertex.presumptiveSuccessor.psTimer.Stop()
		vertex.presumptiveSuccessor = nil
	}
	if vertex.presumptiveSuccessor == nil {
		vertex.presumptiveSuccessor = succ
		succ.psTimer.Start()
	}
}

func (vertex *ChallengeVertex) isPresumptiveSuccessor() bool {
	return vertex.prev == nil || vertex.prev.presumptiveSuccessor == vertex
}

func (vertex *ChallengeVertex) requiredBisectionHeight() (uint64, error) {
	return util.BisectionPoint(vertex.prev.commitment.Height, vertex.commitment.Height)
}

func (vertex *ChallengeVertex) Bisect(tx *ActiveTx, history util.HistoryCommitment, proof []common.Hash) (*ChallengeVertex, error) {
	tx.verifyReadWrite()
	if vertex.isPresumptiveSuccessor() {
		return nil, ErrWrongState
	}
	if !vertex.prev.eligibleForNewSuccessor() {
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
	if err := util.VerifyPrefixProof(history, vertex.commitment, proof); err != nil {
		return nil, err
	}

	vertex.psTimer.Stop()
	newVertex := &ChallengeVertex{
		challenge:            vertex.challenge,
		sequenceNum:          vertex.challenge.nextSequenceNum,
		isLeaf:               false,
		commitment:           history,
		prev:                 vertex.prev,
		presumptiveSuccessor: nil,
		psTimer:              vertex.psTimer.Clone(),
	}
	newVertex.challenge.nextSequenceNum++
	newVertex.maybeNewPresumptiveSuccessor(vertex)
	newVertex.prev.maybeNewPresumptiveSuccessor(newVertex)
	newVertex.challenge.includedHistories[history.Hash()] = true
	newVertex.challenge.feed.Append(&ChallengeBisectEvent{
		FromSequenceNum: vertex.sequenceNum,
		SequenceNum:     newVertex.sequenceNum,
		History:         newVertex.commitment,
		BecomesPS:       newVertex.prev.presumptiveSuccessor == newVertex,
	})
	return newVertex, nil
}

func (vertex *ChallengeVertex) Merge(tx *ActiveTx, newPrev *ChallengeVertex, proof []common.Hash) error {
	tx.verifyReadWrite()
	if !newPrev.eligibleForNewSuccessor() {
		return ErrPastDeadline
	}
	if vertex.prev != newPrev.prev {
		return ErrInvalid
	}
	if vertex.commitment.Height <= newPrev.commitment.Height {
		return ErrInvalidHeight
	}
	if err := util.VerifyPrefixProof(newPrev.commitment, vertex.commitment, proof); err != nil {
		return err
	}

	vertex.prev = newPrev
	newPrev.psTimer.Add(vertex.psTimer.Get())
	newPrev.maybeNewPresumptiveSuccessor(vertex)
	vertex.challenge.feed.Append(&ChallengeMergeEvent{
		DeeperSequenceNum:    vertex.sequenceNum,
		ShallowerSequenceNum: newPrev.sequenceNum,
		BecomesPS:            newPrev.presumptiveSuccessor == vertex,
	})
	return nil
}

func (vertex *ChallengeVertex) ConfirmForSubChallengeWin(tx *ActiveTx) error {
	tx.verifyReadWrite()
	if vertex.status != PendingAssertionState {
		return ErrWrongState
	}
	if vertex.prev.status != ConfirmedAssertionState {
		return ErrWrongPredecessorState
	}
	subChal := vertex.prev.subChallenge
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
	if vertex.prev.status != ConfirmedAssertionState {
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
	if vertex.prev.status != ConfirmedAssertionState {
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
	if winner.prev != sc.parent {
		return ErrInvalid
	}
	sc.winner = winner
	return nil
}

package mocks

import (
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"math/big"
	"time"
)

type MockChallengeVertex struct {
	mock.Mock
	MockId         [32]byte
	MockSeqNum     protocol.VertexSequenceNumber
	MockStatus     protocol.AssertionState
	MockHistory    util.HistoryCommitment
	MockMiniStaker common.Address
	MockPrev       util.Option[protocol.ChallengeVertex]
	MockSubChal    util.Option[protocol.Challenge]
}

func (m *MockChallengeVertex) Id() [32]byte {
	return m.MockId
}

func (m *MockChallengeVertex) SequenceNum() protocol.VertexSequenceNumber {
	return m.MockSeqNum
}

func (m *MockChallengeVertex) Status(ctx context.Context, tx protocol.ActiveTx) (protocol.AssertionState, error) {
	return m.MockStatus, nil
}

func (m *MockChallengeVertex) HistoryCommitment(ctx context.Context, tx protocol.ActiveTx) (util.HistoryCommitment, error) {
	return m.MockHistory, nil
}

func (m *MockChallengeVertex) MiniStaker(ctx context.Context, tx protocol.ActiveTx) (common.Address, error) {
	return m.MockMiniStaker, nil
}

func (m *MockChallengeVertex) Prev(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.ChallengeVertex], error) {
	return m.MockPrev, nil
}

func (m *MockChallengeVertex) GetSubChallenge(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.Challenge], error) {
	return m.MockSubChal, nil
}

func (m *MockChallengeVertex) HasConfirmedSibling(
	ctx context.Context,
	tx protocol.ActiveTx,
) (bool, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(bool), args.Error(1)
}

// Presumptive status / timer readers.
func (m *MockChallengeVertex) EligibleForNewSuccessor(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallengeVertex) IsPresumptiveSuccessor(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallengeVertex) PresumptiveSuccessor(
	ctx context.Context, tx protocol.ActiveTx,
) (util.Option[protocol.ChallengeVertex], error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(util.Option[protocol.ChallengeVertex]), args.Error(1)
}

func (m *MockChallengeVertex) PsTimer(ctx context.Context, tx protocol.ActiveTx) (uint64, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockChallengeVertex) ChessClockExpired(
	ctx context.Context,
	tx protocol.ActiveTx,
	challengePeriodSeconds time.Duration,
) (bool, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallengeVertex) ChildrenAreAtOneStepFork(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(bool), args.Error(1)
}

// Mutating calls for challenge moves.
func (m *MockChallengeVertex) CreateSubChallenge(
	ctx context.Context,
	tx protocol.ActiveTx,
) (protocol.Challenge, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(protocol.Challenge), args.Error(1)
}

func (m *MockChallengeVertex) Bisect(
	ctx context.Context,
	tx protocol.ActiveTx,
	history util.HistoryCommitment,
	proof []byte,
) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, tx, history, proof)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

func (m *MockChallengeVertex) Merge(
	ctx context.Context,
	tx protocol.ActiveTx,
	mergingToHistory util.HistoryCommitment,
	proof []byte,
) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, tx, mergingToHistory, proof)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

// Mutating calls for confirmations.
func (m *MockChallengeVertex) ConfirmForPsTimer(ctx context.Context, tx protocol.ActiveTx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockChallengeVertex) ConfirmForChallengeDeadline(ctx context.Context, tx protocol.ActiveTx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockChallengeVertex) ConfirmForSubChallengeWin(ctx context.Context, tx protocol.ActiveTx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

type MockAssertion struct {
	Prev           util.Option[*MockAssertion]
	MockHeight     uint64
	MockSeqNum     protocol.AssertionSequenceNumber
	MockPrevSeqNum protocol.AssertionSequenceNumber
	MockStateHash  common.Hash
}

func (m *MockAssertion) Height() (uint64, error) {
	return m.MockHeight, nil

}

func (m *MockAssertion) SeqNum() protocol.AssertionSequenceNumber {
	return m.MockSeqNum
}

func (m *MockAssertion) PrevSeqNum() (protocol.AssertionSequenceNumber, error) {
	return m.MockPrevSeqNum, nil
}

func (m *MockAssertion) StateHash() (common.Hash, error) {
	return m.MockStateHash, nil
}

type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) LatestAssertionCreationData(ctx context.Context, prevHeight uint64) (*statemanager.AssertionToCreate, error) {
	args := m.Called(ctx, prevHeight)
	return args.Get(0).(*statemanager.AssertionToCreate), args.Error(1)
}

func (m *MockStateManager) HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error) {

	args := m.Called(ctx, height)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) PrefixProof(ctx context.Context, from, to uint64) ([]byte, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStateManager) HasStateCommitment(ctx context.Context, commit util.StateCommitment) bool {
	args := m.Called(ctx, commit)
	return args.Bool(0)
}

func (m *MockStateManager) BigStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) BigStepCommitmentUpTo(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toBigStep uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight, toBigStep)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) SmallStepLeafCommitment(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) SmallStepCommitmentUpTo(
	ctx context.Context,
	fromAssertionHeight,
	toAssertionHeight,
	toStep uint64,
) (util.HistoryCommitment, error) {
	args := m.Called(ctx, fromAssertionHeight, toAssertionHeight, toStep)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

type MockActiveTx struct {
	ReadWriteTx bool
	Finalized   *big.Int
	Head        *big.Int
	From        common.Address
}

func (m *MockActiveTx) FinalizedBlockNumber() *big.Int {
	return m.Finalized
}

func (m *MockActiveTx) HeadBlockNumber() *big.Int {
	return m.Head
}

func (m *MockActiveTx) ReadOnly() bool {
	return !m.ReadWriteTx
}

func (m *MockActiveTx) Sender() common.Address {
	return m.From
}

type MockChallenge struct {
	mock.Mock
	MockID                protocol.ChallengeHash
	MockType              protocol.ChallengeType
	MockWinningAssertion  util.Option[protocol.AssertionHash]
	MockAssertion         protocol.Assertion
	MockRootVertex        protocol.ChallengeVertex
	MockCreationTime      time.Time
	MockParentStateCommit util.StateCommitment
	MockWinnerVertex      util.Option[protocol.ChallengeVertex]
	MockChallenger        common.Address
}

// Getters.
func (m *MockChallenge) Id() protocol.ChallengeHash {
	return m.MockID
}

func (m *MockChallenge) GetType(ctx context.Context, tx protocol.ActiveTx) (protocol.ChallengeType, error) {
	return m.MockType, nil
}

func (m *MockChallenge) WinningClaim(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.AssertionHash], error) {
	return m.MockWinningAssertion, nil
}

func (m *MockChallenge) RootAssertion(ctx context.Context, tx protocol.ActiveTx) (protocol.Assertion, error) {
	return m.MockAssertion, nil
}

func (m *MockChallenge) RootVertex(ctx context.Context, tx protocol.ActiveTx) (protocol.ChallengeVertex, error) {
	return m.MockRootVertex, nil
}

func (m *MockChallenge) GetCreationTime(ctx context.Context, tx protocol.ActiveTx) (time.Time, error) {
	return m.MockCreationTime, nil
}

func (m *MockChallenge) ParentStateCommitment(ctx context.Context, tx protocol.ActiveTx) (util.StateCommitment, error) {
	return m.MockParentStateCommit, nil
}

func (m *MockChallenge) WinnerVertex(ctx context.Context, tx protocol.ActiveTx) (util.Option[protocol.ChallengeVertex], error) {
	return m.MockWinnerVertex, nil
}

func (m *MockChallenge) Completed(ctx context.Context, tx protocol.ActiveTx) (bool, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockChallenge) Challenger(ctx context.Context, tx protocol.ActiveTx) (common.Address, error) {
	return m.MockChallenger, nil
}

// Mutating calls.
func (m *MockChallenge) AddBlockChallengeLeaf(
	ctx context.Context,
	tx protocol.ActiveTx,
	assertion protocol.Assertion,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, tx, assertion, history)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

func (m *MockChallenge) AddSubChallengeLeaf(
	ctx context.Context,
	tx protocol.ActiveTx,
	vertex protocol.ChallengeVertex,
	history util.HistoryCommitment,
) (protocol.ChallengeVertex, error) {
	args := m.Called(ctx, tx, vertex, history)
	return args.Get(0).(protocol.ChallengeVertex), args.Error(1)
}

type MockChallengeManager struct {
	mock.Mock
	MockAddr common.Address
}

func (m *MockChallengeManager) ChallengePeriodSeconds(
	ctx context.Context, tx protocol.ActiveTx,
) (time.Duration, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockChallengeManager) CalculateChallengeHash(
	ctx context.Context,
	tx protocol.ActiveTx,
	itemId common.Hash,
	challengeType protocol.ChallengeType,
) (protocol.ChallengeHash, error) {
	args := m.Called(ctx, tx, itemId, challengeType)
	return args.Get(0).(protocol.ChallengeHash), args.Error(1)
}

func (m *MockChallengeManager) CalculateChallengeVertexId(
	ctx context.Context,
	tx protocol.ActiveTx,
	challengeId protocol.ChallengeHash,
	history util.HistoryCommitment,
) (protocol.VertexHash, error) {
	args := m.Called(ctx, tx, challengeId, history)
	return args.Get(0).(protocol.VertexHash), args.Error(1)
}

func (m *MockChallengeManager) GetVertex(
	ctx context.Context,
	tx protocol.ActiveTx,
	vertexId protocol.VertexHash,
) (util.Option[protocol.ChallengeVertex], error) {
	args := m.Called(ctx, tx, vertexId)
	return args.Get(0).(util.Option[protocol.ChallengeVertex]), args.Error(1)
}

func (m *MockChallengeManager) GetChallenge(
	ctx context.Context,
	tx protocol.ActiveTx,
	challengeId protocol.ChallengeHash,
) (util.Option[protocol.Challenge], error) {
	args := m.Called(ctx, tx, challengeId)
	return args.Get(0).(util.Option[protocol.Challenge]), args.Error(1)
}

func (m *MockChallengeManager) Address() common.Address {
	return m.MockAddr
}

type MockProtocol struct {
	mock.Mock
}

func (m *MockProtocol) Call(callback func(protocol.ActiveTx) error) error {
	return callback(&MockActiveTx{ReadWriteTx: false})
}

func (m *MockProtocol) Tx(callback func(protocol.ActiveTx) error) error {
	return callback(&MockActiveTx{ReadWriteTx: true})
}

// Read-only methods.
func (m *MockProtocol) NumAssertions(
	ctx context.Context,
	tx protocol.ActiveTx,
) (uint64, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockProtocol) AssertionBySequenceNum(
	ctx context.Context,
	tx protocol.ActiveTx,
	seqNum protocol.AssertionSequenceNumber,
) (protocol.Assertion, error) {
	args := m.Called(ctx, tx, seqNum)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) GetAssertionId(
	ctx context.Context,
	tx protocol.ActiveTx,
	seqNum protocol.AssertionSequenceNumber,
) (protocol.AssertionHash, error) {
	args := m.Called(ctx, tx, seqNum)
	return args.Get(0).(protocol.AssertionHash), args.Error(1)
}

func (m *MockProtocol) GetAssertionNum(
	ctx context.Context,
	tx protocol.ActiveTx,
	assertionHash protocol.AssertionHash,
) (protocol.AssertionSequenceNumber, error) {
	args := m.Called(ctx, tx, assertionHash)
	return args.Get(0).(protocol.AssertionSequenceNumber), args.Error(1)
}

func (m *MockProtocol) LatestConfirmed(ctx context.Context, tx protocol.ActiveTx) (protocol.Assertion, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) CurrentChallengeManager(ctx context.Context, tx protocol.ActiveTx) (protocol.ChallengeManager, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).(protocol.ChallengeManager), args.Error(1)
}

// Mutating methods.
func (m *MockProtocol) CreateAssertion(
	ctx context.Context,
	tx protocol.ActiveTx,
	height uint64,
	prevSeqNum protocol.AssertionSequenceNumber,
	prevAssertionState *protocol.ExecutionState,
	postState *protocol.ExecutionState,
	prevInboxMaxCount *big.Int,
) (protocol.Assertion, error) {
	args := m.Called(ctx, tx, height, prevSeqNum, prevAssertionState, postState, prevInboxMaxCount)
	return args.Get(0).(protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) CreateSuccessionChallenge(
	ctx context.Context, tx protocol.ActiveTx, seqNum protocol.AssertionSequenceNumber,
) (protocol.Challenge, error) {
	args := m.Called(ctx, tx, seqNum)
	return args.Get(0).(protocol.Challenge), args.Error(1)
}

func (m *MockProtocol) Confirm(
	ctx context.Context, tx protocol.ActiveTx, blockHash, sendRoot common.Hash,
) error {
	args := m.Called(ctx, tx, blockHash, sendRoot)
	return args.Error(0)
}

func (m *MockProtocol) Reject(
	ctx context.Context, tx protocol.ActiveTx, staker common.Address,
) error {
	args := m.Called(ctx, tx, staker)
	return args.Error(0)
}

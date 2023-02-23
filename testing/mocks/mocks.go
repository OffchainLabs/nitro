package mocks

import (
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"math/big"
)

type MockAssertion struct {
	Prev           util.Option[*MockAssertion]
	MockHeight     uint64
	MockSeqNum     protocol.AssertionSequenceNumber
	MockPrevSeqNum protocol.AssertionSequenceNumber
	MockStateHash  common.Hash
}

func (m *MockAssertion) Height() uint64 {
	return m.MockHeight

}

func (m *MockAssertion) SeqNum() protocol.AssertionSequenceNumber {
	return m.MockSeqNum
}

func (m *MockAssertion) PrevSeqNum() protocol.AssertionSequenceNumber {
	return m.MockPrevSeqNum
}

func (m *MockAssertion) StateHash() common.Hash {
	return m.MockStateHash
}

type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) LatestAssertionCreationData(ctx context.Context) (*statemanager.AssertionToCreate, error) {
	args := m.Called(ctx)
	return args.Get(0).(*statemanager.AssertionToCreate), args.Error(1)
}

func (m *MockStateManager) LatestHistoryCommitment(ctx context.Context) (util.HistoryCommitment, error) {
	args := m.Called(ctx)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) HasHistoryCommitment(ctx context.Context, commit util.HistoryCommitment) bool {
	args := m.Called(ctx, commit)
	return args.Bool(0)
}

func (m *MockStateManager) HistoryCommitmentUpTo(ctx context.Context, height uint64) (util.HistoryCommitment, error) {

	args := m.Called(ctx, height)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) PrefixProof(ctx context.Context, from, to uint64) ([]common.Hash, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).([]common.Hash), args.Error(1)
}

func (m *MockStateManager) HasStateCommitment(ctx context.Context, commit util.StateCommitment) bool {
	args := m.Called(ctx, commit)
	return args.Bool(0)
}

func (m *MockStateManager) StateCommitmentAtHeight(ctx context.Context, height uint64) (util.StateCommitment, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(util.StateCommitment), args.Error(1)
}

func (m *MockStateManager) LatestStateCommitment(ctx context.Context) (util.StateCommitment, error) {
	args := m.Called(ctx)
	return args.Get(0).(util.StateCommitment), args.Error(1)
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

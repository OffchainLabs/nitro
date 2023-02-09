package mocks

import (
	"context"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockStateManager struct {
	mock.Mock
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

type MockProtocol struct {
	mock.Mock
}

func (m *MockProtocol) Inbox() *protocol.Inbox {
	args := m.Called()
	return args.Get(0).(*protocol.Inbox)
}

func (m *MockProtocol) Tx(clo func(tx *protocol.ActiveTx) error) error {
	ch := protocol.AssertionChain{}
	return ch.Tx(clo)
}

func (m *MockProtocol) Call(clo func(tx *protocol.ActiveTx) error) error {
	return clo(&protocol.ActiveTx{TxStatus: protocol.ReadOnlyTxStatus})
}

func (m *MockProtocol) SubscribeChainEvents(ctx context.Context, ch chan<- protocol.AssertionChainEvent) {
}

func (m *MockProtocol) TimeReference() util.TimeReference {
	return nil
}

func (m *MockProtocol) SubscribeChallengeEvents(ctx context.Context, ch chan<- protocol.ChallengeEvent) {
}

func (m *MockProtocol) AssertionBySequenceNum(tx *protocol.ActiveTx, seqNum protocol.AssertionSequenceNumber) (*protocol.Assertion, error) {
	args := m.Called(tx, seqNum)
	return args.Get(0).(*protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) ChallengeVertexByCommitHash(tx *protocol.ActiveTx, challengeHash protocol.ChallengeCommitHash, vertexHash protocol.VertexCommitHash) (*protocol.ChallengeVertex, error) {
	args := m.Called(tx, challengeHash, vertexHash)
	return args.Get(0).(*protocol.ChallengeVertex), args.Error(1)
}

func (m *MockProtocol) Completed(tx *protocol.ActiveTx) bool {
	args := m.Called(tx)
	return args.Get(0).(bool)
}

func (m *MockProtocol) HasConfirmedAboveSeqNumber(tx *protocol.ActiveTx, seqNum protocol.VertexSequenceNumber) (bool, error) {
	args := m.Called(tx, seqNum)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockProtocol) IsAtOneStepFork(
	ctx context.Context,
	tx *protocol.ActiveTx,
	challengeCommitHash protocol.ChallengeCommitHash,
	vertexCommit util.HistoryCommitment,
	vertexParentCommit util.HistoryCommitment,
) (bool, error) {
	args := m.Called(tx, challengeCommitHash, vertexCommit, vertexParentCommit)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockProtocol) ChallengeByCommitHash(tx *protocol.ActiveTx, commitHash protocol.ChallengeCommitHash) (*protocol.Challenge, error) {
	args := m.Called(tx, commitHash)
	return args.Get(0).(*protocol.Challenge), args.Error(1)
}

func (m *MockProtocol) LatestConfirmed(tx *protocol.ActiveTx) *protocol.Assertion {
	args := m.Called(tx)
	return args.Get(0).(*protocol.Assertion)
}

func (m *MockProtocol) CreateLeaf(tx *protocol.ActiveTx, prev *protocol.Assertion, commitment util.StateCommitment, staker common.Address) (*protocol.Assertion, error) {
	args := m.Called(tx, prev, commitment, staker)
	return args.Get(0).(*protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) ChallengePeriodLength(tx *protocol.ActiveTx) time.Duration {
	args := m.Called(tx)
	return args.Get(0).(time.Duration)
}

func (m *MockProtocol) NumAssertions(tx *protocol.ActiveTx) uint64 {
	args := m.Called(tx)
	return args.Get(0).(uint64)
}

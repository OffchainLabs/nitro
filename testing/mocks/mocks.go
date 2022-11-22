package mocks

import (
	"context"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	statemanager "github.com/OffchainLabs/new-rollup-exploration/state-manager"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) LatestHistoryCommitment(ctx context.Context) util.HistoryCommitment {
	args := m.Called(ctx)
	return args.Get(0).(util.HistoryCommitment)
}

func (m *MockStateManager) HasStateCommitment(ctx context.Context, commit protocol.StateCommitment) bool {
	args := m.Called(ctx, commit)
	return args.Bool(0)
}

func (m *MockStateManager) StateCommitmentAtHeight(ctx context.Context, height uint64) (util.HistoryCommitment, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) LatestStateCommitment(ctx context.Context) (util.HistoryCommitment, error) {
	args := m.Called(ctx)
	return args.Get(0).(util.HistoryCommitment), args.Error(1)
}

func (m *MockStateManager) SubscribeStateEvents(ctx context.Context, ch chan<- *statemanager.L2StateEvent) {
}

type MockProtocol struct {
	mock.Mock
}

func (m *MockProtocol) Inbox() *protocol.Inbox {
	args := m.Called()
	return args.Get(0).(*protocol.Inbox)
}

func (m *MockProtocol) Tx(clo func(tx *protocol.ActiveTx, pro protocol.OnChainProtocol) error) error {
	return clo(&protocol.ActiveTx{}, m)
}

func (m *MockProtocol) SubscribeChainEvents(ctx context.Context, ch chan<- protocol.AssertionChainEvent) {
}

func (m *MockProtocol) AssertionBySequenceNum(tx *protocol.ActiveTx, seqNum uint64) (*protocol.Assertion, error) {
	args := m.Called(tx, seqNum)
	return args.Get(0).(*protocol.Assertion), args.Error(1)
}

func (m *MockProtocol) LatestConfirmed(tx *protocol.ActiveTx) *protocol.Assertion {
	args := m.Called(tx)
	return args.Get(0).(*protocol.Assertion)
}

func (m *MockProtocol) CreateLeaf(tx *protocol.ActiveTx, prev *protocol.Assertion, commitment protocol.StateCommitment, staker common.Address) (*protocol.Assertion, error) {
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

func (m *MockProtocol) Call(clo func(*protocol.ActiveTx, protocol.OnChainProtocol) error) error {
	return clo(&protocol.ActiveTx{}, m)
}

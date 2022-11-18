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
	ret, ok := args.Get(0).(util.HistoryCommitment)
	if !ok {
		panic("not ok")
	}
	return ret
}

func (m *MockStateManager) HasStateRoot(ctx context.Context, stateRoot common.Hash) bool {
	args := m.Called(ctx, stateRoot)
	return args.Bool(0)
}

func (m *MockStateManager) StateCommitmentAtHeight(ctx context.Context, height uint64) (util.HistoryCommitment, error) {
	args := m.Called(ctx, height)
	ret, ok := args.Get(0).(util.HistoryCommitment)
	if !ok {
		panic("not ok")
	}
	return ret, args.Error(1)
}

func (m *MockStateManager) SubscribeStateEvents(ctx context.Context, ch chan<- *statemanager.L2StateEvent) {
}

type MockProtocol struct {
	mock.Mock
}

func (m *MockProtocol) Tx(clo func(*protocol.AssertionChain) error) error {
	args := m.Called(clo)
	return args.Error(0)
}

func (m *MockProtocol) SubscribeChainEvents(ctx context.Context, ch chan<- protocol.AssertionChainEvent) {
}

func (m *MockProtocol) LatestConfirmed() *protocol.Assertion {
	args := m.Called()
	ret, ok := args.Get(0).(*protocol.Assertion)
	if !ok {
		panic("not ok")
	}
	return ret
}

func (m *MockProtocol) CreateLeaf(prev *protocol.Assertion, commitment protocol.StateCommitment, staker common.Address) (*protocol.Assertion, error) {
	args := m.Called(prev, commitment, staker)
	ret, ok := args.Get(0).(*protocol.Assertion)
	if !ok {
		panic("not ok")
	}
	return ret, args.Error(1)
}

func (m *MockProtocol) ChallengePeriodLength() time.Duration {
	args := m.Called()
	dur, ok := args.Get(0).(time.Duration)
	if !ok {
		panic("not ok")
	}
	return dur
}

func (m *MockProtocol) AssertionBySequenceNumber(ctx context.Context, seqNum uint64) (*protocol.Assertion, error) {
	args := m.Called(ctx, seqNum)
	r, ok := args.Get(0).(*protocol.Assertion)
	if !ok {
		panic("not ok")
	}
	return r, args.Error(1)
}

func (m *MockProtocol) NumAssertions() uint64 {
	args := m.Called()
	r, ok := args.Get(0).(uint64)
	if !ok {
		panic("not ok")
	}
	return r
}

func (m *MockProtocol) Call(clo func(*protocol.AssertionChain) error) error {
	args := m.Called(clo)
	return args.Error(0)
}

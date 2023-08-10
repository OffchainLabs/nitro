// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPostAssertion(t *testing.T) {
	t.Run("new stake", func(t *testing.T) {
		ctx := context.Background()
		poster, chain, stateManager := setupPoster(t)
		_, creationInfo := setupAssertions(ctx, chain, stateManager, 10, func(int) bool { return false })
		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		chain.On(
			"ReadAssertionCreationInfo",
			ctx,
			latestValid,
		).Return(creationInfo[len(creationInfo)-1], nil)
		chain.On("IsStaked", ctx).Return(false, nil)
		stateManager.On("ExecutionStateAtMessageNumber", ctx, uint64(10)).Return(&protocol.ExecutionState{}, nil)
		assertion := &mocks.MockAssertion{}
		chain.On("NewStakeOnNewAssertion", ctx, creationInfo[len(creationInfo)-1], &protocol.ExecutionState{}).Return(assertion, nil)
		posted, err := poster.PostAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertion, posted)
	})
	t.Run("existing stake", func(t *testing.T) {
		ctx := context.Background()
		poster, chain, stateManager := setupPoster(t)
		_, creationInfo := setupAssertions(ctx, chain, stateManager, 10, func(int) bool { return false })
		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		chain.On(
			"ReadAssertionCreationInfo",
			ctx,
			latestValid,
		).Return(creationInfo[len(creationInfo)-1], nil)
		chain.On("IsStaked", ctx).Return(true, nil)
		stateManager.On("ExecutionStateAtMessageNumber", ctx, uint64(10)).Return(&protocol.ExecutionState{}, nil)
		assertion := &mocks.MockAssertion{}
		chain.On("StakeOnNewAssertion", ctx, creationInfo[len(creationInfo)-1], &protocol.ExecutionState{}).Return(assertion, nil)
		posted, err := poster.PostAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertion, posted)
	})
}

func Test_findLatestValidAssertion(t *testing.T) {
	ctx := context.Background()
	numAssertions := 10
	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
		poster, chain, stateManager := setupPoster(t)
		setupAssertions(ctx, chain, stateManager, numAssertions, func(int) bool { return false })
		chain.On("LatestConfirmed", ctx).Return(0, nil)
		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(0), latestValid)
	})
	t.Run("all are valid, latest one is picked", func(t *testing.T) {
		poster, chain, stateManager := setupPoster(t)
		setupAssertions(ctx, chain, stateManager, numAssertions, func(int) bool { return true })

		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(10), latestValid)
	})
	t.Run("latest valid is behind", func(t *testing.T) {
		poster, chain, stateManager := setupPoster(t)
		setupAssertions(ctx, chain, stateManager, numAssertions, func(i int) bool { return i <= 5 })
		chain.On("LatestConfirmed", ctx).Return(1, nil)

		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(5), latestValid)
	})
}

func mockId(x uint64) protocol.AssertionHash {
	return protocol.AssertionHash{Hash: common.BytesToHash([]byte(fmt.Sprintf("%d", x)))}
}

func setupAssertions(
	ctx context.Context,
	p *mocks.MockProtocol,
	s *mocks.MockStateManager,
	num int,
	validity func(int) bool,
) ([]protocol.Assertion, []*protocol.AssertionCreatedInfo) {
	if num == 0 {
		return make([]protocol.Assertion, 0), make([]*protocol.AssertionCreatedInfo, 0)
	}
	genesis := &mocks.MockAssertion{
		MockId:        mockId(0),
		MockPrevId:    mockId(0),
		MockHeight:    0,
		MockStateHash: common.Hash{},
		Prev:          option.None[*mocks.MockAssertion](),
	}
	p.On(
		"GetAssertion",
		ctx,
		mockId(uint64(0)),
	).Return(genesis, nil)
	assertions := []protocol.Assertion{genesis}
	creationInfo := make([]*protocol.AssertionCreatedInfo, 0)
	for i := 1; i <= num; i++ {
		mockHash := common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		mockAssertion := &mocks.MockAssertion{
			MockId:        mockId(uint64(i)),
			MockPrevId:    mockId(uint64(i - 1)),
			MockHeight:    uint64(i),
			MockStateHash: mockHash,
			Prev:          option.Some(assertions[i-1].(*mocks.MockAssertion)),
		}
		assertions = append(assertions, protocol.Assertion(mockAssertion))
		p.On(
			"GetAssertion",
			ctx,
			mockId(uint64(i)),
		).Return(protocol.Assertion(mockAssertion), nil)
		mockState := rollupgen.ExecutionState{
			MachineStatus: uint8(protocol.MachineStatusFinished),
			GlobalState: rollupgen.GlobalState(protocol.GoGlobalState{
				BlockHash: mockHash,
			}.AsSolidityStruct()),
		}
		mockAssertionCreationInfo := &protocol.AssertionCreatedInfo{
			AfterState:    mockState,
			InboxMaxCount: new(big.Int).SetUint64(uint64(i)),
		}
		creationInfo = append(creationInfo, mockAssertionCreationInfo)
		p.On(
			"ReadAssertionCreationInfo",
			ctx,
			mockId(uint64(i)),
		).Return(mockAssertionCreationInfo, nil)
		valid := validity(i)
		var arg error
		if !valid {
			arg = l2stateprovider.ErrNoExecutionState
		}
		s.On("ExecutionStateMsgCount", ctx, protocol.GoExecutionStateFromSolidity(mockState)).Return(uint64(i), arg)

		if i == 1 {
			var firstValid protocol.Assertion = genesis
			if valid {
				firstValid = protocol.Assertion(mockAssertion)
			}
			p.On("LatestConfirmed", ctx).Return(firstValid, nil)
		}
	}
	p.On("LatestConfirmed", ctx).Return(assertions[0], nil)
	p.On("LatestCreatedAssertion", ctx).Return(assertions[len(assertions)-1], nil)
	return assertions, creationInfo
}

func setupPoster(t *testing.T) (*Poster, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	chain := &mocks.MockProtocol{}
	ctx := context.Background()
	chain.On("CurrentChallengeManager", ctx).Return(&mocks.MockChallengeManager{}, nil)
	chain.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
	stateProvider := &mocks.MockStateManager{}
	_, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	p := &Poster{
		chain:        chain,
		stateManager: stateProvider,
	}
	return p, chain, stateProvider
}

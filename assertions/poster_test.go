package assertions

import (
	"context"
	"fmt"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

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
	return protocol.AssertionHash(common.BytesToHash([]byte(fmt.Sprintf("%d", x))))
}

func setupAssertions(
	ctx context.Context,
	p *mocks.MockProtocol,
	s *mocks.MockStateManager,
	num int,
	validity func(int) bool,
) []protocol.Assertion {
	if num == 0 {
		return make([]protocol.Assertion, 0)
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
			AfterState: mockState,
		}
		p.On(
			"ReadAssertionCreationInfo",
			ctx,
			mockId(uint64(i)),
		).Return(mockAssertionCreationInfo, nil)
		valid := validity(i)
		s.On("ExecutionStateMsgCount", ctx, protocol.GoExecutionStateFromSolidity(mockState)).Return(uint64(i), valid)

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
	return assertions
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

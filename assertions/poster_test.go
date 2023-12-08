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
	"github.com/OffchainLabs/bold/containers/threadsafe"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestPostAssertion(t *testing.T) {
	t.Run("new stake", func(t *testing.T) {
		ctx := context.Background()
		poster, chain, stateManager := setupPoster(t)
		_, creationInfo := setupAssertions(ctx, chain, stateManager, 10, func(int) bool { return false })
		info := creationInfo[len(creationInfo)-1]

		execState := protocol.GoExecutionStateFromSolidity(info.AfterState)
		stateManager.On("AgreesWithExecutionState", ctx, execState).Return(nil)
		assertion := &mocks.MockAssertion{}

		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)

		chain.On(
			"ReadAssertionCreationInfo",
			ctx,
			latestValid,
		).Return(info, nil)
		chain.On("IsStaked", ctx).Return(false, nil)
		stateManager.On("ExecutionStateAfterBatchCount", ctx, info.InboxMaxCount.Uint64()).Return(execState, nil)

		chain.On("NewStakeOnNewAssertion", ctx, info, execState).Return(assertion, nil)
		posted, err := poster.PostAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertion, posted.Unwrap())
	})
	t.Run("existing stake", func(t *testing.T) {
		ctx := context.Background()
		poster, chain, stateManager := setupPoster(t)
		_, creationInfo := setupAssertions(ctx, chain, stateManager, 10, func(int) bool { return false })
		info := creationInfo[len(creationInfo)-1]

		execState := protocol.GoExecutionStateFromSolidity(info.AfterState)
		stateManager.On("AgreesWithExecutionState", ctx, execState).Return(nil)
		assertion := &mocks.MockAssertion{}

		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)

		chain.On(
			"ReadAssertionCreationInfo",
			ctx,
			latestValid,
		).Return(info, nil)
		chain.On("IsStaked", ctx).Return(true, nil)

		stateManager.On("ExecutionStateAfterBatchCount", ctx, info.InboxMaxCount.Uint64()).Return(execState, nil)

		chain.On("StakeOnNewAssertion", ctx, info, execState).Return(assertion, nil)
		posted, err := poster.PostAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertion, posted.Unwrap())
	})
}

func Test_findLatestValidAssertion(t *testing.T) {
	ctx := context.Background()
	numAssertions := 10
	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
		poster, chain, stateManager := setupPoster(t)
		setupAssertions(ctx, chain, stateManager, numAssertions, func(int) bool { return false })
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

		latestValid, err := poster.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(5), latestValid)
	})
}

func Test_findLatestValidAssertionWithFork(t *testing.T) {
	ctx := context.Background()
	poster, chain, stateManager := setupPoster(t)
	setupAssertionsWithFork(ctx, chain, stateManager)
	latestValid, err := poster.findLatestValidAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, mockId(1), latestValid)
}

// Set-ups a chain with a fork at the 1st assertion
//
//	/-- 1 = Honest
//
// 0--
//
//	\-- 2 = Evil
//
// First honest assertion is posted with id 1 and prevId 0
// Then evil assertion is posted with id 2 and prevId 0
func setupAssertionsWithFork(ctx context.Context, chain *mocks.MockProtocol, stateManager *mocks.MockStateManager) {
	// Setup genesis
	genesisId := uint64(0)
	genesis := &mocks.MockAssertion{
		MockId:        mockId(genesisId),
		MockPrevId:    mockId(0),
		MockHeight:    0,
		MockStateHash: common.Hash{},
		Prev:          option.None[*mocks.MockAssertion](),
	}

	// Setup Valid Assertions
	validAssertionId := uint64(1)
	validHash := common.BytesToHash([]byte(fmt.Sprintf("%d", validAssertionId)))
	validAssertion := &mocks.MockAssertion{
		MockId:        mockId(validAssertionId),
		MockPrevId:    mockId(genesisId),
		MockHeight:    1,
		MockStateHash: validHash,
		Prev:          option.Some(genesis),
	}
	validState := rollupgen.ExecutionState{
		MachineStatus: uint8(protocol.MachineStatusFinished),
		GlobalState: rollupgen.GlobalState(protocol.GoGlobalState{
			BlockHash: validHash,
		}.AsSolidityStruct()),
	}
	validAssertionCreationInfo := &protocol.AssertionCreatedInfo{
		AfterState:    validState,
		InboxMaxCount: new(big.Int).SetUint64(uint64(1)),
	}
	chain.On(
		"ReadAssertionCreationInfo",
		ctx,
		mockId(validAssertionId),
	).Return(validAssertionCreationInfo, nil)
	stateManager.On("AgreesWithExecutionState", ctx, protocol.GoExecutionStateFromSolidity(validState)).Return(nil)

	// Setup Forked Invalid Assertions
	invalidAssertionId := uint64(2)
	invalidHash := common.BytesToHash([]byte(fmt.Sprintf("%d", invalidAssertionId)))
	invalidAssertion := &mocks.MockAssertion{
		MockId:        mockId(invalidAssertionId),
		MockPrevId:    mockId(genesisId),
		MockHeight:    1,
		MockStateHash: invalidHash,
		Prev:          option.Some(genesis),
	}
	invalidState := rollupgen.ExecutionState{
		MachineStatus: uint8(protocol.MachineStatusFinished),
		GlobalState: rollupgen.GlobalState(protocol.GoGlobalState{
			BlockHash: invalidHash,
		}.AsSolidityStruct()),
	}
	invalidAssertionCreationInfo := &protocol.AssertionCreatedInfo{
		AfterState:    invalidState,
		InboxMaxCount: new(big.Int).SetUint64(uint64(1)),
	}
	chain.On(
		"ReadAssertionCreationInfo",
		ctx,
		mockId(invalidAssertionId),
	).Return(invalidAssertionCreationInfo, nil)

	stateManager.On("AgreesWithExecutionState", ctx, protocol.GoExecutionStateFromSolidity(invalidState)).Return(errors.New("invalid"))

	chain.On("LatestConfirmed", ctx).Return(genesis, nil)
	chain.On("LatestCreatedAssertionHashes", ctx).Return([]protocol.AssertionHash{validAssertion.Id(), invalidAssertion.Id()}, nil)
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
		if !valid {
			s.On("AgreesWithExecutionState", ctx, protocol.GoExecutionStateFromSolidity(mockState)).Return(errors.New("invalid"))
		} else {
			s.On("AgreesWithExecutionState", ctx, protocol.GoExecutionStateFromSolidity(mockState)).Return(nil)
		}

	}
	var assertionHashes []protocol.AssertionHash
	for _, assertion := range assertions {
		assertionHashes = append(assertionHashes, assertion.Id())
	}
	p.On("LatestConfirmed", ctx).Return(genesis, nil)
	p.On("LatestCreatedAssertionHashes", ctx).Return(assertionHashes[1:], nil)
	return assertions, creationInfo
}

func setupPoster(t *testing.T) (*Manager, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	chain := &mocks.MockProtocol{}
	ctx := context.Background()
	chain.On("CurrentChallengeManager", ctx).Return(&mocks.MockChallengeManager{}, nil)
	chain.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
	stateProvider := &mocks.MockStateManager{}
	_, err := setup.ChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	p := &Manager{
		chain:               chain,
		stateManager:        stateProvider,
		submittedAssertions: threadsafe.NewSet[common.Hash](),
	}
	return p, chain, stateProvider
}

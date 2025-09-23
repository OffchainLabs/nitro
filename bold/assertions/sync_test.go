// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package assertions

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers/threadsafe"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/casttest"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

func Test_extractAssertionFromEvent(t *testing.T) {
	m := &Manager{}
	ctx := context.Background()

	t.Run("ignores empty hash", func(t *testing.T) {
		opt, err := m.extractAssertionFromEvent(ctx, &rollupgen.RollupUserLogicAssertionCreated{
			AssertionHash: common.Hash{},
		})
		require.NoError(t, err)
		require.Equal(t, true, opt.IsNone())
	})

	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)
	_ = setup

	bridgeBindings, err := mocksgen.NewBridgeStub(setup.Addrs.Bridge, setup.Backend)
	require.NoError(t, err)

	msgCount, err := bridgeBindings.SequencerMessageCount(setup.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())

	aliceChain := setup.Chains[0]
	genesisHash, err := setup.Chains[1].GenesisAssertionHash(ctx)
	require.NoError(t, err)
	genesisCreationInfo, err := setup.Chains[1].ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: genesisHash})
	require.NoError(t, err)

	stateManagerOpts := setup.StateManagerOpts
	aliceStateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)

	preState, err := aliceStateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	postState, err := aliceStateManager.ExecutionStateAfterPreviousState(ctx, 1, preState.GlobalState)
	require.NoError(t, err)
	assertion, err := aliceChain.NewStakeOnNewAssertion(
		ctx,
		genesisCreationInfo,
		postState,
	)
	require.NoError(t, err)

	t.Run("ignores genesis assertion", func(t *testing.T) {
		m.chain = aliceChain
		opt, err := m.extractAssertionFromEvent(ctx, &rollupgen.RollupUserLogicAssertionCreated{
			AssertionHash: genesisHash,
		})
		require.NoError(t, err)
		require.Equal(t, true, opt.IsNone())
	})
	t.Run("extracts assertion", func(t *testing.T) {
		m.chain = aliceChain
		opt, err := m.extractAssertionFromEvent(ctx, &rollupgen.RollupUserLogicAssertionCreated{
			AssertionHash: assertion.Id().Hash,
		})
		require.NoError(t, err)
		require.Equal(t, true, opt.IsSome())
		require.Equal(t, assertion.Id(), opt.Unwrap().AssertionHash)
	})
}

func Test_findCanonicalAssertionBranch(t *testing.T) {
	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     32,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agreesWithIds := map[uint64]*protocol.AssertionCreatedInfo{
		2: {
			ParentAssertionHash: numToHash(1),
			AssertionHash:       numToHash(2),
			AfterState:          numToState(2, t),
		},
		4: {
			ParentAssertionHash: numToHash(2),
			AssertionHash:       numToHash(4),
			AfterState:          numToState(4, t),
		},
		6: {
			ParentAssertionHash: numToHash(4),
			AssertionHash:       numToHash(6),
			AfterState:          numToState(6, t),
		},
	}
	provider := &mockStateProvider{
		agreesWith: agreesWithIds,
	}
	manager := &Manager{
		execProvider:                provider,
		chain:                       setup.Chains[0],
		observedCanonicalAssertions: make(chan protocol.AssertionHash),
		confirming:                  threadsafe.NewLruSet[protocol.AssertionHash](1000),
		assertionChainData: &assertionChainData{
			latestAgreedAssertion: numToAssertionHash(1),
			canonicalAssertions:   make(map[protocol.AssertionHash]*protocol.AssertionCreatedInfo),
		},
	}
	go func() {
		for {
			select {
			case <-manager.observedCanonicalAssertions:
			case <-ctx.Done():
				return
			}
		}
	}()
	require.NoError(t, manager.findCanonicalAssertionBranch(
		ctx,
		[]assertionAndParentCreationInfo{
			{
				parent: &protocol.AssertionCreatedInfo{
					InboxMaxCount: big.NewInt(2),
				},
				assertion: &protocol.AssertionCreatedInfo{
					ParentAssertionHash: numToHash(1),
					AssertionHash:       numToHash(2),
					AfterState:          numToState(2, t),
				},
			},
			{
				parent: &protocol.AssertionCreatedInfo{
					InboxMaxCount: big.NewInt(3),
				},
				assertion: &protocol.AssertionCreatedInfo{
					ParentAssertionHash: numToHash(1),
					AssertionHash:       numToHash(3),
					AfterState:          numToState(3, t),
				},
			},
			{
				parent: &protocol.AssertionCreatedInfo{
					InboxMaxCount: big.NewInt(4),
				},
				assertion: &protocol.AssertionCreatedInfo{
					ParentAssertionHash: numToHash(2),
					AssertionHash:       numToHash(4),
					AfterState:          numToState(4, t),
				},
			},
			{
				parent: &protocol.AssertionCreatedInfo{
					InboxMaxCount: big.NewInt(5),
				},
				assertion: &protocol.AssertionCreatedInfo{
					ParentAssertionHash: numToHash(2),
					AssertionHash:       numToHash(5),
					AfterState:          numToState(5, t),
				},
			},
			{
				parent: &protocol.AssertionCreatedInfo{
					InboxMaxCount: big.NewInt(6),
				},
				assertion: &protocol.AssertionCreatedInfo{
					ParentAssertionHash: numToHash(4),
					AssertionHash:       numToHash(6),
					AfterState:          numToState(6, t),
				},
			},
			{
				parent: &protocol.AssertionCreatedInfo{
					InboxMaxCount: big.NewInt(7),
				},
				assertion: &protocol.AssertionCreatedInfo{
					ParentAssertionHash: numToHash(4),
					AssertionHash:       numToHash(7),
					AfterState:          numToState(7, t),
				},
			},
		},
	))
	require.Equal(t, numToAssertionHash(6), manager.assertionChainData.latestAgreedAssertion)
	wanted := make(map[protocol.AssertionHash]bool)
	for id := range agreesWithIds {
		wanted[numToAssertionHash(casttest.ToInt(t, id))] = true
	}
	for assertionHash := range manager.assertionChainData.canonicalAssertions {
		require.Equal(t, true, wanted[assertionHash])
	}
}

func numToAssertionHash(i int) protocol.AssertionHash {
	return protocol.AssertionHash{Hash: common.BytesToHash([]byte(fmt.Sprintf("%d", i)))}
}

func numToHash(i int) protocol.AssertionHash {
	return protocol.AssertionHash{Hash: common.BytesToHash([]byte(fmt.Sprintf("%d", i)))}
}

func numToState(i int, t *testing.T) rollupgen.AssertionState {
	return rollupgen.AssertionState{
		GlobalState: rollupgen.GlobalState{
			U64Vals: [2]uint64{casttest.ToUint64(t, i), uint64(0)},
		},
	}
}

type mockStateProvider struct {
	agreesWith map[uint64]*protocol.AssertionCreatedInfo
}

func (m *mockStateProvider) ExecutionStateAfterPreviousState(
	ctx context.Context,
	maxInboxCount uint64,
	previousGlobalState protocol.GoGlobalState,
) (*protocol.ExecutionState, error) {
	agreement, ok := m.agreesWith[maxInboxCount]
	if !ok {
		return &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: common.BytesToHash([]byte("wrong")),
			},
		}, nil
	}
	return protocol.GoExecutionStateFromSolidity(agreement.AfterState), nil
}

func Test_respondToAnyInvalidAssertions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := &Manager{
		observedCanonicalAssertions: make(chan protocol.AssertionHash),
		submittedAssertions:         threadsafe.NewLruSet(1000, threadsafe.LruSetWithMetric[protocol.AssertionHash]("submittedAssertions")),
		confirming:                  threadsafe.NewLruSet[protocol.AssertionHash](1000),
		assertionChainData: &assertionChainData{
			latestAgreedAssertion: numToAssertionHash(1),
			canonicalAssertions:   make(map[protocol.AssertionHash]*protocol.AssertionCreatedInfo),
		},
	}
	go func() {
		for {
			select {
			case <-manager.observedCanonicalAssertions:
			case <-ctx.Done():
				return
			}
		}
	}()

	manager.assertionChainData.canonicalAssertions[numToAssertionHash(1)] = &protocol.AssertionCreatedInfo{}
	manager.assertionChainData.canonicalAssertions[numToAssertionHash(2)] = &protocol.AssertionCreatedInfo{
		ParentAssertionHash: numToHash(1),
	}
	manager.assertionChainData.canonicalAssertions[numToAssertionHash(4)] = &protocol.AssertionCreatedInfo{
		ParentAssertionHash: numToHash(2),
	}
	manager.assertionChainData.canonicalAssertions[numToAssertionHash(6)] = &protocol.AssertionCreatedInfo{
		ParentAssertionHash: numToHash(4),
	}

	t.Run("all assertions canonical no rivals posted", func(t *testing.T) {
		poster := &mockRivalPoster{}
		require.NoError(t, manager.respondToAnyInvalidAssertions(
			ctx,
			[]assertionAndParentCreationInfo{
				{
					parent: &protocol.AssertionCreatedInfo{},
					assertion: &protocol.AssertionCreatedInfo{
						ParentAssertionHash: numToHash(2),
						AssertionHash:       numToHash(4),
						AfterState:          numToState(4, t),
					},
				},
				{
					parent: &protocol.AssertionCreatedInfo{},
					assertion: &protocol.AssertionCreatedInfo{
						ParentAssertionHash: numToHash(4),
						AssertionHash:       numToHash(6),
						AfterState:          numToState(6, t),
					},
				},
			},
			poster,
		))
		require.Equal(t, uint64(0), manager.submittedRivalsCount)
	})
	t.Run("invalid assertions but no canonical parent in list", func(t *testing.T) {
		poster := &mockRivalPoster{}
		require.NoError(t, manager.respondToAnyInvalidAssertions(
			ctx,
			[]assertionAndParentCreationInfo{
				{
					parent: &protocol.AssertionCreatedInfo{},
					assertion: &protocol.AssertionCreatedInfo{
						ParentAssertionHash: numToHash(200),
						AssertionHash:       numToHash(400),
						AfterState:          numToState(400, t),
					},
				},
				{
					parent: &protocol.AssertionCreatedInfo{},
					assertion: &protocol.AssertionCreatedInfo{
						ParentAssertionHash: numToHash(400),
						AssertionHash:       numToHash(600),
						AfterState:          numToState(600, t),
					},
				},
			},
			poster,
		))
		require.Equal(t, uint64(0), manager.submittedRivalsCount)
	})
	t.Run("rivals posted successfully", func(t *testing.T) {
		poster := &mockRivalPoster{}
		require.NoError(t, manager.respondToAnyInvalidAssertions(
			ctx,
			[]assertionAndParentCreationInfo{
				// Some evil hashes which must be acted upon.
				{
					parent: &protocol.AssertionCreatedInfo{},
					assertion: &protocol.AssertionCreatedInfo{
						ParentAssertionHash: numToHash(2),
						AssertionHash:       numToHash(3),
						AfterState:          numToState(3, t),
					},
				},
				{
					parent: &protocol.AssertionCreatedInfo{},
					assertion: &protocol.AssertionCreatedInfo{
						ParentAssertionHash: numToHash(4),
						AssertionHash:       numToHash(5),
						AfterState:          numToState(5, t),
					},
				},
			},
			poster,
		))
		require.Equal(t, uint64(2), manager.submittedRivalsCount)
	})
}

type mockRivalPoster struct {
}

func (m *mockRivalPoster) maybePostRivalAssertionAndChallenge(
	ctx context.Context,
	args rivalPosterArgs,
) (*protocol.AssertionCreatedInfo, error) {
	if args.invalidAssertion.AssertionHash == numToHash(3) {
		return &protocol.AssertionCreatedInfo{
			AssertionHash: numToHash(300),
		}, nil
	}
	if args.invalidAssertion.AssertionHash == numToHash(5) {
		return &protocol.AssertionCreatedInfo{
			AssertionHash: numToHash(500),
		}, nil
	}
	panic("must have been able to post")
}

// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package assertions_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/assertions"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/testgen"
)

func TestPostAssertion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chainSetup, chalManager, assertionManager, stateManager := setupAssertionPosting(t)
	aliceChain := chainSetup.Chains[0]

	// Force the posting of a new sequencer batch.
	forceSequencerMessageBatchPosting(
		t, ctx, chainSetup.Accounts[0].TxOpts, chainSetup.Addrs.SequencerInbox, chainSetup.Backend,
	)

	// We have enabled auto-deposits for this test, so we expect that before starting the challenge manager,
	// Alice has an ERC20 deposit token balance of 0. After starting, we should expect a non-zero balance.
	rollup, err := rollupgen.NewRollupUserLogic(chainSetup.Addrs.Rollup, chainSetup.Backend)
	require.NoError(t, err)
	requiredStake, err := rollup.BaseStake(&bind.CallOpts{})
	require.NoError(t, err)
	stakeTokenAddr, err := rollup.StakeToken(&bind.CallOpts{})
	require.NoError(t, err)

	erc20, err := testgen.NewERC20Token(stakeTokenAddr, chainSetup.Backend)
	require.NoError(t, err)
	balance, err := erc20.BalanceOf(&bind.CallOpts{}, aliceChain.StakerAddress())
	require.NoError(t, err)
	require.True(t, big.NewInt(0).Cmp(balance) == 0)

	chalManager.Start(ctx)

	// Wait a little for the chain watcher to be ready.
	time.Sleep(time.Second)

	preState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 0, protocol.GoGlobalState{})
	require.NoError(t, err)
	postState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 1, preState.GlobalState)
	require.NoError(t, err)
	nextState, err := stateManager.ExecutionStateAfterPreviousState(ctx, 2, postState.GlobalState)
	require.NoError(t, err)

	// Expect a non-zero balance equal to the required stake after the challenge manager auto-deposited.
	balance, err = erc20.BalanceOf(&bind.CallOpts{}, aliceChain.StakerAddress())
	require.NoError(t, err)
	require.True(t, requiredStake.Cmp(balance) == 0)

	// Verify that alice can post an assertion correctly.
	posted, err := assertionManager.PostAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, true, posted.IsSome())

	creationInfo, err := aliceChain.ReadAssertionCreationInfo(ctx, posted.Unwrap().Id())
	require.NoError(t, err)
	require.Equal(t, postState, protocol.GoExecutionStateFromSolidity(creationInfo.AfterState))

	// Wait a little and advance the chain to allow the next assertion to be posted.
	time.Sleep(time.Second * 5)
	chainSetup.Backend.Commit()

	// Expect a zero ERC20 balance after the first staked assertion was posted.
	balance, err = erc20.BalanceOf(&bind.CallOpts{}, aliceChain.StakerAddress())
	require.NoError(t, err)
	require.True(t, big.NewInt(0).Cmp(balance) == 0)

	posted, err = assertionManager.PostAssertion(ctx)
	require.NoError(t, err)
	require.Equal(t, true, posted.IsSome())

	creationInfo, err = aliceChain.ReadAssertionCreationInfo(ctx, posted.Unwrap().Id())
	require.NoError(t, err)
	require.Equal(t, nextState, protocol.GoExecutionStateFromSolidity(creationInfo.AfterState))

	// Continue to expect a zero ERC20 balance after the second assertion was posted, as no new
	// stake was expected for the validator.
	time.Sleep(time.Second * 5)
	chainSetup.Backend.Commit()

	balance, err = erc20.BalanceOf(&bind.CallOpts{}, chainSetup.Accounts[0].TxOpts.From)
	require.NoError(t, err)
	require.True(t, big.NewInt(0).Cmp(balance) == 0)

	// We then filter all the transactions to the staken address from the validator and expect
	// there was only a single deposit event (a transfer event with from set to 0x0).
	it, err := erc20.FilterTransfer(
		&bind.FilterOpts{
			Start: 0,
			End:   nil,
		},
		[]common.Address{{}},
		[]common.Address{aliceChain.StakerAddress()},
	)
	require.NoError(t, err)
	defer func() {
		if err = it.Close(); err != nil {
			t.Error(err)
		}
	}()
	totalTransfers := 0
	for it.Next() {
		totalTransfers++
	}
	require.Equal(t, 1, totalTransfers, "Expected only one deposit event by the staker")
}

func setupAssertionPosting(t *testing.T) (*setup.ChainSetup, *challengemanager.Manager, *assertions.Manager, *stateprovider.L2StateBackend) {
	setup, err := setup.ChainsWithEdgeChallengeManager(
		setup.WithMockOneStepProver(),
		setup.WithAutoDeposit(),
		setup.WithChallengeTestingOpts(
			challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
				BlockChallengeHeight:     64,
				BigStepChallengeHeight:   32,
				SmallStepChallengeHeight: 32,
			}),
		),
	)
	require.NoError(t, err)
	bridgeBindings, err := mocksgen.NewBridgeStub(setup.Addrs.Bridge, setup.Backend)
	require.NoError(t, err)
	msgCount, err := bridgeBindings.SequencerMessageCount(setup.Chains[0].GetCallOptsWithDesiredRpcHeadBlockNumber(&bind.CallOpts{}))
	require.NoError(t, err)
	require.Equal(t, uint64(1), msgCount.Uint64())
	aliceChain := setup.Chains[0]
	stateManagerOpts := setup.StateManagerOpts
	stateManagerOpts = append(
		stateManagerOpts,
		stateprovider.WithNumBatchesRead(5),
	)
	stateManager, err := stateprovider.NewForSimpleMachine(t, stateManagerOpts...)
	require.NoError(t, err)
	// Set MinimumGapToBlockCreationTime as 1 second to verify that a new assertion is only posted after 1 sec has passed
	// from parent assertion creation. This will make the test run for ~19 seconds as the parent assertion time is
	// ~18 seconds in the future
	assertionManager, err := assertions.NewManager(
		aliceChain,
		stateManager,
		"alice",
		types.DefensiveMode,
		assertions.WithPollingInterval(time.Millisecond*200),
		assertions.WithAverageBlockCreationTime(time.Second),
		assertions.WithMinimumGapToParentAssertion(time.Second),
	)
	require.NoError(t, err)
	chalManager, err := challengemanager.NewChallengeStack(
		aliceChain,
		stateManager,
		challengemanager.StackWithMode(types.DefensiveMode),
		challengemanager.StackWithName("alice"),
		challengemanager.OverrideAssertionManager(assertionManager),
	)
	require.NoError(t, err)
	return setup, chalManager, assertionManager, stateManager
}

func forceSequencerMessageBatchPosting(
	t *testing.T,
	ctx context.Context,
	sequencerOpts *bind.TransactOpts,
	seqInboxAddr common.Address,
	backend *setup.SimulatedBackendWrapper,
) {
	batchCompressedBytes := hexutil.MustDecode("0x94643ec208c5558027fa768281f28aa273f01537942cd58cdd9c17e97e30281f")
	message := append([]byte{0}, batchCompressedBytes...)
	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, backend)
	require.NoError(t, err)
	tx, err := seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(
		sequencerOpts, seqNum, message, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0),
	)
	require.NoError(t, err)
	require.NoError(t, challenge_testing.WaitForTx(ctx, backend, tx))
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, gethtypes.ReceiptStatusSuccessful, receipt.Status)
}

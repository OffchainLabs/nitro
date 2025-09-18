package endtoend

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/chain-abstraction/sol-implementation"
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/runtime"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/endtoend/backend"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

func TestEndToEnd_DelegatedStaking(t *testing.T) {
	neutralCtx, neutralCancel := context.WithCancel(context.Background())
	defer neutralCancel()
	evilCtx, evilCancel := context.WithCancel(context.Background())
	defer evilCancel()
	honestCtx, honestCancel := context.WithCancel(context.Background())
	defer honestCancel()

	protocolCfg := defaultProtocolParams()
	protocolCfg.challengePeriodBlocks = 25
	timeCfg := defaultTimeParams()
	timeCfg.blockTime = time.Second
	inboxCfg := defaultInboxParams()

	challengeTestingOpts := []challenge_testing.Opt{
		challenge_testing.WithConfirmPeriodBlocks(protocolCfg.challengePeriodBlocks),
		challenge_testing.WithLayerZeroHeights(&protocolCfg.layerZeroHeights),
		challenge_testing.WithNumBigStepLevels(protocolCfg.numBigStepLevels),
	}
	deployOpts := []setup.Opt{
		setup.WithMockBridge(),
		setup.WithMockOneStepProver(),
		setup.WithNumAccounts(10),
		setup.WithChallengeTestingOpts(challengeTestingOpts...),
	}

	simBackend, err := backend.NewSimulated(timeCfg.blockTime, deployOpts...)
	require.NoError(t, err)
	bk := simBackend

	rollupAddr, err := bk.DeployRollup(neutralCtx, challengeTestingOpts...)
	require.NoError(t, err)

	require.NoError(t, bk.Start(neutralCtx))

	accounts := bk.Accounts()
	bk.Commit()

	rollupUserBindings, err := rollupgen.NewRollupUserLogic(rollupAddr.Rollup, bk.Client())
	require.NoError(t, err)
	bridgeAddr, err := rollupUserBindings.Bridge(&bind.CallOpts{})
	require.NoError(t, err)
	dataHash := common.Hash{1}
	enqueueSequencerMessageAsExecutor(
		t, accounts[0], rollupAddr.UpgradeExecutor, bk.Client(), bridgeAddr, seqMessage{
			dataHash:                 dataHash,
			afterDelayedMessagesRead: big.NewInt(1),
			prevMessageCount:         big.NewInt(1),
			newMessageCount:          big.NewInt(2),
		},
	)

	baseStateManagerOpts := []stateprovider.Opt{
		stateprovider.WithNumBatchesRead(inboxCfg.numBatchesPosted),
		stateprovider.WithLayerZeroHeights(&protocolCfg.layerZeroHeights, protocolCfg.numBigStepLevels),
	}
	honestStateManager, err := stateprovider.NewForSimpleMachine(t, baseStateManagerOpts...)
	require.NoError(t, err)

	shp := &simpleHeaderProvider{b: bk, chs: make([]chan<- *gethtypes.Header, 0)}
	shp.Start(neutralCtx)

	baseStackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithMode(types.MakeMode),
		challengemanager.StackWithPollingInterval(timeCfg.assertionScanningInterval),
		challengemanager.StackWithPostingInterval(timeCfg.assertionPostingInterval),
		challengemanager.StackWithAverageBlockCreationTime(timeCfg.blockTime),
		challengemanager.StackWithConfirmationInterval(timeCfg.assertionConfirmationAttemptInterval),
		challengemanager.StackWithMinimumGapToParentAssertion(0),
		challengemanager.StackWithHeaderProvider(shp),
		challengemanager.StackWithDelegatedStaking(), // Enable delegated staking.
		challengemanager.StackWithoutAutoDeposit(),
	}

	name := "honest"

	// Ensure the honest validator is a generated account that has no erc20 token balance,
	// but has some ETH to pay for gas costs of BoLD. We ensure that the honest validator
	// is not initially staked, and that the actual address that will be funding the honest
	// validator has enough funds.
	fundsCustodianOpts := accounts[1] // The 1st and 2nd accounts should be the funds' custodians.
	evilFundsCustodianOpts := accounts[2]
	honestTxOpts := accounts[len(accounts)-1]
	evilTxOpts := accounts[len(accounts)-2]

	//nolint:gocritic
	honestOpts := append(
		baseStackOpts,
		challengemanager.StackWithName(name),
	)
	// Ensure the funds custodian is the withdrawal address for the honest validator.
	honestChain := setupAssertionChain(
		t,
		honestCtx,
		bk.Client(),
		rollupAddr.Rollup,
		honestTxOpts,
		solimpl.WithCustomWithdrawalAddress(fundsCustodianOpts.From),
	)

	machineDivergenceStep := uint64(1)
	assertionDivergenceHeight := uint64(1)
	assertionBlockHeightDifference := int64(1)

	//nolint:gocritic
	evilStateManagerOpts := append(
		baseStateManagerOpts,
		stateprovider.WithMachineDivergenceStep(machineDivergenceStep),
		stateprovider.WithBlockDivergenceHeight(assertionDivergenceHeight),
		stateprovider.WithDivergentBlockHeightOffset(assertionBlockHeightDifference),
	)
	evilStateManager, err := stateprovider.NewForSimpleMachine(t, evilStateManagerOpts...)
	require.NoError(t, err)

	//nolint:gocritic
	evilOpts := append(
		baseStackOpts,
		challengemanager.StackWithName("evil"),
	)
	evilChain := setupAssertionChain(
		t,
		evilCtx,
		bk.Client(),
		rollupAddr.Rollup,
		evilTxOpts,
		solimpl.WithCustomWithdrawalAddress(evilFundsCustodianOpts.From),
	)

	// Ensure that both validators are not yet staked.
	isStaked, err := honestChain.IsStaked(honestCtx)
	require.NoError(t, err)
	require.False(t, isStaked)
	isStaked, err = evilChain.IsStaked(evilCtx)
	require.NoError(t, err)
	require.False(t, isStaked)

	chalManagerAddr := honestChain.SpecChallengeManager().Address()
	cmBindings, err := challengeV2gen.NewEdgeChallengeManager(chalManagerAddr, bk.Client())
	require.NoError(t, err)
	stakeToken, err := cmBindings.StakeToken(&bind.CallOpts{})
	require.NoError(t, err)
	requiredStake, err := honestChain.RollupCore().BaseStake(&bind.CallOpts{})
	require.NoError(t, err)

	tokenBindings, err := mocksgen.NewTestWETH9(stakeToken, bk.Client())
	require.NoError(t, err)

	balCustodian, err := tokenBindings.BalanceOf(&bind.CallOpts{}, fundsCustodianOpts.From)
	require.NoError(t, err)
	require.True(t, balCustodian.Cmp(requiredStake) >= 0) // Ensure funds custodian DOES have enough stake token balance.
	balEvilCustodian, err := tokenBindings.BalanceOf(&bind.CallOpts{}, evilFundsCustodianOpts.From)
	require.NoError(t, err)
	require.True(t, balEvilCustodian.Cmp(requiredStake) >= 0) // Ensure funds custodian DOES have enough stake token balance.

	honestManager, err := challengemanager.NewChallengeStack(honestChain, honestStateManager, honestOpts...)
	require.NoError(t, err)
	_ = honestManager

	evilManager, err := challengemanager.NewChallengeStack(evilChain, evilStateManager, evilOpts...)
	require.NoError(t, err)
	_ = evilManager

	honestManager.Start(honestCtx)
	evilManager.Start(evilCtx)

	// Next, the custodians add deposits.
	// Waits until the validators are staked with a value of 0 before adding the deposit.
	var isStakedWithZero bool
	for honestCtx.Err() == nil && !isStakedWithZero {
		isStaked, err = honestChain.IsStaked(honestCtx)
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		if isStaked {
			isStakedWithZero = true
		}
	}
	isStakedWithZero = false
	for evilCtx.Err() == nil && !isStakedWithZero {
		isStaked, err = evilChain.IsStaked(evilCtx)
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		if isStaked {
			isStakedWithZero = true
		}
	}

	// Now, adds the deposit.
	rollupUserLogic, err := rollupgen.NewRollupUserLogic(rollupAddr.Rollup, bk.Client())
	require.NoError(t, err)
	tx, err := rollupUserLogic.AddToDeposit(fundsCustodianOpts, honestTxOpts.From, fundsCustodianOpts.From, balCustodian)
	require.NoError(t, err)
	_, err = bind.WaitMined(honestCtx, bk.Client(), tx)
	require.NoError(t, err)

	tx, err = rollupUserLogic.AddToDeposit(evilFundsCustodianOpts, evilTxOpts.From, evilFundsCustodianOpts.From, balEvilCustodian)
	require.NoError(t, err)
	_, err = bind.WaitMined(evilCtx, bk.Client(), tx)
	require.NoError(t, err)

	t.Log("Delegated validators now have a deposit balance")

	t.Run("expects honest validator to win challenge", func(t *testing.T) {
		chainId, err := bk.Client().ChainID(honestCtx)
		require.NoError(t, err)
		// Wait until a challenged assertion is confirmed by time.
		var confirmed bool
		for neutralCtx.Err() == nil && !confirmed {
			var i *rollupgen.RollupCoreAssertionConfirmedIterator
			i, err = retry.UntilSucceeds(neutralCtx, func() (*rollupgen.RollupCoreAssertionConfirmedIterator, error) {
				return honestChain.RollupCore().FilterAssertionConfirmed(nil, nil)
			})
			require.NoError(t, err)
			for i.Next() {
				creationInfo, err2 := evilChain.ReadAssertionCreationInfo(evilCtx, protocol.AssertionHash{Hash: i.Event.AssertionHash})
				require.NoError(t, err2)

				var parent rollupgen.AssertionNode
				parent, err = retry.UntilSucceeds(neutralCtx, func() (rollupgen.AssertionNode, error) {
					return honestChain.RollupCore().GetAssertion(&bind.CallOpts{Context: neutralCtx}, creationInfo.ParentAssertionHash.Hash)
				})
				require.NoError(t, err)

				tx, _, err2 := bk.Client().TransactionByHash(neutralCtx, creationInfo.TransactionHash)
				require.NoError(t, err2)
				sender, err2 := gethtypes.Sender(gethtypes.NewCancunSigner(chainId), tx)
				require.NoError(t, err2)
				honestConfirmed := sender == honestTxOpts.From

				isChallengeChild := parent.FirstChildBlock > 0 && parent.SecondChildBlock > 0
				if !isChallengeChild {
					// Assertion must be a challenge child.
					continue
				}
				// We expect the honest party to have confirmed it.
				if !honestConfirmed {
					t.Fatal("Evil party confirmed the assertion by challenge win")
				}
				confirmed = true
				break
			}
			time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		}
		// Once the honest, claimed assertion in the challenge is confirmed by time, we win the test.
		t.Log("Assertion was confirmed by time")
	})
}

// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

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
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/runtime"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/endtoend/backend"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/challengeV2gen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

// This test ensures a challenge can complete even if the honest validator crashes mid-challenge.
// We cancel the honest validator's context after it opens the first subchallenge and prove that it
// can restart and carry things out to confirm the honest, claimed assertion in the challenge.
func TestEndToEnd_HonestValidatorCrashes(t *testing.T) {
	t.Skip("Flakey in CI, needs investigation")
	neutralCtx, neutralCancel := context.WithCancel(context.Background())
	defer neutralCancel()
	evilCtx, evilCancel := context.WithCancel(context.Background())
	defer evilCancel()
	honestCtx, honestCancel := context.WithCancel(context.Background())
	defer honestCancel()

	protocolCfg := defaultProtocolParams()
	protocolCfg.challengePeriodBlocks = 40
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
		setup.WithNumAccounts(5),
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
	}

	name := "honest"
	txOpts := accounts[1]
	//nolint:gocritic
	honestOpts := append(
		baseStackOpts,
		challengemanager.StackWithName(name),
	)
	honestChain := setupAssertionChain(t, honestCtx, bk.Client(), rollupAddr.Rollup, txOpts)
	honestManager, err := challengemanager.NewChallengeStack(honestChain, honestStateManager, honestOpts...)
	require.NoError(t, err)

	totalOpcodes := totalWasmOpcodes(&protocolCfg.layerZeroHeights, protocolCfg.numBigStepLevels)
	t.Logf("Total wasm opcodes in test: %d", totalOpcodes)

	assertionDivergenceHeight := uint64(1)
	assertionBlockHeightDifference := int64(1)

	machineDivergenceStep := uint64(1)
	//nolint:gocritic
	evilStateManagerOpts := append(
		baseStateManagerOpts,
		stateprovider.WithMachineDivergenceStep(machineDivergenceStep),
		stateprovider.WithBlockDivergenceHeight(assertionDivergenceHeight),
		stateprovider.WithDivergentBlockHeightOffset(assertionBlockHeightDifference),
	)
	evilStateManager, err := stateprovider.NewForSimpleMachine(t, evilStateManagerOpts...)
	require.NoError(t, err)

	// Honest validator has index 1 in the accounts slice, as 0 is admin, so
	// evil ones should start at 2.
	evilTxOpts := accounts[2]
	//nolint:gocritic
	evilOpts := append(
		baseStackOpts,
		challengemanager.StackWithName("evil"),
	)
	evilChain := setupAssertionChain(t, evilCtx, bk.Client(), rollupAddr.Rollup, evilTxOpts)
	evilManager, err := challengemanager.NewChallengeStack(evilChain, evilStateManager, evilOpts...)
	require.NoError(t, err)

	chalManagerAddr := honestChain.SpecChallengeManager().Address()
	cmBindings, err := challengeV2gen.NewEdgeChallengeManager(chalManagerAddr, bk.Client())
	require.NoError(t, err)

	honestManager.Start(honestCtx)
	evilManager.Start(evilCtx)

	t.Run("crashes mid-challenge and recovers to complete it", func(t *testing.T) {
		// We will listen for the first subchallenge edge created by the honest validator to appear, and then
		// we will cancel the honest validator context. We will then wait for a bit, then restart the honest
		// validator and we should expect the honest assertion is still confirmed by time.
		// No more edges will be added here, so we then scrape all the edges added to the challenge.
		// We await until all the essential root edges are also confirmed by time.
		chainId, err2 := bk.Client().ChainID(neutralCtx)
		require.NoError(t, err2)
		var foundSubchalEdge bool
		for neutralCtx.Err() == nil && !foundSubchalEdge {
			it, err3 := cmBindings.FilterEdgeAdded(nil, nil, nil, nil)
			require.NoError(t, err3)
			for it.Next() {
				txHash := it.Event.Raw.TxHash
				tx, _, err3 := bk.Client().TransactionByHash(neutralCtx, txHash)
				require.NoError(t, err3)
				sender, err3 := gethtypes.Sender(gethtypes.NewCancunSigner(chainId), tx)
				require.NoError(t, err3)
				if sender != txOpts.From {
					continue
				}
				if it.Event.Level > 0 {
					foundSubchalEdge = true
					t.Log("Honest validator made a subchallenge")
					break // The honest validator made a subchallenge.
				}
			}
			time.Sleep(500 * time.Millisecond) // Don't spam the backend.
		}
		// Cancel the honest context.
		honestCancel()
		t.Log("Honest context has been canceled")

		// We then restart the honest validator after a few seconds of wait time.
		time.Sleep(time.Second * 3)

		honestCtx, honestCancel = context.WithCancel(context.Background())
		honestChain := setupAssertionChain(t, honestCtx, bk.Client(), rollupAddr.Rollup, txOpts)
		honestManager, err := challengemanager.NewChallengeStack(honestChain, honestStateManager, honestOpts...)
		require.NoError(t, err)

		honestManager.Start(honestCtx)

		rc, err2 := rollupgen.NewRollupCore(rollupAddr.Rollup, bk.Client())
		require.NoError(t, err2)

		// Wait until a challenged assertion is confirmed by time.
		var confirmed bool
		for neutralCtx.Err() == nil && !confirmed {
			var i *rollupgen.RollupCoreAssertionConfirmedIterator
			i, err = retry.UntilSucceeds(neutralCtx, func() (*rollupgen.RollupCoreAssertionConfirmedIterator, error) {
				return rc.FilterAssertionConfirmed(nil, nil)
			})
			require.NoError(t, err)
			for i.Next() {
				creationInfo, err2 := evilChain.ReadAssertionCreationInfo(evilCtx, protocol.AssertionHash{Hash: i.Event.AssertionHash})
				require.NoError(t, err2)

				var parent rollupgen.AssertionNode
				parent, err = retry.UntilSucceeds(neutralCtx, func() (rollupgen.AssertionNode, error) {
					return rc.GetAssertion(&bind.CallOpts{Context: neutralCtx}, creationInfo.ParentAssertionHash.Hash)
				})
				require.NoError(t, err)

				tx, _, err2 := bk.Client().TransactionByHash(neutralCtx, creationInfo.TransactionHash)
				require.NoError(t, err2)
				sender, err2 := gethtypes.Sender(gethtypes.NewCancunSigner(chainId), tx)
				require.NoError(t, err2)
				honestConfirmed := sender == txOpts.From

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
		// Once the honest, claimed assertion in the challenge is confirmed by time, we
		// then continue the test.
		t.Log("Assertion was confirmed by time")
		honestCancel()
	})
	// This test ensures that an honest validator can crash after a challenge has completed, can resync
	// the completed challenge and continue playing the game until all essential edges are confirmed.
	// This is to ensure that even if a challenge is completed, we can still resync it and continue
	// playing for the sake of refunding honest stakes.
	t.Run(
		"crashes once challenged assertion is confirmed and restarts to confirm essential edges",
		func(t *testing.T) {
			// We restart the honest validator after a few seconds of wait time.
			time.Sleep(time.Second * 5)

			ctx := context.Background()
			honestChain := setupAssertionChain(t, ctx, bk.Client(), rollupAddr.Rollup, txOpts)
			honestManager, err := challengemanager.NewChallengeStack(honestChain, honestStateManager, honestOpts...)
			require.NoError(t, err)

			honestManager.Start(ctx)

			t.Log("Restarted honest validator to continue playing game after challenge has finished")

			// We then expect that all essential root edges created by the honest validator are confirmed by time.
			// Scrape all the honest edges onchain (the ones made by the honest address).
			// Check if the edges that have claim id != None are confirmed (those are essential root edges)
			// and also check one step edges from honest party are confirmed.
			honestEssentialRootIds := make(map[common.Hash]bool, 0)
			chainId, err := bk.Client().ChainID(neutralCtx)
			require.NoError(t, err)
			it, err := cmBindings.FilterEdgeAdded(nil, nil, nil, nil)
			require.NoError(t, err)
			for it.Next() {
				txHash := it.Event.Raw.TxHash
				tx, _, err2 := bk.Client().TransactionByHash(neutralCtx, txHash)
				require.NoError(t, err2)
				sender, err2 := gethtypes.Sender(gethtypes.NewCancunSigner(chainId), tx)
				require.NoError(t, err2)
				if sender != txOpts.From {
					continue
				}
				// Skip edges that are not essential roots (skip the top-level edge).
				if it.Event.ClaimId == (common.Hash{}) || it.Event.Level == 0 {
					continue
				}
				honestEssentialRootIds[it.Event.EdgeId] = false
			}
			// Wait until all of the honest essential root ids are confirmed.
			startBlk, err := bk.Client().HeaderU64(neutralCtx)
			require.NoError(t, err)
			chalPeriodBlocks, err := cmBindings.ChallengePeriodBlocks(&bind.CallOpts{})
			require.NoError(t, err)
			totalPeriod := chalPeriodBlocks * uint64(len(honestEssentialRootIds))
			confirmedCount := 0
			_ = totalPeriod
			_ = startBlk
			for confirmedCount < len(honestEssentialRootIds) {
				latestBlk, err2 := bk.Client().HeaderU64(neutralCtx)
				require.NoError(t, err2)
				numBlocksElapsed := latestBlk - startBlk
				if numBlocksElapsed > totalPeriod {
					t.Fatalf("%d blocks have passed without essential edges being confirmed", numBlocksElapsed)
				}
				for k, markedConfirmed := range honestEssentialRootIds {
					edge, err2 := cmBindings.GetEdge(&bind.CallOpts{}, k)
					require.NoError(t, err2)
					if edge.Status == 1 && !markedConfirmed {
						confirmedCount += 1
						honestEssentialRootIds[k] = true
						t.Logf("Confirmed %d honest essential edges, got edge at level %d", confirmedCount, edge.Level)
					}
				}
				time.Sleep(500 * time.Millisecond) // Don't spam the backend.
			}
		})
}

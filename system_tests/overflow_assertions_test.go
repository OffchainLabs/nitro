// Copyright 2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	protocol "github.com/offchainlabs/bold/chain-abstraction"
	challengemanager "github.com/offchainlabs/bold/challenge-manager"
	modes "github.com/offchainlabs/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/bold/solgen/go/mocksgen"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/bold/testing/setup"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestOverflowAssertions(t *testing.T) {
	// Get a simulated geth backend running.
	//
	// Create enough messages in batches to overflow the block level challenge
	// height. (height == 32, messages = 45)
	//
	// Start the challenge manager with a minimumAssertionPeriod of 7 and make
	// sure that it posts overflow-assertions right away instead of waiting for
	// the 7 blocks to pass.
	goodDir, err := os.MkdirTemp("", "good_*")
	Require(t, err)
	t.Cleanup(func() {
		Require(t, os.RemoveAll(goodDir))
	})
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	// This is important to show that overflow assertions don't wait.
	minAssertionBlocks := int64(7)
	ownerBal := big.NewInt(params.Ether)
	ownerBal.Mul(ownerBal, big.NewInt(1_000_000))
	l2info.GenerateGenesisAccount("Owner", ownerBal)
	sconf := setup.RollupStackConfig{
		UseMockBridge:          false,
		UseMockOneStepProver:   false,
		MinimumAssertionPeriod: minAssertionBlocks,
	}

	_, l2node, _, _, l1info, _, l1client, l1stack, assertionChain, _ := createTestNodeOnL1ForBoldProtocol(t, ctx, true, nil, l2chainConfig, nil, sconf, l2info)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()

	// Make sure we shut down test functionality before the rest of the node
	ctx, cancelCtx = context.WithCancel(ctx)
	defer cancelCtx()

	go keepChainMoving(t, ctx, l1info, l1client)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)

	valCfg := valnode.TestValidationConfig
	valCfg.UseJit = false
	_, valStack := createTestValidationNode(t, ctx, &valCfg)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.Execution,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)

	blockValidator, err := staker.NewBlockValidator(
		stateless,
		l2node.InboxTracker,
		l2node.TxStreamer,
		StaticFetcherFrom(t, &blockValidatorConfig),
		nil,
	)
	Require(t, err)
	Require(t, blockValidator.Initialize(ctx))
	Require(t, blockValidator.Start(ctx))

	stateManager, err := bold.NewBOLDStateProvider(
		blockValidator,
		stateless,
		l2stateprovider.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          "good",
			MachineLeavesCachePath: goodDir,
			CheckBatchFinality:     false,
		},
		goodDir,
	)
	Require(t, err)

	Require(t, l2node.Start(ctx))

	l2info.GenerateAccount("Destination")
	sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	honestSeqInbox := l1info.GetAddress("SequencerInbox")
	honestSeqInboxBinding, err := bridgegen.NewSequencerInbox(honestSeqInbox, l1client)
	Require(t, err)

	// Post batches to the honest and inbox.
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err)

	honestUpgradeExec, err := mocksgen.NewUpgradeExecutorMock(l1info.GetAddress("UpgradeExecutor"), l1client)
	Require(t, err)
	data, err := seqInboxABI.Pack(
		"setIsBatchPoster",
		sequencerTxOpts.From,
		true,
	)
	Require(t, err)
	honestRollupOwnerOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = honestUpgradeExec.ExecuteCall(&honestRollupOwnerOpts, honestSeqInbox, data)
	Require(t, err)

	// Post enough messages (45 across 2 batches) to overflow the block level
	// challenge height (32).
	totalMessagesPosted := int64(0)
	numMessagesPerBatch := int64(32)
	divergeAt := int64(-1)
	makeBoldBatch(t, l2node, l2info, l1client, &sequencerTxOpts, honestSeqInboxBinding, honestSeqInbox, numMessagesPerBatch, divergeAt)
	totalMessagesPosted += numMessagesPerBatch

	numMessagesPerBatch = int64(13)
	makeBoldBatch(t, l2node, l2info, l1client, &sequencerTxOpts, honestSeqInboxBinding, honestSeqInbox, numMessagesPerBatch, divergeAt)
	totalMessagesPosted += numMessagesPerBatch

	bc, err := l2node.InboxTracker.GetBatchCount()
	Require(t, err)
	msgs, err := l2node.InboxTracker.GetBatchMessageCount(bc - 1)
	Require(t, err)

	t.Logf("Node batch count %d, msgs %d", bc, msgs)

	// Wait for the node to catch up.
	nodeExec, ok := l2node.Execution.(*gethexec.ExecutionNode)
	if !ok {
		Fatal(t, "not geth execution node")
	}
	for {
		latest := nodeExec.Backend.APIBackend().CurrentHeader()
		isCaughtUp := latest.Number.Uint64() == uint64(totalMessagesPosted)
		if isCaughtUp {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}

	bridgeBinding, err := bridgegen.NewBridge(l1info.GetAddress("Bridge"), l1client)
	Require(t, err)
	totalBatchesBig, err := bridgeBinding.SequencerMessageCount(&bind.CallOpts{Context: ctx})
	Require(t, err)
	totalBatches := totalBatchesBig.Uint64()

	// Wait until the validator has validated the batches.
	for {
		lastInfo, err := blockValidator.ReadLastValidatedInfo()
		if lastInfo == nil || err != nil {
			continue
		}
		t.Log("Batch", lastInfo.GlobalState.Batch, "Total", totalBatches-1)
		if lastInfo.GlobalState.Batch >= totalBatches-1 {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}

	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateManager,
		stateManager,
		stateManager,
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		stateManager,
		nil, // Api db
	)

	stackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithName("default"),
		challengemanager.StackWithMode(modes.MakeMode),
		challengemanager.StackWithPostingInterval(time.Second),
		challengemanager.StackWithPollingInterval(time.Millisecond * 500),
		challengemanager.StackWithAverageBlockCreationTime(time.Second),
		challengemanager.StackWithMinimumGapToParentAssertion(0),
	}

	manager, err := challengemanager.NewChallengeStack(
		assertionChain,
		provider,
		stackOpts...,
	)
	Require(t, err)
	manager.Start(ctx)

	rollup, err := rollupgen.NewRollupUserLogic(assertionChain.RollupAddress(), assertionChain.Backend())
	Require(t, err)
	filterer, err := rollupgen.NewRollupUserLogicFilterer(assertionChain.RollupAddress(), assertionChain.Backend())
	Require(t, err)

	// The goal of this test is to observe:
	//
	// 1. The genisis assertion (non-overflow)
	// 2. The assertion of the first 32 blocks of the two batches manually set up
	//    above (non-overflow)
	// 3. The overflow assertion that should be posted in fewer than
	//     minAssertionBlocks. (overflow)
	// 4. One more normal assertion in >= minAssertionBlocks. (non-overflow)

	overflow := true
	nonOverflow := false
	expectedAssertions := []bool{nonOverflow, nonOverflow, overflow, nonOverflow}
	mab64, err := safecast.ToUint64(minAssertionBlocks)
	Require(t, err)

	lastInboxMax := uint64(0)
	lastAssertionBlock := uint64(0)
	fromBlock := uint64(0)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for len(expectedAssertions) > 0 {
		select {
		case <-ticker.C:
			latestBlock, err := l1client.HeaderByNumber(ctx, nil)
			Require(t, err)
			toBlock := latestBlock.Number.Uint64()
			if fromBlock >= toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			it, err := filterer.FilterAssertionCreated(filterOpts, nil, nil)
			Require(t, err)
			for it.Next() {
				if it.Error() != nil {
					t.Fatalf("Error in filter iterator: %v", it.Error())
				}
				t.Log("Received event of assertion created!")
				assertionHash := protocol.AssertionHash{Hash: it.Event.AssertionHash}
				creationInfo, err := assertionChain.ReadAssertionCreationInfo(ctx, assertionHash)
				Require(t, err)
				assertionCreationBlock, err := rollup.GetAssertionCreationBlockForLogLookup(&bind.CallOpts{Context: ctx}, it.Event.AssertionHash)
				Require(t, err)
				creationBlock := assertionCreationBlock.Uint64()
				t.Logf("Created assertion in block: %d", creationBlock)
				newState := protocol.GoGlobalStateFromSolidity(creationInfo.AfterState.GlobalState)
				t.Logf("NewState PosInBatch: %d", newState.PosInBatch)
				inboxMax := creationInfo.InboxMaxCount.Uint64()
				t.Logf("InboxMax: %d", inboxMax)
				blocks := creationBlock - lastAssertionBlock
				// PosInBatch == 0 && inboxMax > lastInboxMax means it is NOT an overflow assertion.
				if newState.PosInBatch == 0 && inboxMax > lastInboxMax {
					if expectedAssertions[0] == overflow {
						t.Errorf("Expected overflow assertion, got non-overflow assertion")
					}
					if blocks < mab64 {
						t.Errorf("non-overflow assertions should have >= =%d blocks between them. Got: %d", mab64, blocks)
					}
				} else {
					if expectedAssertions[0] == nonOverflow {
						t.Errorf("Expected non-overflow assertion, got overflow assertion")
					}
					if blocks >= mab64 {
						t.Errorf("overflow assertions should not have %d blocks between them. Got: %d", mab64, blocks)
					}
				}
				lastAssertionBlock = creationBlock
				lastInboxMax = inboxMax
				expectedAssertions = expectedAssertions[1:]
			}
			fromBlock = toBlock + 1
		case <-ctx.Done():
			return
		}
	}
	// PASS: All expected assertions were seen.
}

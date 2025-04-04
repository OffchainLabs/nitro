// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build challengetest && !race

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	protocol "github.com/offchainlabs/bold/chain-abstraction"
	solimpl "github.com/offchainlabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/offchainlabs/bold/challenge-manager"
	modes "github.com/offchainlabs/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	butil "github.com/offchainlabs/bold/util"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/staker/bold"
)

func TestChallengeProtocolBOLD_L3Support(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithBoldDeployment()

	// Block validation requires db hash scheme.
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	builder.nodeConfig.BlockValidator.Enable = true
	builder.nodeConfig.Staker.Enable = true
	builder.nodeConfig.Staker.Strategy = "MakeNodes"
	builder.nodeConfig.Bold.Strategy = "MakeNodes"
	builder.nodeConfig.Bold.RPCBlockNumber = "latest"
	builder.nodeConfig.Bold.StateProviderConfig.CheckBatchFinality = false
	builder.nodeConfig.Bold.StateProviderConfig.ValidatorName = "L2-validator"
	builder.valnodeConfig.UseJit = false

	cleanupL1AndL2 := builder.Build(t)
	defer cleanupL1AndL2()

	builder.l3Config.execConfig.Caching.StateScheme = rawdb.HashScheme
	builder.l3Config.nodeConfig.Staker.Enable = true
	builder.l3Config.nodeConfig.BlockValidator.Enable = true
	builder.l3Config.nodeConfig.Staker.Strategy = "MakeNodes"
	builder.l3Config.nodeConfig.Bold.Strategy = "MakeNodes"
	builder.l3Config.nodeConfig.Bold.RPCBlockNumber = "latest"
	builder.l3Config.nodeConfig.Bold.StateProviderConfig.CheckBatchFinality = false
	builder.l3Config.nodeConfig.Bold.StateProviderConfig.ValidatorName = "L3-validator"
	builder.l3Config.valnodeConfig.UseJit = false
	cleanupL3FirstNode := builder.BuildL3OnL2(t)
	defer cleanupL3FirstNode()
	firstNodeTestClient := builder.L3

	secondNodeNodeConfig := arbnode.ConfigDefaultL1NonSequencerTest()
	secondNodeNodeConfig.BlockValidator.Enable = true
	secondNodeNodeConfig.Staker.Enable = true
	secondNodeNodeConfig.Staker.Strategy = "MakeNodes"
	secondNodeNodeConfig.Bold.Strategy = "MakeNodes"
	secondNodeNodeConfig.Bold.StateProviderConfig.CheckBatchFinality = false
	secondNodeNodeConfig.Bold.StateProviderConfig.ValidatorName = "Second-L2-validator"
	secondNodeNodeConfig.Bold.RPCBlockNumber = "latest"
	secondNodeTestClient, cleanupL3SecondNode := builder.Build2ndNodeOnL3(t, &SecondNodeParams{nodeConfig: secondNodeNodeConfig})
	defer cleanupL3SecondNode()

	go keepChainMoving(t, ctx, builder.L1Info, builder.L1.Client) // Advance L1.
	go keepChainMoving(t, ctx, builder.L2Info, builder.L2.Client) // Advance L2.

	builder.L2Info.GenerateAccount("HonestAsserter")
	fundL3Staker(t, ctx, builder, builder.L2.Client, "HonestAsserter")
	builder.L2Info.GenerateAccount("EvilAsserter")
	fundL3Staker(t, ctx, builder, builder.L2.Client, "EvilAsserter")

	assertionChain, cleanupHonestChallengeManager := startL3BoldChallengeManager(t, ctx, builder, firstNodeTestClient, "HonestAsserter", nil)
	defer cleanupHonestChallengeManager()

	_ = assertionChain

	_, cleanupEvilChallengeManager := startL3BoldChallengeManager(t, ctx, builder, secondNodeTestClient, "EvilAsserter", func(stateManager BoldStateProviderInterface) BoldStateProviderInterface {
		return &incorrectBlockStateProvider{
			honest:              stateManager,
			chain:               assertionChain,
			wrongAtFirstVirtual: false,
			wrongAtBlockHeight:  blockChallengeLeafHeight - 2,
		}
	})
	defer cleanupEvilChallengeManager()

	TransferBalance(t, "Faucet", "Faucet", common.Big0, builder.L3Info, builder.L3.Client, ctx)

	// Everything's setup, now just wait for the challenge to complete and ensure the honest party won
	filterer, err := rollupgen.NewRollupUserLogicFilterer(builder.l3Addresses.Rollup, builder.L2.Client)
	Require(t, err)

	fromBlock := uint64(0)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := builder.L2.Client.HeaderByNumber(ctx, nil)
			Require(t, err)
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			it, err := filterer.FilterAssertionConfirmed(filterOpts, nil)
			Require(t, err)
			for it.Next() {
				if it.Error() != nil {
					t.Fatalf("Error in filter iterator: %v", it.Error())
				}
				t.Log("Received event of assertion confirmation")
				assertion, err := assertionChain.GetAssertion(ctx, &bind.CallOpts{}, protocol.AssertionHash{
					Hash: it.Event.AssertionHash,
				})
				Require(t, err)
				creationInfo, err := assertionChain.ReadAssertionCreationInfo(ctx, assertion.Id())
				Require(t, err)
				parentAssertionHash := creationInfo.ParentAssertionHash
				parentAssertion, err := assertionChain.GetAssertion(ctx, &bind.CallOpts{}, parentAssertionHash)
				Require(t, err)
				hasSecondChild, err := parentAssertion.HasSecondChild(ctx, &bind.CallOpts{})
				Require(t, err)
				if !hasSecondChild {
					t.Log("Assertion did not have a second child")
					continue
				}
				// If the parent assertion has a second child, it means the child was a confirmed assertion
				// by challenge winner, so then we assert the winner was indeed the honest asserter.
				tx, _, err := builder.L2.Client.TransactionByHash(ctx, it.Event.Raw.TxHash)
				Require(t, err)
				signer := types.NewCancunSigner(tx.ChainId())
				address, err := signer.Sender(tx)
				Require(t, err)
				if address == builder.L2Info.GetAddress("HonestAsserter") {
					t.Log("Honest party confirmed an assertion by challenge win")
					Require(t, it.Close())
					return
				} else {
					t.Fatal("Evil party won a challenge")
				}
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

func fundL3Staker(t *testing.T, ctx context.Context, builder *NodeBuilder, l2Client *ethclient.Client, name string) {
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", name, balance, builder.L2Info, l2Client, ctx)

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(builder.l3Addresses.Rollup, l2Client)
	Require(t, err)
	stakeToken, err := rollupUserLogic.StakeToken(&bind.CallOpts{Context: ctx})
	Require(t, err)
	stakeTokenWeth, err := mocksgen.NewTestWETH9(stakeToken, l2Client)
	Require(t, err)

	txOpts := builder.L2Info.GetDefaultTransactOpts(name, ctx)

	txOpts.Value = big.NewInt(params.Ether)
	tx, err := stakeTokenWeth.Deposit(&txOpts)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	txOpts.Value = nil

	tx, err = stakeTokenWeth.Approve(&txOpts, builder.l3Addresses.Rollup, balance)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	challengeManager, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{Context: ctx})
	Require(t, err)
	tx, err = stakeTokenWeth.Approve(&txOpts, challengeManager, balance)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func startL3BoldChallengeManager(t *testing.T, ctx context.Context, builder *NodeBuilder, node *TestClient, addressName string, mockStateProvider func(BoldStateProviderInterface) BoldStateProviderInterface) (*solimpl.AssertionChain, func()) {
	if !builder.deployBold {
		t.Fatal("bold deployment not enabled")
	}

	var stateManager BoldStateProviderInterface
	var err error
	cacheDir := t.TempDir()
	stateManager, err = bold.NewBOLDStateProvider(
		node.ConsensusNode.BlockValidator,
		node.ConsensusNode.StatelessBlockValidator,
		l2stateprovider.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          addressName,
			MachineLeavesCachePath: cacheDir,
			CheckBatchFinality:     false,
		},
		cacheDir,
	)
	Require(t, err)

	if mockStateProvider != nil {
		stateManager = mockStateProvider(stateManager)
	}

	provider := l2stateprovider.NewHistoryCommitmentProvider(
		stateManager,
		stateManager,
		stateManager,
		[]l2stateprovider.Height{
			l2stateprovider.Height(blockChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(bigStepChallengeLeafHeight),
			l2stateprovider.Height(smallStepChallengeLeafHeight),
		},
		stateManager,
		nil, // Api db
	)

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(builder.l3Addresses.Rollup, builder.L2.Client)
	Require(t, err)
	chalManagerAddr, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{})
	Require(t, err)

	txOpts := builder.L2Info.GetDefaultTransactOpts(addressName, ctx)

	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(node.ConsensusNode.ArbDB, storage.StakerPrefix),
		builder.L3.ConsensusNode.L1Reader,
		&txOpts,
		NewFetcherFromConfig(builder.nodeConfig),
		node.ConsensusNode.SyncMonitor,
		builder.L2Info.Signer.ChainID(),
	)
	Require(t, err)

	assertionChain, err := solimpl.NewAssertionChain(
		ctx,
		builder.l3Addresses.Rollup,
		chalManagerAddr,
		&txOpts,
		butil.NewBackendWrapper(builder.L2.Client, rpc.LatestBlockNumber),
		bold.NewDataPosterTransactor(dp),
		solimpl.WithRpcHeadBlockNumber(rpc.LatestBlockNumber),
	)
	Require(t, err)

	stackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithName(addressName),
		challengemanager.StackWithMode(modes.MakeMode),
		challengemanager.StackWithPostingInterval(time.Second * 3),
		challengemanager.StackWithPollingInterval(time.Second),
		challengemanager.StackWithAverageBlockCreationTime(time.Second),
		challengemanager.StackWithMinimumGapToParentAssertion(0),
	}

	challengeManager, err := challengemanager.NewChallengeStack(
		assertionChain,
		provider,
		stackOpts...,
	)
	Require(t, err)

	challengeManager.Start(ctx)
	return assertionChain, challengeManager.StopAndWait
}

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
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"
	solimpl "github.com/offchainlabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/offchainlabs/bold/challenge-manager"
	modes "github.com/offchainlabs/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestChallengeProtocolBOLDVirtualBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithBoldDeployment()

	// Block validation requires db hash scheme
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme

	valConf := valnode.TestValidationConfig
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(builder.nodeConfig, valStack)

	builder.execConfig.Sequencer.MaxRevertGasReject = 0

	cleanup := builder.Build(t)
	defer cleanup()

	evilNode, cleanupEvilNode := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupEvilNode()

	go keepChainMoving(t, ctx, builder.L1Info, builder.L1.Client)

	builder.L1Info.GenerateAccount("Asserter")
	builder.L1Info.GenerateAccount("EvilAsserter")
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, builder.L1Info, builder.L1.Client, ctx)
	TransferBalance(t, "Faucet", "EvilAsserter", balance, builder.L1Info, builder.L1.Client, ctx)

	cleanupHonestChallengeManager := startBoldChallengeManager(t, ctx, builder, builder.L2, "Asserter")
	defer cleanupHonestChallengeManager()

	// TODO: inject an evil BOLDStateProvider to the evil node (right now it's using an honest one)
	cleanupEvilChallengeManager := startBoldChallengeManager(t, ctx, builder, evilNode, "Asserter")
	defer cleanupEvilChallengeManager()

	// TODO: the rest of the test
}

func startBoldChallengeManager(t *testing.T, ctx context.Context, builder *NodeBuilder, node *TestClient, addressName string) func() {
	if !builder.deployBold {
		t.Fatal("bold deployment not enabled")
	}

	stateManager, err := bold.NewBOLDStateProvider(
		node.ConsensusNode.BlockValidator,
		node.ConsensusNode.StatelessBlockValidator,
		l2stateprovider.Height(blockChallengeLeafHeight),
		&bold.StateProviderConfig{
			ValidatorName:          addressName,
			MachineLeavesCachePath: t.TempDir(),
			CheckBatchFinality:     false,
		},
	)
	Require(t, err)

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

	rollupUserLogic, err := rollupgen.NewRollupUserLogic(builder.addresses.Rollup, builder.L1.Client)
	Require(t, err)
	chalManagerAddr, err := rollupUserLogic.ChallengeManager(&bind.CallOpts{})
	Require(t, err)

	txOpts := builder.L1Info.GetDefaultTransactOpts(addressName, ctx)

	dp, err := arbnode.StakerDataposter(
		ctx,
		rawdb.NewTable(node.ConsensusNode.ArbDB, storage.StakerPrefix),
		node.ConsensusNode.L1Reader,
		&txOpts,
		NewFetcherFromConfig(builder.nodeConfig),
		node.ConsensusNode.SyncMonitor,
		builder.L1Info.Signer.ChainID(),
	)
	Require(t, err)

	assertionChain, err := solimpl.NewAssertionChain(
		ctx,
		builder.addresses.Rollup,
		chalManagerAddr,
		&txOpts,
		builder.L1.Client,
		solimpl.NewDataPosterTransactor(dp),
	)

	Require(t, err)
	challengeManager, err := challengemanager.New(
		ctx,
		assertionChain,
		provider,
		assertionChain.RollupAddress(),
		challengemanager.WithName("honest"),
		challengemanager.WithMode(modes.MakeMode),
		challengemanager.WithAddress(txOpts.From),
		challengemanager.WithAssertionPostingInterval(time.Second*3),
		challengemanager.WithAssertionScanningInterval(time.Second),
		challengemanager.WithAvgBlockCreationTime(time.Second),
	)
	Require(t, err)

	challengeManager.Start(ctx)
	return challengeManager.StopAndWait
}

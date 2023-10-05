// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

//go:build assertion_on_large_number_of_batch_test
// +build assertion_on_large_number_of_batch_test

package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/math"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var (
	blockChallengeLeafHeight     = uint64(1 << 26) // 32
	bigStepChallengeLeafHeight   = uint64(1 << 11) // 2048
	smallStepChallengeLeafHeight = uint64(1 << 20) // 1048576
)

// Helps in testing the feasibility of assertion after the protocol upgrade.
func TestAssertionOnLargeNumberOfBlocks(t *testing.T) {
	setupStartTime := time.Now().Unix()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2node, assertionChain := setupAndPostBatches(t, ctx)

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
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

	challengeLeafHeights := []l2stateprovider.Height{
		l2stateprovider.Height(blockChallengeLeafHeight),
		l2stateprovider.Height(bigStepChallengeLeafHeight),
		l2stateprovider.Height(smallStepChallengeLeafHeight),
	}
	manager, err := staker.NewStateManager(stateless, t.TempDir(), nil)
	Require(t, err)
	provider := l2stateprovider.NewHistoryCommitmentProvider(
		manager,
		manager,
		manager,
		challengeLeafHeights,
		manager,
	)
	poster := assertions.NewPoster(
		assertionChain,
		provider,
		"test",
		time.Second,
	)
	assertion, err := poster.PostAssertion(ctx)
	Require(t, err)
	setupEndTime := time.Now().Unix()
	print("Time taken for setup:")
	print(setupEndTime - setupStartTime)

	assertion, err = poster.PostAssertion(ctx)
	Require(t, err)
	assertionPostingEndTime := time.Now().Unix()
	print("Time taken to post assertion:")
	print(assertionPostingEndTime - setupEndTime)
	startHeight, endHeight, wasmModuleRoot, topLevelClaimEndBatchCount := testCalculatingBlockChallengeLevelZeroEdge(t, ctx, assertionChain, assertion, provider)
	levelZeroEdgeEndTime := time.Now().Unix()
	print("Time taken Calculating BlockChallenge LevelZeroEdge:")
	print(levelZeroEdgeEndTime - assertionPostingEndTime)
	testCalculatingBlockChallengeLevelZeroEdgeBisection(t, ctx, provider, startHeight, endHeight, wasmModuleRoot, topLevelClaimEndBatchCount)
	bisectionOfLevelZeroEdgeEndTime := time.Now().Unix()
	print("Time taken Calculating BlockChallenge LevelZeroEdge Bisection:")
	print(bisectionOfLevelZeroEdgeEndTime - levelZeroEdgeEndTime)

}
func testCalculatingBlockChallengeLevelZeroEdgeBisection(
	t *testing.T,
	ctx context.Context,
	provider *l2stateprovider.HistoryCommitmentProvider,
	startHeight uint64,
	endHeight uint64,
	wasmModuleRoot common.Hash,
	topLevelClaimEndBatchCount uint64,
) {
	bisectTo, err := math.Bisect(startHeight, endHeight)
	Require(t, err)
	_, err = provider.HistoryCommitment(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot: wasmModuleRoot,
			Batch:          l2stateprovider.Batch(topLevelClaimEndBatchCount),
			FromHeight:     l2stateprovider.Height(0),
			UpToHeight:     option.Some[l2stateprovider.Height](l2stateprovider.Height(bisectTo)),
		},
	)

	Require(t, err)
	_, err = provider.PrefixProof(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot: wasmModuleRoot,
			Batch:          l2stateprovider.Batch(topLevelClaimEndBatchCount),
			FromHeight:     l2stateprovider.Height(bisectTo),
			UpToHeight:     option.Some[l2stateprovider.Height](l2stateprovider.Height(endHeight)),
		},
		l2stateprovider.Height(bisectTo),
	)
	Require(t, err)
}

func testCalculatingBlockChallengeLevelZeroEdge(
	t *testing.T,
	ctx context.Context,
	assertionChain protocol.Protocol,
	assertion protocol.Assertion,
	provider *l2stateprovider.HistoryCommitmentProvider,
) (uint64, uint64, common.Hash, uint64) {

	creationInfo, err := assertionChain.ReadAssertionCreationInfo(ctx, assertion.Id())
	Require(t, err)

	startCommit, err := provider.HistoryCommitment(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot: creationInfo.WasmModuleRoot,
			Batch:          l2stateprovider.Batch(0),
			FromHeight:     l2stateprovider.Height(0),
			UpToHeight:     option.Some[l2stateprovider.Height](l2stateprovider.Height(0)),
		},
	)
	Require(t, err)
	levelZeroBlockEdgeHeight := uint64(1 << 26)
	Require(t, err)

	endCommit, err := provider.HistoryCommitment(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot: creationInfo.WasmModuleRoot,
			Batch:          l2stateprovider.Batch(creationInfo.InboxMaxCount.Uint64()),
			FromHeight:     l2stateprovider.Height(0),
			UpToHeight:     option.Some[l2stateprovider.Height](l2stateprovider.Height(levelZeroBlockEdgeHeight)),
		},
	)
	Require(t, err)
	_, err = provider.PrefixProof(
		ctx,
		&l2stateprovider.HistoryCommitmentRequest{
			WasmModuleRoot: creationInfo.WasmModuleRoot,
			Batch:          l2stateprovider.Batch(creationInfo.InboxMaxCount.Uint64()),
			FromHeight:     l2stateprovider.Height(0),
			UpToHeight:     option.Some[l2stateprovider.Height](l2stateprovider.Height(levelZeroBlockEdgeHeight)),
		},
		l2stateprovider.Height(0),
	)
	Require(t, err)
	return startCommit.Height, endCommit.Height, creationInfo.WasmModuleRoot, creationInfo.InboxMaxCount.Uint64()
}
func setupAndPostBatches(t *testing.T, ctx context.Context) (*arbnode.Node, protocol.Protocol) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 250)
	l1Info := NewL1TestInfo(t)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)
	l1Info.GenerateGenesisAccount("RollupOwner", initialBalance)

	chainConfig := params.ArbitrumDevTestChainConfig()
	l1Info, l1Backend, _, _ := createTestL1BlockChain(t, l1Info)
	conf := arbnode.ConfigDefaultL1Test()
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false
	conf.InboxReader.CheckDelay = time.Second

	var valStack *node.Node
	_, valStack = createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	configByValidationNode(t, conf, valStack)

	l1TransactionOpts := l1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		&l1TransactionOpts,
		l1Backend,
		"Weth",
		"WETH",
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)
	value, _ := new(big.Int).SetString("10000", 10)
	l1TransactionOpts.Value = value
	tx, err = tokenBindings.Deposit(&l1TransactionOpts)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)
	l1TransactionOpts.Value = nil
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)
	rollupAddresses, assertionChain := deployBoldContracts(t, ctx, l1Info, l1Backend, chainConfig.ChainID, stakeToken)
	l1Info.SetContract("Bridge", rollupAddresses.Bridge)
	l1Info.SetContract("SequencerInbox", rollupAddresses.SequencerInbox)
	l1Info.SetContract("Inbox", rollupAddresses.Inbox)
	initMessage := getInitMessage(ctx, t, l1Backend, rollupAddresses)

	l2Info, l2Stack, l2ChainDb, l2ArbDb, l2Blockchain := createL2BlockChainWithStackConfig(t, nil, "", chainConfig, initMessage, nil, nil)

	fatalErrChan := make(chan error, 10)
	execConfigFetcher := func() *gethexec.Config { return gethexec.ConfigDefaultTest() }
	execNode, err := gethexec.CreateExecutionNode(ctx, l2Stack, l2ChainDb, l2Blockchain, l1Backend, execConfigFetcher)
	Require(t, err)
	l2Node, err := arbnode.CreateNode(ctx, l2Stack, execNode, l2ArbDb, NewFetcherFromConfig(conf), l2Blockchain.Config(), l1Backend, rollupAddresses, nil, nil, nil, fatalErrChan)
	Require(t, err)
	err = l2Node.Start(ctx)
	Require(t, err)

	l2Info.GenerateAccount("Destination")

	rollup, err := rollupgen.NewRollupAdminLogic(l2Node.DeployInfo.Rollup, l1Backend)
	Require(t, err)
	deployAuth := l1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	_, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(0))
	Require(t, err)

	emptyArray, err := rlp.EncodeToBytes([]uint8{0})
	Require(t, err)
	var out []byte
	for i := 0; i < arbstate.MaxSegmentsPerSequencerMessage-1; i++ {
		out = append(out, emptyArray...)
	}
	batch := []uint8{0}
	compressed, err := arbcompress.CompressWell(out)
	Require(t, err)
	batch = append(batch, compressed...)

	txOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)
	simpleAddress, simple := deploySimple(t, ctx, txOpts, l1Backend)
	seqInbox, err := bridgegen.NewSequencerInbox(rollupAddresses.SequencerInbox, l1Backend)
	Require(t, err)
	tx, err = seqInbox.SetIsBatchPoster(&deployAuth, simpleAddress, true)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)
	for i := 0; i < 3; i++ {
		tx, err = simple.PostManyBatches(&txOpts, rollupAddresses.SequencerInbox, batch, big.NewInt(300))
		Require(t, err)
		receipt, err = EnsureTxSucceeded(ctx, l1Backend, tx)
		Require(t, err)

		nodeSeqInbox, err := arbnode.NewSequencerInbox(l1Backend, rollupAddresses.SequencerInbox, 0)
		Require(t, err)
		batches, err := nodeSeqInbox.LookupBatchesInRange(ctx, receipt.BlockNumber, receipt.BlockNumber)
		Require(t, err)
		if len(batches) != 300 {
			Fatal(t, "300 batch not found after PostManyBatches")
		}
		err = l2Node.InboxTracker.AddSequencerBatches(ctx, l1Backend, batches)
		Require(t, err)
		_, err = l2Node.InboxTracker.GetBatchMetadata(0)
		Require(t, err, "failed to get batch metadata after adding batch:")
	}
	return l2Node, assertionChain
}

func deployBoldContracts(
	t *testing.T,
	ctx context.Context,
	l1info info,
	backend *ethclient.Client,
	chainId *big.Int,
	stakeToken common.Address,
) (*chaininfo.RollupAddresses, *solimpl.AssertionChain) {
	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)

	cfg := challenge_testing.GenerateRollupConfig(
		false,
		locator.LatestWasmModuleRoot(),
		l1TransactionOpts.From,
		chainId,
		common.Address{},
		big.NewInt(1),
		stakeToken,
		rollupgen.ExecutionState{
			GlobalState:   rollupgen.GlobalState{},
			MachineStatus: 1,
		},
		big.NewInt(0),
		common.Address{},
	)
	config, err := json.Marshal(params.ArbitrumDevTestChainConfig())
	if err != nil {
		return nil, nil
	}
	cfg.ChainConfig = string(config)

	addresses, err := setup.DeployFullRollupStack(
		ctx,
		backend,
		&l1TransactionOpts,
		l1info.GetAddress("sequencer"),
		cfg,
		false,
		true,
	)
	Require(t, err)

	asserter := l1info.GetDefaultTransactOpts("asserter", ctx)
	chain, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		&asserter,
		backend,
	)
	Require(t, err)

	chalManager, err := chain.SpecChallengeManager(ctx)
	Require(t, err)
	chalManagerAddr := chalManager.Address()
	seed, _ := new(big.Int).SetString("1000", 10)
	value, _ := new(big.Int).SetString("10000", 10)
	tokenBindings, err := mocksgen.NewTestWETH9(stakeToken, backend)
	Require(t, err)
	tx, err := tokenBindings.TestWETH9Transactor.Transfer(&l1TransactionOpts, asserter.From, seed)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&asserter, addresses.Rollup, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&asserter, chalManagerAddr, value)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, backend, tx)
	Require(t, err)

	return &chaininfo.RollupAddresses{
		Bridge:                 addresses.Bridge,
		Inbox:                  addresses.Inbox,
		SequencerInbox:         addresses.SequencerInbox,
		Rollup:                 addresses.Rollup,
		ValidatorUtils:         addresses.ValidatorUtils,
		ValidatorWalletCreator: addresses.ValidatorWalletCreator,
		DeployedAt:             addresses.DeployedAt,
	}, chain
}

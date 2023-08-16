// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

//go:build assertion_on_large_number_of_batch_test
// +build assertion_on_large_number_of_batch_test

package arbtest

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var (
	blockChallengeLeafHeight     = uint64(1 << 5)  // 32
	bigStepChallengeLeafHeight   = uint64(1 << 11) // 2048
	smallStepChallengeLeafHeight = uint64(1 << 20) // 1048576
)

// Helps in testing the feasibility of assertion after the protocol upgrade.
func TestAssertionOnLargeNumberOfBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l2node, assertionChain := setupAndPostBatches(t, ctx)

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig
	stateless, err := staker.NewStatelessBlockValidator(
		l2node.InboxReader,
		l2node.InboxTracker,
		l2node.TxStreamer,
		l2node.Execution.Recorder,
		l2node.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = stateless.Start(ctx)
	Require(t, err)

	manager, err := staker.NewStateManager(stateless, nil, numOpcodesPerBigStepTest, maxWavmOpcodesTest, t.TempDir())
	Require(t, err)

	poster := assertions.NewPoster(
		assertionChain,
		manager,
		"test",
		time.Second,
	)
	_, err = poster.PostAssertion(ctx)
	Require(t, err)
}

func setupAndPostBatches(t *testing.T, ctx context.Context) (*arbnode.Node, protocol.Protocol) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
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

	sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)

	bridgeAddr, seqInbox, seqInboxAddr := setupSequencerInboxStub(ctx, t, l1Info, l1Backend, chainConfig)

	l2Info, l2Stack, l2ChainDb, l2ArbDb, l2Blockchain := createL2BlockChainWithStackConfig(t, nil, "", chainConfig, initMessage, nil)
	rollupAddresses.Bridge = bridgeAddr
	rollupAddresses.SequencerInbox = seqInboxAddr

	fatalErrChan := make(chan error, 10)
	l2Node, err := arbnode.CreateNode(ctx, l2Stack, l2ChainDb, l2ArbDb, NewFetcherFromConfig(conf), l2Blockchain, l1Backend, rollupAddresses, nil, nil, nil, fatalErrChan)
	Require(t, err)
	err = l2Node.Start(ctx)
	Require(t, err)

	l2Info.GenerateAccount("Destination")

	rollup, err := rollupgen.NewRollupAdminLogic(l2Node.DeployInfo.Rollup, l1Backend)
	Require(t, err)
	deployAuth := l1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))

	for i := 0; i <= int(math.Pow(2, 26)); i++ {
		makeBatch(t, l2Node, l2Info, l1Backend, &sequencerTxOpts, seqInbox, seqInboxAddr, -1)
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
		challenge_testing.WithLevelZeroHeights(&challenge_testing.LevelZeroHeights{
			BlockChallengeHeight:     blockChallengeLeafHeight,
			BigStepChallengeHeight:   bigStepChallengeLeafHeight,
			SmallStepChallengeHeight: smallStepChallengeLeafHeight,
		}),
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
	EnsureTxSucceeded(ctx, backend, tx)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&asserter, addresses.Rollup, value)
	Require(t, err)
	EnsureTxSucceeded(ctx, backend, tx)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&asserter, chalManagerAddr, value)
	Require(t, err)
	EnsureTxSucceeded(ctx, backend, tx)

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

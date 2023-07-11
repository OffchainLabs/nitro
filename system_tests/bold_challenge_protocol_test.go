package arbtest

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	challenge_testing "github.com/OffchainLabs/challenge-protocol-v2/testing"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

func setupAddrDeployments(
	t *testing.T,
	ctx context.Context,
	l1info info,
	backend *ethclient.Client,
	chainId *big.Int,
	sequecerInboxAddr common.Address,
) *setup.ChainSetup {

	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	SendWaitTestTransactions(t, ctx, backend, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil),
	})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	wasmModuleRoot := locator.LatestWasmModuleRoot()

	prod := false
	loserStakeEscrow := common.Address{}
	miniStake := big.NewInt(1)
	cfg := challenge_testing.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		l1TransactionOpts.From,
		chainId,
		loserStakeEscrow,
		miniStake,
	)
	addresses, err := setup.DeployFullRollupStack(
		ctx,
		backend,
		&l1TransactionOpts,
		sequecerInboxAddr,
		cfg,
	)
	Require(t, err)

	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	asserter := l1info.GetDefaultTransactOpts("asserter", ctx)
	chain, err := solimpl.NewAssertionChain(
		ctx,
		addresses.Rollup,
		&asserter,
		backend,
	)
	Require(t, err)

	return &setup.ChainSetup{
		Chains:       []*solimpl.AssertionChain{chain},
		Addrs:        addresses,
		RollupConfig: cfg,
	}
}

func TestBoldChallengeProtocol(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info := NewL1TestInfo(t)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)
	l1Info.GenerateGenesisAccount("asserter", initialBalance)
	l1Info.GenerateGenesisAccount("sequencer", initialBalance)

	chainConfig := params.ArbitrumDevTestChainConfig()
	l1Info, l1Backend, _, _ := createTestL1BlockChain(t, l1Info)
	conf := arbnode.ConfigDefaultL1Test()
	conf.BlockValidator.Enable = false
	conf.BatchPoster.Enable = false
	conf.InboxReader.CheckDelay = time.Second

	var valStack *node.Node
	_, valStack = createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	configByValidationNode(t, conf, valStack)

	asserterBridgeAddr, asserterSeqInbox, asserterSeqInboxAddr := setupSequencerInboxStub(
		ctx,
		t,
		l1Info,
		l1Backend,
		chainConfig,
	)
	_ = asserterBridgeAddr
	_ = asserterSeqInbox
	_ = asserterSeqInboxAddr

	rollupSetup := setupAddrDeployments(t, ctx, l1Info, l1Backend, chainConfig.ChainID, asserterSeqInboxAddr)
	_ = rollupSetup

	// // asserterRollupAddresses.Bridge = asserterBridgeAddr
	// // asserterRollupAddresses.SequencerInbox = asserterSeqInboxAddr
	asserterL2Info := NewArbTestInfo(t, chainConfig.ChainID)
	fatalErrChan := make(chan error, 10)
	asserterRollupAddresses := &chaininfo.RollupAddresses{
		Bridge:                 rollupSetup.Addrs.Bridge,
		Inbox:                  rollupSetup.Addrs.Inbox,
		SequencerInbox:         rollupSetup.Addrs.SequencerInbox,
		Rollup:                 rollupSetup.Addrs.Rollup,
		ValidatorUtils:         rollupSetup.Addrs.ValidatorUtils,
		ValidatorWalletCreator: rollupSetup.Addrs.ValidatorWalletCreator,
		DeployedAt:             rollupSetup.Addrs.DeployedAt,
	}
	asserterL2, asserterExec := createL2Nodes(t, ctx, conf, chainConfig, l1Backend, asserterL2Info, asserterRollupAddresses, nil, nil, fatalErrChan)
	Require(t, asserterExec.Initialize(ctx))
	err := asserterL2.Start(ctx)
	Require(t, err)
	Require(t, asserterExec.Start(ctx))

	asserterL2Info.GenerateAccount("Destination")

	// sequencerTxOpts := l1Info.GetDefaultTransactOpts("sequencer", ctx)
	// seqNum := common.Big2
	// makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)

	// seqNum.Add(seqNum, common.Big1)
	// makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)

	// seqNum.Add(seqNum, common.Big1)
	// makeBatch(t, asserterL2, asserterL2Info, l1Backend, &sequencerTxOpts, asserterSeqInbox, asserterSeqInboxAddr, -1)

	// trueSeqInboxAddr := challengerSeqInboxAddr
	// trueDelayedBridge := challengerBridgeAddr
	// ospEntry := DeployOneStepProofEntry(t, ctx, &deployerTxOpts, l1Backend)

	// locator, err := server_common.NewMachineLocator("")
	// if err != nil {
	// 	Fail(t, err)
	// }
	// wasmModuleRoot := locator.LatestWasmModuleRoot()
	// if (wasmModuleRoot == common.Hash{}) {
	// 	Fail(t, "latest machine not found")
	// }

	// asserterGenesis := asserterExec.ArbInterface.BlockChain().Genesis()
	// challengerGenesis := challengerExec.ArbInterface.BlockChain().Genesis()
	// if asserterGenesis.Hash() != challengerGenesis.Hash() {
	// 	Fail(t, "asserter and challenger have different genesis hashes")
	// }
	// asserterLatestBlock := asserterExec.ArbInterface.BlockChain().CurrentBlock()
	// challengerLatestBlock := challengerExec.ArbInterface.BlockChain().CurrentBlock()
	// if asserterLatestBlock.Hash() == challengerLatestBlock.Hash() {
	// 	Fail(t, "asserter and challenger have the same end block")
	// }

	// asserterStartGlobalState := validator.GoGlobalState{
	// 	BlockHash:  asserterGenesis.Hash(),
	// 	Batch:      1,
	// 	PosInBatch: 0,
	// }
	// asserterEndGlobalState := validator.GoGlobalState{
	// 	BlockHash:  asserterLatestBlock.Hash(),
	// 	Batch:      4,
	// 	PosInBatch: 0,
	// }
	// numBlocks := asserterLatestBlock.NumberU64() - asserterGenesis.NumberU64()

	// _, challengeManagerAddr := CreateChallenge(
	// 	t,
	// 	ctx,
	// 	&deployerTxOpts,
	// 	l1Backend,
	// 	ospEntry,
	// 	trueSeqInboxAddr,
	// 	trueDelayedBridge,
	// 	wasmModuleRoot,
	// 	asserterStartGlobalState,
	// 	asserterEndGlobalState,
	// 	numBlocks,
	// 	l1Info.GetAddress("asserter"),
	// 	l1Info.GetAddress("challenger"),
	// )

	// confirmLatestBlock(ctx, t, l1Info, l1Backend)

	asserterValidator, err := staker.NewStatelessBlockValidator(asserterL2.InboxReader, asserterL2.InboxTracker, asserterL2.TxStreamer, asserterExec.Recorder, asserterL2.ArbDB, nil, StaticFetcherFrom(t, &conf.BlockValidator), valStack)
	if err != nil {
		Fail(t, err)
	}
	err = asserterValidator.Start(ctx)
	if err != nil {
		Fail(t, err)
	}
	defer asserterValidator.Stop()

	Fail(t, "Could not start stateless validator")

	// Fail(t, "challenge timed out without winner")
}

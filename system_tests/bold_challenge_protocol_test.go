package arbtest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/solgen/go/mocksgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/execution"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_common"
)

// One Arbitrum block had 1,849,212,947 total opcodes. The closest, higher power of two
// is 2^31. So we if we make our small step heights 2^20, we need 2048 big steps
// to cover the block. With 2^20, our small step history commitments will be approx
// 32 Mb of state roots in memory at once.
var (
	blockChallengeLeafHeight     = uint64(1 << 5) // 32
	bigStepChallengeLeafHeight   = uint64(2048)   // this + the number below should be 2^43 total WAVM opcodes per block.
	smallStepChallengeLeafHeight = uint64(1 << 20)
)

func TestBoldProtocol(t *testing.T) {
	t.Parallel()
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)

	_, l2nodeA, l2clientA, _, l1info, _, l1client, l1stack, assertionChain, stakeTokenAddr := createTestNodeOnL1ForBoldProtocol(t, ctx, true, nil, l2chainConfig, nil, l2info)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()

	l2clientB, l2nodeB, assertionChainB := create2ndNodeWithConfigForBoldProtocol(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, arbnode.ConfigDefaultL1Test(), nil, stakeTokenAddr)
	defer l2nodeB.StopAndWait()

	nodeAGenesis := l2nodeA.Execution.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := l2nodeB.Execution.Backend.APIBackend().CurrentHeader().Hash()
	if nodeAGenesis != nodeBGenesis {
		Fail(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
	}
	bridgeBalancesToBoldL2s(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2clientA, l2clientB, ctx)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
	TransferBalance(t, "Faucet", "EvilAsserter", balance, l1info, l1client, ctx)
	//l1authB := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)

	t.Log("Setting the minimum assertion period")
	rollup, err := rollupgen.NewRollupAdminLogicTransactor(assertionChain.RollupAddress(), l1client)
	Require(t, err)
	tx, err := rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(0))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	rollup, err = rollupgen.NewRollupAdminLogicTransactor(assertionChainB.RollupAddress(), l1client)
	Require(t, err)
	tx, err = rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(0))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	// valConfig := staker.L1ValidatorConfig{}
	// valConfig.Strategy = "MakeNodes"
	// _, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	// blockValidatorConfig := staker.TestBlockValidatorConfig

	// statelessA, err := staker.NewStatelessBlockValidator(
	// 	l2nodeA.InboxReader,
	// 	l2nodeA.InboxTracker,
	// 	l2nodeA.TxStreamer,
	// 	l2nodeA.Execution.Recorder,
	// 	l2nodeA.ArbDB,
	// 	nil,
	// 	StaticFetcherFrom(t, &blockValidatorConfig),
	// 	valStack,
	// )
	// Require(t, err)
	// err = statelessA.Start(ctx)
	// Require(t, err)

	// statelessB, err := staker.NewStatelessBlockValidator(
	// 	l2nodeB.InboxReader,
	// 	l2nodeB.InboxTracker,
	// 	l2nodeB.TxStreamer,
	// 	l2nodeB.Execution.Recorder,
	// 	l2nodeB.ArbDB,
	// 	nil,
	// 	StaticFetcherFrom(t, &blockValidatorConfig),
	// 	valStack,
	// )
	// Require(t, err)
	// err = statelessB.Start(ctx)
	// Require(t, err)

	// stateManager, err := staker.NewStateManager(
	// 	statelessA,
	// 	nil,
	// 	smallStepChallengeLeafHeight,
	// 	smallStepChallengeLeafHeight*bigStepChallengeLeafHeight,
	// 	"/tmp/good",
	// )
	// Require(t, err)
	// poster := assertions.NewPoster(
	// 	assertionChain,
	// 	stateManager,
	// 	"good",
	// 	time.Hour,
	// )

	// stateManagerB, err := staker.NewStateManager(
	// 	statelessB,
	// 	nil,
	// 	smallStepChallengeLeafHeight,
	// 	smallStepChallengeLeafHeight*bigStepChallengeLeafHeight,
	// 	"/tmp/evil",
	// )
	// Require(t, err)
	// chainB, err := solimpl.NewAssertionChain(
	// 	ctx,
	// 	assertionChain.RollupAddress(),
	// 	&l1authB,
	// 	l1client,
	// )
	// Require(t, err)
	// posterB := assertions.NewPoster(
	// 	chainB,
	// 	stateManagerB,
	// 	"evil",
	// 	time.Hour,
	// )

	// t.Log("Sending a tx from faucet to L2 node A background user")
	// l2info.GenerateAccount("BackgroundUser")
	// tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, common.Big1, nil)
	// err = l2clientA.SendTransaction(ctx, tx)
	// Require(t, err)
	// _, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	// Require(t, err)

	// t.Log("Sending a tx from faucet to L2 node B background user")
	// l2info.Accounts["Faucet"].Nonce = 0
	// tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, common.Big2, nil)
	// err = l2clientB.SendTransaction(ctx, tx)
	// Require(t, err)
	// _, err = EnsureTxSucceeded(ctx, l2clientB, tx)
	// Require(t, err)

	// bcA, err := l2nodeA.InboxTracker.GetBatchCount()
	// Require(t, err)
	// bcB, err := l2nodeB.InboxTracker.GetBatchCount()
	// Require(t, err)
	// msgA, err := l2nodeA.InboxTracker.GetBatchMessageCount(bcA - 1)
	// Require(t, err)
	// msgB, err := l2nodeB.InboxTracker.GetBatchMessageCount(bcB - 1)
	// Require(t, err)
	// accA, err := l2nodeA.InboxTracker.GetBatchAcc(bcA - 1)
	// Require(t, err)
	// accB, err := l2nodeB.InboxTracker.GetBatchAcc(bcB - 1)
	// Require(t, err)
	// t.Logf("Node A, count %d, msgs %d, acc %s", bcA, msgA, accA)
	// t.Logf("Node B, count %d, msgs %d, acc %s", bcB, msgB, accB)

	// nodeALatest := l2nodeA.Execution.Backend.APIBackend().CurrentHeader().Hash()
	// nodeBLatest := l2nodeB.Execution.Backend.APIBackend().CurrentHeader().Hash()
	// if nodeALatest == nodeBLatest {
	// 	Fail(t, "node A L2 hash", nodeALatest, "matches node B L2 hash", nodeBLatest)
	// }

	// t.Log("Honest party posting assertion at batch 1, pos 0")
	// _, err = poster.PostAssertion(ctx)
	// Require(t, err)

	// time.Sleep(10 * time.Second)

	// t.Log("Honest party posting assertion at batch 2, pos 0")
	// _, err = poster.PostAssertion(ctx)
	// Require(t, err)

	// t.Log("Evil party posting rival assertion at batch 2, pos 0")
	// _, err = posterB.PostAssertion(ctx)
	// Require(t, err)

	// manager, err := challengemanager.New(
	// 	ctx,
	// 	assertionChain,
	// 	l1client,
	// 	stateManager,
	// 	assertionChain.RollupAddress(),
	// 	challengemanager.WithName("honest"),
	// 	challengemanager.WithMode(modes.DefensiveMode),
	// 	challengemanager.WithAssertionPostingInterval(time.Hour),
	// 	challengemanager.WithAssertionScanningInterval(5*time.Second),
	// 	challengemanager.WithEdgeTrackerWakeInterval(time.Second),
	// )
	// Require(t, err)
	// manager.Start(ctx)

	// managerB, err := challengemanager.New(
	// 	ctx,
	// 	chainB,
	// 	l1client,
	// 	stateManagerB,
	// 	assertionChain.RollupAddress(),
	// 	challengemanager.WithName("evil"),
	// 	challengemanager.WithMode(modes.DefensiveMode),
	// 	challengemanager.WithAssertionPostingInterval(time.Hour),
	// 	challengemanager.WithAssertionScanningInterval(5*time.Second),
	// 	challengemanager.WithEdgeTrackerWakeInterval(time.Second),
	// )
	// Require(t, err)
	// managerB.Start(ctx)

	// creationInfo, err := chainB.ReadAssertionCreationInfo(ctx, honest.Id())
	// Require(t, err)

	// entry, err := statelessA.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(1))
	// Require(t, err)
	// input, err := entry.ToInput()
	// Require(t, err)
	// execRun, err := statelessA.ExecutionSpawner().CreateExecutionRun(creationInfo.WasmModuleRoot, input).Await(ctx)
	// Require(t, err)

	// bigStepLeaves := execRun.GetBigStepLeavesUpTo(bigStepChallengeLeafHeight, smallStepChallengeLeafHeight)
	// result, err := bigStepLeaves.Await(ctx)
	// Require(t, err)
	// t.Logf("Got result %d with first %#x and last %#x", len(result), result[0], result[len(result)-1])

	// entry, err = statelessA.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(1))
	// Require(t, err)
	// input, err = entry.ToInput()
	// Require(t, err)
	// execRun, err = statelessA.ExecutionSpawner().CreateExecutionRun(creationInfo.WasmModuleRoot, input).Await(ctx)
	// Require(t, err)

	// t.Log("=======")
	// bigStep := uint64(58)
	// bigStepLeaves = execRun.GetSmallStepLeavesUpTo(bigStep, smallStepChallengeLeafHeight, smallStepChallengeLeafHeight)
	// result, err = bigStepLeaves.Await(ctx)
	// Require(t, err)
	// t.Logf("Got result %d with first %#x and last %#x", len(result), result[0], result[len(result)-1])

	// entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(blockHeight))
	// input, err := entry.ToInput()
	// execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	// bigStepLeaves := execRun.GetSmallStepLeavesUpTo(toBigStep, s.numOpcodesPerBigStep)
	// result, err := bigStepLeaves.Await(ctx)

	time.Sleep(time.Hour)

}

func createTestNodeOnL1ForBoldProtocol(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	stackConfig *node.Config,
	l2info_in info,
) (
	l2info info, currentNode *arbnode.Node, l2client *ethclient.Client, l2stack *node.Node,
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
	assertionChain *solimpl.AssertionChain, stakeTokenAddr common.Address,
) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	if chainConfig == nil {
		chainConfig = params.ArbitrumDevTestChainConfig()
	}
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 0
	fatalErrChan := make(chan error, 10)
	l1info, l1client, l1backend, l1stack = createTestL1BlockChain(t, nil)
	var l2chainDb ethdb.Database
	var l2arbDb ethdb.Database
	var l2blockchain *core.BlockChain
	l2info = l2info_in
	if l2info == nil {
		l2info = NewArbTestInfo(t, chainConfig.ChainID)
	}

	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")
	l1info.GenerateAccount("Asserter")
	l1info.GenerateAccount("EvilAsserter")

	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Asserter", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "EvilAsserter", 30000, big.NewInt(9223372036854775807), nil),
	})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		&l1TransactionOpts,
		l1client,
		"Weth",
		"WETH",
	)
	Require(t, err)
	EnsureTxSucceeded(ctx, l1client, tx)
	stakeTokenAddr = stakeToken
	value, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		t.Fatal(t, "could not set value")
	}
	l1TransactionOpts.Value = value
	tx, err = tokenBindings.Deposit(&l1TransactionOpts)
	Require(t, err)
	EnsureTxSucceeded(ctx, l1client, tx)
	l1TransactionOpts.Value = nil

	addresses, assertionChainBindings := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, stakeToken)

	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	_, l2stack, l2chainDb, l2arbDb, l2blockchain = createL2BlockChainWithStackConfig(t, l2info, "", chainConfig, getInitMessage(ctx, t, l1client, addresses), stackConfig)
	assertionChain = assertionChainBindings
	var sequencerTxOptsPtr *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	if isSequencer {
		sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
		sequencerTxOptsPtr = &sequencerTxOpts
		dataSigner = signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	}

	if !isSequencer {
		nodeConfig.BatchPoster.Enable = false
		nodeConfig.DelayedSequencer.Enable = false
	}

	AddDefaultValNode(t, ctx, nodeConfig, true)

	currentNode, err = arbnode.CreateNode(
		ctx, l2stack, l2chainDb, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain, l1client,
		addresses, sequencerTxOptsPtr, sequencerTxOptsPtr, dataSigner, fatalErrChan,
	)
	Require(t, err)

	Require(t, currentNode.Start(ctx))

	l2client = ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode)

	return
}

func deployContractsOnly(
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
		stakeToken,
		challenge_testing.WithLayerZeroHeights(&protocol.LayerZeroHeights{
			BlockChallengeHeight:     blockChallengeLeafHeight,
			BigStepChallengeHeight:   bigStepChallengeLeafHeight,
			SmallStepChallengeHeight: smallStepChallengeLeafHeight,
		}),
	)
	config, err := json.Marshal(params.ArbitrumDevTestChainConfig())
	Require(t, err)
	cfg.ChainConfig = string(config)
	addresses, err := setup.DeployFullRollupStack(
		ctx,
		backend,
		&l1TransactionOpts,
		l1info.GetAddress("Sequencer"),
		cfg,
		false, // do not use mock bridge.
	)
	Require(t, err)

	asserter := l1info.GetDefaultTransactOpts("Asserter", ctx)
	evilAsserter := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)
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
	seed, ok := new(big.Int).SetString("1000", 10)
	if !ok {
		t.Fatal("not ok")
	}
	value, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		t.Fatal(t, "could not set value")
	}
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

	tx, err = tokenBindings.TestWETH9Transactor.Transfer(&l1TransactionOpts, evilAsserter.From, seed)
	Require(t, err)
	EnsureTxSucceeded(ctx, backend, tx)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&evilAsserter, addresses.Rollup, value)
	Require(t, err)
	EnsureTxSucceeded(ctx, backend, tx)
	tx, err = tokenBindings.TestWETH9Transactor.Approve(&evilAsserter, chalManagerAddr, value)
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

func bridgeBalancesToBoldL2s(
	t *testing.T, account string, amount *big.Int, l1info info, l2info info, l1client client, l2clientA client, l2clientB client, ctx context.Context,
) (*types.Transaction, *types.Receipt) {
	t.Helper()

	// setup or validate the same account on l2info
	l1acct := l1info.GetInfoWithPrivKey(account)
	if l2info.Accounts[account] == nil {
		l2info.SetFullAccountInfo(account, &AccountInfo{
			Address:    l1acct.Address,
			PrivateKey: l1acct.PrivateKey,
			Nonce:      0,
		})
	} else {
		l2acct := l2info.GetInfoWithPrivKey(account)
		if l2acct.PrivateKey.X.Cmp(l1acct.PrivateKey.X) != 0 ||
			l2acct.PrivateKey.Y.Cmp(l1acct.PrivateKey.Y) != 0 {
			Fatal(t, "l2 account already exists and not compatible to l1")
		}
	}

	// check previous balance
	l2Balance, err := l2clientA.BalanceAt(ctx, l2info.GetAddress("Faucet"), nil)
	Require(t, err)
	l2BalanceB, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("Faucet"), nil)
	Require(t, err)

	// send transaction
	data, err := hex.DecodeString("0f4d14e9000000000000000000000000000000000000000000000000000082f79cd90000")
	Require(t, err)
	tx := l1info.PrepareTx(account, "Inbox", l1info.TransferGas*100, amount, data)
	err = l1client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	tx = l1info.PrepareTx(account, "EvilInbox", l1info.TransferGas*100, amount, data)
	err = l1client.SendTransaction(ctx, tx)
	Require(t, err)
	res, err := EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
	_ = res

	// wait for balance to appear in l2
	l2Balance.Add(l2Balance, amount)
	l2BalanceB.Add(l2BalanceB, amount)
	for i := 0; true; i++ {
		balanceA, err := l2clientA.BalanceAt(ctx, l2info.GetAddress("Faucet"), nil)
		Require(t, err)
		balanceB, err := l2clientB.BalanceAt(ctx, l2info.GetAddress("Faucet"), nil)
		Require(t, err)
		if balanceA.Cmp(l2Balance) >= 0 && balanceB.Cmp(l2BalanceB) >= 0 {
			t.Log("Balance was bridged to two L2 nodes successfully")
			break
		}
		TransferBalance(t, "Faucet", "User", big.NewInt(1), l1info, l1client, ctx)
		if i > 50 {
			Fatal(t, "bridging failed")
		}
		<-time.After(time.Millisecond * 100)
	}

	return tx, res
}

func create2ndNodeWithConfigForBoldProtocol(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	nodeConfig *arbnode.Config,
	stackConfig *node.Config,
	stakeTokenAddr common.Address,
) (*ethclient.Client, *arbnode.Node, *solimpl.AssertionChain) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	nodeConfig.BatchPoster.DataPoster.MaxMempoolTransactions = 0
	fatalErrChan := make(chan error, 10)
	l1rpcClient, err := l1stack.Attach()
	if err != nil {
		Fatal(t, err)
	}
	l1client := ethclient.NewClient(l1rpcClient)

	if stackConfig == nil {
		stackConfig = stackConfigForTest(t)
	}
	l2stack, err := node.New(stackConfig)
	Require(t, err)

	l2chainDb, err := l2stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	l2arbDb, err := l2stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	chainConfig := first.Execution.ArbInterface.BlockChain().Config()
	addresses, assertionChain := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, stakeTokenAddr)

	l1info.SetContract("EvilBridge", addresses.Bridge)
	l1info.SetContract("EvilSequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("EvilInbox", addresses.Inbox)

	AddDefaultValNode(t, ctx, nodeConfig, true)

	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	txOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)

	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)
	initMessage := getInitMessage(ctx, t, l1client, first.DeployInfo)

	l2blockchain, err := execution.WriteOrTestBlockChain(l2chainDb, nil, initReader, chainConfig, initMessage, arbnode.ConfigDefaultL2Test().TxLookupLimit, 0)
	Require(t, err)

	l2node, err := arbnode.CreateNode(ctx, l2stack, l2chainDb, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain, l1client, addresses, &txOpts, &txOpts, dataSigner, fatalErrChan)
	Require(t, err)

	Require(t, l2node.Start(ctx))

	l2client := ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, l2node)

	return l2client, l2node, assertionChain
}

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/mocksgen"
	challenge_testing "github.com/OffchainLabs/challenge-protocol-v2/testing"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
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
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/execution/execclient"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

func TestBoldProtocol(t *testing.T) {
	t.Parallel()
	faultyStaker := true
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	l2chainConfig := params.ArbitrumDevTestChainConfig()
	l2info := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(l2chainConfig.ChainID)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	_, l2nodeA, l2clientA, _, l1info, _, l1client, l1stack, assertionChain := createTestNodeOnL1ForBoldProtocol(t, ctx, true, nil, nil, l2chainConfig, nil, l2info)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()
	execNodeA := getExecNode(t, l2nodeA)
	_ = l2clientA
	_ = l1client

	if faultyStaker {
		l2info.GenerateGenesisAccount("FaultyAddr", common.Big1)
	}
	l2clientB, l2nodeB := Create2ndNodeWithConfig(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, arbnode.ConfigDefaultL1Test(), gethexec.ConfigDefaultTest(), nil)
	defer l2nodeB.StopAndWait()
	execNodeB := getExecNode(t, l2nodeB)
	_ = l2clientB

	nodeAGenesis := execNodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := execNodeB.Backend.APIBackend().CurrentHeader().Hash()
	if faultyStaker {
		if nodeAGenesis == nodeBGenesis {
			Fail(t, "node A L2 genesis hash", nodeAGenesis, "== node B L2 genesis hash", nodeBGenesis)
		}
	} else {
		if nodeAGenesis != nodeBGenesis {
			Fail(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
		}
	}
	BridgeBalance(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2clientA, ctx)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	_ = deployAuth

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
	l1authA := l1info.GetDefaultTransactOpts("Asserter", ctx)

	valWalletAddrAPtr, err := staker.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	valWalletAddrA := *valWalletAddrAPtr
	valWalletAddrCheck, err := staker.GetValidatorWalletContract(ctx, l2nodeA.DeployInfo.ValidatorWalletCreator, 0, &l1authA, l2nodeA.L1Reader, true)
	Require(t, err)
	if valWalletAddrA == *valWalletAddrCheck {
		Require(t, err, "didn't cache validator wallet address", valWalletAddrA.String(), "vs", valWalletAddrCheck.String())
	}

	edgeManagerAddr, err := assertionChain.SpecChallengeManager(ctx)
	Require(t, err)
	edgeHeight, err := edgeManagerAddr.LevelZeroBlockEdgeHeight(ctx)
	Require(t, err)
	t.Logf("WE HAVE THE ASSERTION CHAIN: %d", edgeHeight)

	// rollup, err := rollupgen.NewAbsRollupUserLogicCaller(r)
	// tx, err := rollup.SetMinimumAssertionPeriod(&deployAuth, big.NewInt(1))
	// Require(t, err)
	// _, err = EnsureTxSucceeded(ctx, l1client, tx)
	// Require(t, err)

	valConfig := staker.L1ValidatorConfig{}

	valWalletA, err := staker.NewContractValidatorWallet(nil, l2nodeA.DeployInfo.ValidatorWalletCreator, l2nodeA.DeployInfo.Rollup, l2nodeA.L1Reader, &l1authA, 0, func(common.Address) {})
	Require(t, err)
	valConfig.Strategy = "MakeNodes"

	_, valStack := createTestValidationNode(t, ctx, &valnode.TestValidationConfig)
	blockValidatorConfig := staker.TestBlockValidatorConfig

	statelessA, err := staker.NewStatelessBlockValidator(
		l2nodeA.InboxReader,
		l2nodeA.InboxTracker,
		l2nodeA.TxStreamer,
		execNodeA,
		l2nodeA.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessA.Start(ctx)
	Require(t, err)
	_ = valWalletA
	managerA, err := staker.NewManager(ctx, valWalletA.RollupAddress(), &l1authA, bind.CallOpts{}, l2nodeA.L1Reader.Client(), statelessA, nil)
	Require(t, err)
	managerA.Start(ctx)

	// Continually make L2 transactions in a background thread
	// backgroundTxsCtx, cancelBackgroundTxs := context.WithCancel(ctx)
	// backgroundTxsShutdownChan := make(chan struct{})
	// defer (func() {
	// 	cancelBackgroundTxs()
	// 	<-backgroundTxsShutdownChan
	// })()
	// go (func() {
	// 	defer close(backgroundTxsShutdownChan)
	// 	err := makeBackgroundTxs(backgroundTxsCtx, l2info, l2clientA, l2clientB, faultyStaker)
	// 	if !errors.Is(err, context.Canceled) {
	// 		log.Warn("error making background txs", "err", err)
	// 	}
	// })()
	time.Sleep(time.Minute)

}

func createTestNodeOnL1ForBoldProtocol(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	execConfig *gethexec.Config,
	chainConfig *params.ChainConfig,
	stackConfig *node.Config,
	l2info_in info,
) (
	l2info info, currentNode *arbnode.Node, l2client *ethclient.Client, l2stack *node.Node,
	l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
	assertionChain *solimpl.AssertionChain,
) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	if execConfig == nil {
		execConfig = gethexec.ConfigDefaultTest()
	}
	if chainConfig == nil {
		chainConfig = params.ArbitrumDevTestChainConfig()
	}
	fatalErrChan := make(chan error, 10)
	l1info, l1client, l1backend, l1stack = createTestL1BlockChain(t, nil)
	var l2chainDb ethdb.Database
	var l2arbDb ethdb.Database
	var l2blockchain *core.BlockChain
	l2info = l2info_in
	if l2info == nil {
		l2info = NewArbTestInfo(t, chainConfig.ChainID)
	}
	_, l2stack, l2chainDb, l2arbDb, l2blockchain = createL2BlockChainWithStackConfig(t, l2info, "", chainConfig, stackConfig)
	addresses, assertionChainBindings := deployBoldProtocolContracts(t, ctx, l1info, l1client, chainConfig.ChainID)
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
		nodeConfig.Sequencer = false
		nodeConfig.DelayedSequencer.Enable = false
		execConfig.Sequencer.Enable = false
	}

	AddDefaultValNode(t, ctx, nodeConfig, true)

	Require(t, execConfig.Validate())
	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(ctx, l2stack, l2chainDb, l2blockchain, l1client, execConfigFetcher)
	Require(t, err)

	execclient := execclient.NewClient(StaticFetcherFrom(t, &rpcclient.TestClientConfig), l2stack)
	currentNode, err = arbnode.CreateNode(
		ctx, l2stack, execclient, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client,
		addresses, sequencerTxOptsPtr, sequencerTxOptsPtr, dataSigner, fatalErrChan,
	)
	Require(t, err)

	Require(t, execNode.Initialize(ctx))

	Require(t, currentNode.Start(ctx))

	Require(t, execNode.Start(ctx))

	l2client = ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode, execNode)

	return
}

func deployBoldProtocolContracts(
	t *testing.T,
	ctx context.Context,
	l1info info,
	backend *ethclient.Client,
	chainId *big.Int,
) (*chaininfo.RollupAddresses, *solimpl.AssertionChain) {

	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")
	l1info.GenerateAccount("Asserter")

	SendWaitTestTransactions(t, ctx, backend, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Asserter", 30000, big.NewInt(9223372036854775807), nil),
	})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	wasmModuleRoot := locator.LatestWasmModuleRoot()

	prod := false
	loserStakeEscrow := common.Address{}
	miniStake := big.NewInt(1)

	stakeToken, tx, tokenBindings, err := mocksgen.DeployTestWETH9(
		&l1TransactionOpts,
		backend,
		"Weth",
		"WETH",
	)
	Require(t, err)
	EnsureTxSucceeded(ctx, backend, tx)
	value, ok := new(big.Int).SetString("10000", 10)
	if !ok {
		t.Fatal(t, "could not set value")
	}
	l1TransactionOpts.Value = value
	tx, err = tokenBindings.Deposit(&l1TransactionOpts)
	Require(t, err)
	EnsureTxSucceeded(ctx, backend, tx)
	l1TransactionOpts.Value = nil

	cfg := challenge_testing.GenerateRollupConfig(
		prod,
		wasmModuleRoot,
		l1TransactionOpts.From,
		chainId,
		loserStakeEscrow,
		miniStake,
		stakeToken,
	)
	addresses, err := setup.DeployFullRollupStack(
		ctx,
		backend,
		&l1TransactionOpts,
		l1info.GetAddress("Sequencer"),
		cfg,
	)
	Require(t, err)

	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	asserter := l1info.GetDefaultTransactOpts("Asserter", ctx)
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
	tx, err = tokenBindings.TestWETH9Transactor.Transfer(&l1TransactionOpts, asserter.From, seed)
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

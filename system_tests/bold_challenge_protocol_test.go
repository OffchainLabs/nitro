package arbtest

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/OffchainLabs/bold/assertions"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	modes "github.com/OffchainLabs/bold/challenge-manager/types"
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
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
)

// One Arbitrum block had 1,849,212,947 total opcodes. The closest, higher power of two
// is 2^31. So we if we make our small step heights 2^20, we need 2048 big steps
// to cover the block. With 2^20, our small step history commitments will be approx
// 32 Mb of state roots in memory at once.
var (
	blockChallengeLeafHeight     = uint64(1 << 5)  // 32
	bigStepChallengeLeafHeight   = uint64(1 << 11) // 2048
	smallStepChallengeLeafHeight = uint64(1 << 20) // 1048576
)

// func TestWeirdProofError(t *testing.T) {
// 	rawRoots := [][]byte{
// 		hexutil.MustDecode("0xd781e3a672f2f16cbf9f3a2f4d2d9714ba2818f4b378870010d4c78b60ca7301"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 		hexutil.MustDecode("0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96"),
// 	}
// 	stateRoots := make([]common.Hash, len(rawRoots))
// 	for i := 0; i < len(rawRoots); i++ {
// 		stateRoots[i] = common.BytesToHash(rawRoots[i])
// 	}
// 	compute, err := prefixproofs.Root(stateRoots)
// 	Require(t, err)
// 	t.Logf("%#x and %d", compute, len(stateRoots))
// 	Fail(t, "oops")
// 	//
// 	// DONE WITH PREFIX PROOF COMPUTATION, commit {Height:16 Merkle:0x433f2814c890c651c5fe70c208c6d25436325bbcddb054706a784aa9551396f0 FirstLeaf:0xd781e3a672f2f16cbf9f3a2f4d2d9714ba2818f4b378870010d4c78b60ca7301 LastLeafProof:[0x0000000000000000000000000000000000000000000000000000000000000000 0x0000000000000000000000000000000000000000000000000000000000000000 0x0000000000000000000000000000000000000000000000000000000000000000 0x0000000000000000000000000000000000000000000000000000000000000000 0x76469513ad2bfb1ab0e25233daf50dcf2bf9c413a509ba3d9aca48fbc361aea4] FirstLeafProof:[0xc21068ed09a6bed80fa47668e8b04c83aac25b6e5c6d0c6eb0616a77b5dca2af 0xa5637fb3c6ac430e11175e34abcbb0b3caffba79ac15b9d8439a3b3722415263 0xf80f90ce52e1abc37ee458cbf22c839d7facba76c400f5e4e106d7ab68c38bbb 0xbb999b6b6b11cc690c159dad5dc5984a30b48200fe6ce2e681ebdcf0ff6f6a7c 0x9982027f45622ecf649efb8373312e366662a4badac4013083bafffeaef7cdf0] LastLeaf:0x2805bce9a15caab3f29c31391ec595da38aae54446080aae2043404cd8135b96}
// }

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

	_, l2nodeA, l2clientA, _, l1info, _, l1client, l1stack, assertionChain, stakeTokenAddr := createTestNodeOnL1ForBoldProtocol(t, ctx, true, nil, nil, l2chainConfig, nil, l2info)
	defer requireClose(t, l1stack)
	defer l2nodeA.StopAndWait()
	execNodeA := getExecNode(t, l2nodeA)

	l2clientB, l2nodeB, assertionChainB := create2ndNodeWithConfigForBoldProtocol(t, ctx, l2nodeA, l1stack, l1info, &l2info.ArbInitData, arbnode.ConfigDefaultL1Test(), gethexec.ConfigDefaultTest(), nil, stakeTokenAddr)
	defer l2nodeB.StopAndWait()
	execNodeB := getExecNode(t, l2nodeB)

	nodeAGenesis := execNodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBGenesis := execNodeB.Backend.APIBackend().CurrentHeader().Hash()
	if nodeAGenesis != nodeBGenesis {
		Fail(t, "node A L2 genesis hash", nodeAGenesis, "!= node B L2 genesis hash", nodeBGenesis)
	}
	bridgeBalancesToBoldL2s(t, "Faucet", big.NewInt(1).Mul(big.NewInt(params.Ether), big.NewInt(10000)), l1info, l2info, l1client, l2clientA, l2clientB, ctx)

	deployAuth := l1info.GetDefaultTransactOpts("RollupOwner", ctx)

	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	TransferBalance(t, "Faucet", "Asserter", balance, l1info, l1client, ctx)
	TransferBalance(t, "Faucet", "EvilAsserter", balance, l1info, l1client, ctx)
	l1authB := l1info.GetDefaultTransactOpts("EvilAsserter", ctx)

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

	valConfig := staker.L1ValidatorConfig{}
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

	statelessB, err := staker.NewStatelessBlockValidator(
		l2nodeB.InboxReader,
		l2nodeB.InboxTracker,
		l2nodeB.TxStreamer,
		execNodeB,
		l2nodeB.ArbDB,
		nil,
		StaticFetcherFrom(t, &blockValidatorConfig),
		valStack,
	)
	Require(t, err)
	err = statelessB.Start(ctx)
	Require(t, err)

	stateManager, err := staker.NewStateManager(
		statelessA,
		nil,
		smallStepChallengeLeafHeight,
		smallStepChallengeLeafHeight*bigStepChallengeLeafHeight,
		"/tmp/good",
	)
	Require(t, err)
	poster := assertions.NewPoster(
		assertionChain,
		stateManager,
		"good",
		time.Hour,
	)

	stateManagerB, err := staker.NewStateManager(
		statelessB,
		nil,
		smallStepChallengeLeafHeight,
		smallStepChallengeLeafHeight*bigStepChallengeLeafHeight,
		"/tmp/evil",
	)
	Require(t, err)
	chainB, err := solimpl.NewAssertionChain(
		ctx,
		assertionChain.RollupAddress(),
		&l1authB,
		l1client,
	)
	Require(t, err)
	posterB := assertions.NewPoster(
		chainB,
		stateManagerB,
		"evil",
		time.Hour,
	)

	t.Log("Sending a tx from faucet to L2 node A background user")
	l2info.GenerateAccount("BackgroundUser")
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, common.Big1, nil)
	err = l2clientA.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2clientA, tx)
	Require(t, err)

	t.Log("Sending a tx from faucet to L2 node B background user")
	l2info.Accounts["Faucet"].Nonce = 0
	tx = l2info.PrepareTx("Faucet", "BackgroundUser", l2info.TransferGas, common.Big2, nil)
	err = l2clientB.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2clientB, tx)
	Require(t, err)

	bcA, err := l2nodeA.InboxTracker.GetBatchCount()
	Require(t, err)
	bcB, err := l2nodeB.InboxTracker.GetBatchCount()
	Require(t, err)
	msgA, err := l2nodeA.InboxTracker.GetBatchMessageCount(bcA - 1)
	Require(t, err)
	msgB, err := l2nodeB.InboxTracker.GetBatchMessageCount(bcB - 1)
	Require(t, err)
	accA, err := l2nodeA.InboxTracker.GetBatchAcc(bcA - 1)
	Require(t, err)
	accB, err := l2nodeB.InboxTracker.GetBatchAcc(bcB - 1)
	Require(t, err)
	t.Logf("Node A, count %d, msgs %d, acc %s", bcA, msgA, accA)
	t.Logf("Node B, count %d, msgs %d, acc %s", bcB, msgB, accB)

	nodeALatest := execNodeA.Backend.APIBackend().CurrentHeader().Hash()
	nodeBLatest := execNodeB.Backend.APIBackend().CurrentHeader().Hash()
	if nodeALatest == nodeBLatest {
		Fail(t, "node A L2 hash", nodeALatest, "matches node B L2 hash", nodeBLatest)
	}

	t.Log("Honest party posting assertion at batch 1, pos 0")
	_, err = poster.PostAssertion(ctx)
	Require(t, err)

	t.Log("Honest party posting assertion at batch 2, pos 0")
	_, err = poster.PostAssertion(ctx)
	Require(t, err)

	t.Log("Evil party posting rival assertion at batch 2, pos 0")
	_, err = posterB.PostAssertion(ctx)
	Require(t, err)

	manager, err := challengemanager.New(
		ctx,
		assertionChain,
		l1client,
		stateManager,
		assertionChain.RollupAddress(),
		challengemanager.WithName("honest"),
		challengemanager.WithMode(modes.DefensiveMode),
		challengemanager.WithAssertionPostingInterval(time.Hour),
		challengemanager.WithAssertionScanningInterval(5*time.Second),
	)
	Require(t, err)
	manager.Start(ctx)

	managerB, err := challengemanager.New(
		ctx,
		chainB,
		l1client,
		stateManagerB,
		assertionChain.RollupAddress(),
		challengemanager.WithName("evil"),
		challengemanager.WithMode(modes.DefensiveMode),
		challengemanager.WithAssertionPostingInterval(time.Hour),
		challengemanager.WithAssertionScanningInterval(5*time.Second),
	)
	Require(t, err)
	managerB.Start(ctx)

	time.Sleep(time.Hour)

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
	assertionChain *solimpl.AssertionChain, stakeTokenAddr common.Address,
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
		challenge_testing.WithLevelZeroHeights(&challenge_testing.LevelZeroHeights{
			BlockChallengeHeight:     blockChallengeLeafHeight,
			BigStepChallengeHeight:   bigStepChallengeLeafHeight,
			SmallStepChallengeHeight: smallStepChallengeLeafHeight,
		}),
	)
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
	execConfig *gethexec.Config,
	stackConfig *node.Config,
	stakeTokenAddr common.Address,
) (*ethclient.Client, *arbnode.Node, *solimpl.AssertionChain) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if execConfig == nil {
		execConfig = gethexec.ConfigDefaultNonSequencerTest()
	}
	feedErrChan := make(chan error, 10)
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
	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)

	dataSigner := signature.DataSignerFromPrivateKey(l1info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	txOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
	firstExec := getExecNode(t, first)

	chainConfig := firstExec.ArbInterface.BlockChain().Config()
	addresses, assertionChain := deployContractsOnly(t, ctx, l1info, l1client, chainConfig.ChainID, stakeTokenAddr)

	l1info.SetContract("EvilBridge", addresses.Bridge)
	l1info.SetContract("EvilSequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("EvilInbox", addresses.Inbox)

	initMessage := getInitMessage(ctx, t, l1client, addresses)
	l2blockchain, err := gethexec.WriteOrTestBlockChain(l2chainDb, nil, initReader, chainConfig, initMessage, gethexec.ConfigDefaultTest().TxLookupLimit, 0)
	Require(t, err)

	AddDefaultValNode(t, ctx, nodeConfig, true)

	Require(t, execConfig.Validate())
	configFetcher := func() *gethexec.Config { return execConfig }
	currentExec, err := gethexec.CreateExecutionNode(ctx, l2stack, l2chainDb, l2blockchain, l1client, configFetcher)
	Require(t, err)

	execclient := execclient.NewClient(StaticFetcherFrom(t, &rpcclient.TestClientConfig), l2stack)

	currentNode, err := arbnode.CreateNode(ctx, l2stack, execclient, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client, addresses, &txOpts, &txOpts, dataSigner, feedErrChan)
	Require(t, err)

	Require(t, currentExec.Initialize(ctx))

	err = currentNode.Start(ctx)
	Require(t, err, nodeConfig.BlockValidator.ValidationServer.URL, " l2: ", l2stack.WSEndpoint())
	l2client := ClientForStack(t, l2stack)

	Require(t, currentExec.Start(ctx))

	StartWatchChanErr(t, ctx, feedErrChan, currentNode, currentExec)

	return l2client, currentNode, assertionChain
}

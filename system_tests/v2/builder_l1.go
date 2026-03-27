// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

// builder_l1.go contains the L1+L2 node lifecycle helpers.
// The core logic is ported from system_tests/common_test.go
// (createTestL1BlockChain + deployOnParentChain + buildOnParentChain).

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	boldsetup "github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	nitroinit "github.com/offchainlabs/nitro/cmd/nitro/init"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/statetransfer"
	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
	"github.com/offchainlabs/nitro/validator/server_common"
)

// L1Handle provides access to the running L1 node.
type L1Handle struct {
	Client  *ethclient.Client
	Backend *eth.Ethereum
	Stack   *node.Node
}

// DefaultChainAccounts are the standard accounts created on L1 for testing.
var DefaultChainAccounts = []string{"RollupOwner", "Sequencer", "Validator", "User"}

// buildL1L2Node creates a full L1+L2 test setup:
// 1. Creates an L1 geth dev chain with SimulatedBeacon
// 2. Deploys BOLD rollup contracts on L1
// 3. Creates L2 chain connected to L1
func buildL1L2Node(t *testing.T, ctx context.Context, spec *BuilderSpec) (*TestEnv, func()) {
	t.Helper()

	stateScheme := spec.resolvedScheme
	if stateScheme == "" {
		stateScheme = env.GetTestStateScheme()
	}

	// --- Step 1: Create L1 chain ---
	l1Info := arbtest.NewL1TestInfo(t)
	l1Info.GenerateAccount("Faucet")
	for _, acct := range DefaultChainAccounts {
		l1Info.GenerateAccount(acct)
	}

	l1StackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	l1StackConfig.DataDir = ""

	l1Stack, err := node.New(l1StackConfig)
	if err != nil {
		t.Fatalf("L1 node.New: %v", err)
	}

	l1ChainConfig := chaininfo.ArbitrumDevTestChainConfig()
	l1ChainConfig.ArbitrumChainParams = params.ArbitrumChainParams{}

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = l1ChainConfig.ChainID.Uint64()
	faucetAddr := l1Info.GetAddress("Faucet")
	l1Genesis := core.DeveloperGenesisBlock(15_000_000, &faucetAddr)

	bigBalance := new(big.Int).SetUint64(9223372036854775807)
	for _, acct := range DefaultChainAccounts {
		addr := l1Info.GetAddress(acct)
		l1Genesis.Alloc[addr] = types.Account{Balance: new(big.Int).Set(bigBalance)}
	}
	for acct, info := range l1Info.GetGenesisAlloc() {
		l1Genesis.Alloc[acct] = info
	}
	l1Genesis.BaseFee = new(big.Int).Mul(big.NewInt(50), big.NewInt(params.GWei))
	nodeConf.Genesis = l1Genesis
	nodeConf.Miner.Etherbase = faucetAddr
	nodeConf.Miner.PendingFeeRecipient = faucetAddr
	nodeConf.SyncMode = ethconfig.FullSync

	l1Backend, err := eth.New(l1Stack, &nodeConf)
	if err != nil {
		t.Fatalf("eth.New: %v", err)
	}

	simBeacon, err := catalyst.NewSimulatedBeacon(0, common.Address{}, l1Backend)
	if err != nil {
		t.Fatalf("NewSimulatedBeacon: %v", err)
	}
	catalyst.RegisterSimulatedBeaconAPIs(l1Stack, simBeacon)
	l1Stack.RegisterLifecycle(simBeacon)

	tempKeyStore := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	faucetAccount, err := tempKeyStore.ImportECDSA(l1Info.Accounts["Faucet"].PrivateKey, "passphrase")
	if err != nil {
		t.Fatalf("ImportECDSA: %v", err)
	}
	if err := tempKeyStore.Unlock(faucetAccount, "passphrase"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	l1Backend.AccountManager().AddBackend(tempKeyStore)

	l1Stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filters.NewFilterSystem(l1Backend.APIBackend, filters.Config{})),
	}})

	if err := l1Stack.Start(); err != nil {
		t.Fatalf("L1 stack.Start: %v", err)
	}

	l1Client := ethclient.NewClient(l1Stack.Attach())

	// --- Step 2: Deploy rollup contracts ---
	l2ChainConfig := chaininfo.ArbitrumDevTestChainConfig()
	if spec.resolvedArbOS != 0 {
		l2ChainConfig.ArbitrumChainParams.InitialArbOSVersion = spec.resolvedArbOS
	}

	addresses, initMessage := deployBoldRollup(t, ctx, l1Info, l1Client, l2ChainConfig)

	// --- Step 3: Build L2 connected to L1 ---
	l2NodeConfig := arbnode.ConfigDefaultL2Test()
	l2NodeConfig.ParentChainReader.Enable = true
	l2NodeConfig.BatchPoster.Enable = true

	l2ExecCfg := defaultExecConfig(t, stateScheme)
	l2StackCfg := testhelpers.CreateStackConfigForTest(t.TempDir())

	l2Info, l2Stack, executionDB, consensusDB, blockchain := createBlockChainWithInit(
		t, l2ChainConfig, l2StackCfg, l2ExecCfg, initMessage)

	seqTxOpts := l1Info.GetDefaultTransactOpts("Sequencer", ctx)
	execFetcher := &staticConfigFetcher[gethexec.Config]{cfg: l2ExecCfg}
	execNode, err := gethexec.CreateExecutionNode(ctx, l2Stack, executionDB, blockchain, l1Client, execFetcher, big.NewInt(1337), 0)
	if err != nil {
		t.Fatalf("CreateExecutionNode: %v", err)
	}

	fatalCh := make(chan error, 10)
	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("NewMachineLocator: %v", err)
	}
	nodeFetcher := &staticConfigFetcher[arbnode.Config]{cfg: l2NodeConfig}
	consensusNode, err := arbnode.CreateConsensusNode(
		ctx, l2Stack, execNode, consensusDB, nodeFetcher, blockchain.Config(),
		l1Client, addresses, &seqTxOpts, &seqTxOpts, nil, fatalCh,
		big.NewInt(1337), nil, locator.LatestWasmModuleRoot())
	if err != nil {
		t.Fatalf("CreateConsensusNode: %v", err)
	}

	cleanup, err := execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, l2Stack, execNode, consensusNode)
	if err != nil {
		t.Fatalf("InitAndStart: %v", err)
	}

	l2Client := ethclient.NewClient(l2Stack.Attach())

	// Make Owner a chain owner on L2.
	debugAuth := l2Info.GetDefaultTransactOpts("Owner", ctx)
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), l2Client)
	if err != nil {
		t.Fatalf("NewArbDebug: %v", err)
	}
	tx, err := arbDebug.BecomeChainOwner(&debugAuth)
	if err != nil {
		t.Fatalf("BecomeChainOwner: %v", err)
	}
	if _, err := waitForTxWithTimeout(ctx, l2Client, tx.Hash(), 10*time.Second); err != nil {
		t.Fatalf("BecomeChainOwner tx: %v", err)
	}

	go watchFatalChan(t, ctx, fatalCh)

	l1Handle := &L1Handle{
		Client:  l1Client,
		Backend: l1Backend,
		Stack:   l1Stack,
	}
	l2Handle := &L2Handle{
		Client:  l2Client,
		cleanup: cleanup,
	}

	testEnv := &TestEnv{
		T:      t,
		Ctx:    ctx,
		L1:     l1Handle,
		L1Info: l1Info,
		L2:     l2Handle,
		L2Info: l2Info,
		Spec:   spec,
	}

	return testEnv, func() {
		l2Handle.cleanup()
		l1Stack.Close()
	}
}

// deployBoldRollup deploys BOLD rollup contracts on L1.
func deployBoldRollup(
	t *testing.T,
	ctx context.Context,
	l1Info *arbtest.BlockchainTestInfo,
	l1Client *ethclient.Client,
	l2ChainConfig *params.ChainConfig,
) (*chaininfo.RollupAddresses, *arbostypes.ParsedInitMessage) {
	t.Helper()

	deployOpts := l1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	serializedChainConfig, err := json.Marshal(l2ChainConfig)
	if err != nil {
		t.Fatalf("marshal chainConfig: %v", err)
	}

	// Create header reader for L1.
	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
	parentChainReader, err := headerreader.New(ctx, l1Client, func() *headerreader.Config { return &headerreader.TestConfig }, arbSys)
	if err != nil {
		t.Fatalf("headerreader.New: %v", err)
	}
	parentChainReader.Start(ctx)
	defer parentChainReader.StopAndWait()

	// Deploy WETH9 as stake token.
	stakeToken, tx, _, err := localgen.DeployTestWETH9(&deployOpts, l1Client, "Weth", "WETH")
	if err != nil {
		t.Fatalf("DeployTestWETH9: %v", err)
	}
	if _, err := waitForTxWithTimeout(ctx, l1Client, tx.Hash(), 30*time.Second); err != nil {
		t.Fatalf("WETH9 deploy tx: %v", err)
	}

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("NewMachineLocator: %v", err)
	}

	cfg := rollupgen.Config{
		MiniStakeValues:        []*big.Int{big.NewInt(5), big.NewInt(4), big.NewInt(3), big.NewInt(2), big.NewInt(1)},
		ConfirmPeriodBlocks:    120,
		StakeToken:             stakeToken,
		BaseStake:              big.NewInt(1),
		WasmModuleRoot:         locator.LatestWasmModuleRoot(),
		Owner:                  deployOpts.From,
		LoserStakeEscrow:       deployOpts.From,
		MinimumAssertionPeriod: big.NewInt(75),
		ValidatorAfkBlocks:     201600,
		ChainId:                l2ChainConfig.ChainID,
		ChainConfig:            string(serializedChainConfig),
		SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
			DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
			FutureBlocks:  big.NewInt(12),
			DelaySeconds:  big.NewInt(60 * 60 * 24),
			FutureSeconds: big.NewInt(60 * 60),
		},
		LayerZeroBlockEdgeHeight:     big.NewInt(1 << 5),
		LayerZeroBigStepEdgeHeight:   big.NewInt(1 << 10),
		LayerZeroSmallStepEdgeHeight: big.NewInt(1 << 10),
		GenesisAssertionState: rollupgen.AssertionState{
			GlobalState:   rollupgen.GlobalState{},
			MachineStatus: 1,
		},
		GenesisInboxCount:          common.Big0,
		AnyTrustFastConfirmer:      common.Address{},
		NumBigStepLevel:            3,
		ChallengeGracePeriodBlocks: 3,
		BufferConfig: rollupgen.BufferConfig{
			Threshold:            300,
			Max:                  14400,
			ReplenishRateInBasis: 500,
		},
		DataCostEstimate: big.NewInt(0),
	}

	boldAddresses, err := boldsetup.DeployFullRollupStack(
		ctx,
		l1Client,
		&deployOpts,
		l1Info.GetAddress("Sequencer"),
		cfg,
		boldsetup.RollupStackConfig{},
	)
	if err != nil {
		t.Fatalf("DeployFullRollupStack: %v", err)
	}

	addresses := &chaininfo.RollupAddresses{
		Bridge:                 boldAddresses.Bridge,
		Inbox:                  boldAddresses.Inbox,
		SequencerInbox:         boldAddresses.SequencerInbox,
		Rollup:                 boldAddresses.Rollup,
		UpgradeExecutor:        boldAddresses.UpgradeExecutor,
		ValidatorUtils:         boldAddresses.ValidatorUtils,
		ValidatorWalletCreator: boldAddresses.ValidatorWalletCreator,
		StakeToken:             stakeToken,
		DeployedAt:             boldAddresses.DeployedAt,
	}

	initMessage, err := nitroinit.GetConsensusParsedInitMsg(ctx, true, l2ChainConfig.ChainID, l1Client, addresses, l2ChainConfig)
	if err != nil {
		t.Fatalf("GetConsensusParsedInitMsg: %v", err)
	}

	return addresses, initMessage
}

// createBlockChainWithInit is like createBlockChain but uses a ParsedInitMessage
// (from L1 deployment) instead of generating a fake one.
func createBlockChainWithInit(
	t *testing.T,
	chainConfig *params.ChainConfig,
	stackCfg *node.Config,
	execCfg *gethexec.Config,
	initMsg *arbostypes.ParsedInitMessage,
) (*arbtest.BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	t.Helper()

	l2Info := arbtest.NewArbTestInfo(t, chainConfig.ChainID)

	stack, err := node.New(stackCfg)
	if err != nil {
		t.Fatalf("node.New: %v", err)
	}

	var executionDB ethdb.Database
	if stackCfg.DBEngine == env.MemoryDB {
		executionDB = rawdb.WrapDatabaseWithWasm(rawdb.NewMemoryDatabase(), rawdb.NewMemoryDatabase())
	} else {
		chainData, err := stack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{
			MetricsNamespace:   "l2chaindata/",
			PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata"),
		})
		if err != nil {
			t.Fatalf("open l2chaindata: %v", err)
		}
		wasmData, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{
			MetricsNamespace:   "wasm/",
			PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("wasm"),
			NoFreezer:          true,
		})
		if err != nil {
			t.Fatalf("open wasm: %v", err)
		}
		executionDB = rawdb.WrapDatabaseWithWasm(chainData, wasmData)
	}

	var consensusDB ethdb.Database
	if stackCfg.DBEngine == env.MemoryDB {
		consensusDB = rawdb.NewMemoryDatabase()
	} else {
		consensusDB, err = stack.OpenDatabaseWithOptions("arbitrumdata", node.DatabaseOptions{
			MetricsNamespace:   "arbitrumdata/",
			PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("arbitrumdata"),
			NoFreezer:          true,
		})
		if err != nil {
			t.Fatalf("open arbitrumdata: %v", err)
		}
	}

	initReader := statetransfer.NewMemoryInitDataReader(&l2Info.ArbInitData)
	coreCacheConfig := gethexec.DefaultCacheConfigTrieNoFlushFor(&execCfg.Caching, false)
	blockchain, err := gethexec.WriteOrTestBlockChain(
		executionDB, coreCacheConfig, initReader, chainConfig, nil, nil, initMsg,
		&gethexec.ConfigDefault.TxIndexer, 0, execCfg.ExposeMultiGas)
	if err != nil {
		t.Fatalf("WriteOrTestBlockChain: %v", err)
	}

	return l2Info, stack, executionDB, consensusDB, blockchain
}

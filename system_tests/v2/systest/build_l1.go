// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	nitroinit "github.com/offchainlabs/nitro/cmd/nitro/init"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/deploy"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollup_legacy_gen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/server_common"
)

// defaultL1Accounts are funded at L1 genesis: RollupOwner deploys the rollup,
// Sequencer is batch poster + data signer, User drives the delayed inbox.
// Validator stakes in the full-stack topology.
var defaultL1Accounts = []string{"RollupOwner", "Sequencer", "Validator", "User"}

// maxL1DataSize bounds sequencer-inbox batch data on the parent chain.
const maxL1DataSize = 117964

// WithL1 builds the test on a real parent chain (L1 + sequencer L2 with batch
// posting and inbox reading) instead of the default L2-only node.
func WithL1() TestOption {
	return func(b *builder) {
		setTopology(b, TopologyL1L2, "WithL1")
	}
}

// setTopology pins the node layout, rejecting a second topology option.
func setTopology(b *builder, topo Topology, name string) {
	if b.topology == topo {
		panic(fmt.Sprintf("systest: %s applied twice", name))
	}
	if b.topology != TopologyL2Only {
		panic(fmt.Sprintf("systest: %s conflicts with another topology option", name))
	}
	b.topology = topo
}

// buildL1L2Node brings up an L1 parent chain, deploys the rollup, and starts a
// sequencer L2 wired to it. Returns Env (with L1 populated) and cleanup.
func buildL1L2Node(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	return buildL1Stack(t, ctx, spec, overrides, false)
}

// buildL1Stack builds the L1 + sequencer L2. disableValidatorWhitelist opens the
// rollup's validator whitelist at deploy (for the full-stack staker).
func buildL1Stack(t *testing.T, ctx context.Context, spec Spec, overrides overrides, disableValidatorWhitelist bool) (*Env, func()) {
	t.Helper()

	nodeConfig := cloneConfig(arbnode.ConfigDefaultL1Test())
	chainConfig, execCfg, stackCfg := seedConfigs(t, spec, overrides, nodeConfig)

	var rb rollbackGuard
	defer rb.run()

	// Phase 1: parent chain.
	l1Info, l1Client, l1Backend, l1Stack, l1BlobReader := createL1Chain(t)
	closeL1Chain := func() {
		l1Client.Close()
		closeStack("l1", l1Stack)
	}
	rb.stage(closeL1Chain)

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("NewMachineLocator: %v", err)
	}
	wasmModuleRoot := locator.LatestWasmModuleRoot()
	if wasmModuleRoot == (common.Hash{}) {
		t.Fatalf("no wasm module root found under target/machines; run `make build-replay-env`")
	}

	// Phase 2: deploy rollup, derive the real init message from the L1 inbox.
	addresses, initMsg := deployL1Rollup(t, ctx, l1Info, l1Client, chainConfig, wasmModuleRoot, disableValidatorWhitelist)

	// Phase 3: L2 chain seeded with the deployed init message.
	l2Info, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg, initMsg, spec.arbOSInit, overrides.InitData)
	rb.stage(func() { blockchain.Stop(); closeStack("l2", stack); closeL1Chain() })

	// Phase 4: parent chain reader for the consensus node. This reader's poll
	// loop is never started here; the node builds and owns its own started reader.
	nodeFetcher := newConfigFetcher(nodeConfig)
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
	if err != nil {
		t.Fatalf("NewArbSys: %v", err)
	}
	l1Reader, err := headerreader.New(ctx, l1Client, func() *headerreader.Config {
		return &nodeFetcher.Get().ParentChainReader
	}, arbSys)
	if err != nil {
		t.Fatalf("headerreader.New: %v", err)
	}
	parentChain := parent.NewParentChainWithConfig(ctx, simulatedParentChainID, l1Reader,
		func() *parent.Config { return &parent.TestConfig })

	// Phase 5: execution + consensus nodes.
	fatalCh := make(chan error, 10)
	execFetcher := newConfigFetcher(execCfg)
	execNode, err := gethexec.CreateExecutionNode(ctx, stack, executionDB, blockchain,
		containers.Some(l1Client), execFetcher, 0, parentChain, fatalCh)
	if err != nil {
		t.Fatalf("CreateExecutionNode: %v", err)
	}

	seqTxOpts := l1Info.GetDefaultTransactOpts("Sequencer", ctx)
	dataSigner := signature.DataSignerFromPrivateKey(l1Info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	consensusNode, err := arbnode.CreateConsensusNode(
		ctx, stack, execNode, consensusDB, nodeFetcher, blockchain.Config(), l1Client,
		addresses, nil, &seqTxOpts, dataSigner, fatalCh, l1BlobReader, wasmModuleRoot, parentChain)
	if err != nil {
		t.Fatalf("CreateConsensusNode: %v", err)
	}

	e := &Env{
		t:    t,
		Ctx:  ctx,
		Spec: spec,
	}
	l2Handle := &L2Handle{
		ChainHandle: ChainHandle{Info: l2Info, e: e, name: "l2"},
		Stack:       stack,
		ExecNode:    execNode,
		Consensus:   consensusNode,
	}
	fullCleanup := startL2Node(t, ctx, &rb, l2Handle, fatalCh, closeL1Chain)

	if !spec.SkipChainOwner {
		becomeChainOwner(t, ctx, l2Handle.Client, l2Info)
	}

	l1Handle := &L1Handle{
		ChainHandle: ChainHandle{Client: l1Client, Info: l1Info, e: e, name: "l1"},
		Backend:     l1Backend,
		Stack:       l1Stack,
		blobReader:  l1BlobReader,
		initMsg:     initMsg,
		wasmRoot:    wasmModuleRoot,
	}

	e.L2 = l2Handle
	e.L1 = l1Handle

	rb.commit()
	return e, fullCleanup
}

// createL1Chain starts a geth devnet (PoS via SimulatedBeacon) funding the
// rollup accounts at genesis. Mirrors v1 createTestL1BlockChain.
func createL1Chain(t *testing.T) (*arbtest.BlockchainTestInfo, *ethclient.Client, *eth.Ethereum, *node.Node, containers.Option[daprovider.BlobReader]) {
	t.Helper()

	l1Info := arbtest.NewL1TestInfo(t)
	l1Info.GenerateAccount("Faucet")
	for _, acct := range defaultL1Accounts {
		l1Info.GenerateAccount(acct)
	}

	stackCfg := testhelpers.CreateStackConfigForTest(t.TempDir())
	stackCfg.DataDir = ""
	stack, err := node.New(stackCfg)
	if err != nil {
		t.Fatalf("L1 node.New: %v", err)
	}
	started := false
	defer func() {
		if !started {
			closeStack("l1", stack)
		}
	}()

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = simulatedParentChainID.Uint64()
	faucetAddr := l1Info.GetAddress("Faucet")
	l1Genesis := core.DeveloperGenesisBlock(15_000_000, &faucetAddr)

	bigBalance := new(big.Int).SetUint64(math.MaxInt64)
	for _, acct := range defaultL1Accounts {
		addr := l1Info.GetAddress(acct)
		if l1Genesis.Alloc[addr].Balance == nil {
			l1Genesis.Alloc[addr] = types.Account{Balance: new(big.Int).Set(bigBalance)}
		} else {
			l1Genesis.Alloc[addr].Balance.Add(l1Genesis.Alloc[addr].Balance, bigBalance)
		}
	}
	l1Genesis.BaseFee = big.NewInt(50 * params.GWei)
	nodeConf.Genesis = l1Genesis
	nodeConf.Miner.Etherbase = faucetAddr
	nodeConf.Miner.PendingFeeRecipient = faucetAddr
	nodeConf.SyncMode = ethconfig.FullSync

	l1Backend, err := eth.New(stack, &nodeConf)
	if err != nil {
		t.Fatalf("eth.New: %v", err)
	}

	simBeacon, err := catalyst.NewSimulatedBeacon(0, common.Address{}, l1Backend)
	if err != nil {
		t.Fatalf("NewSimulatedBeacon: %v", err)
	}
	if err := simBeacon.Initialize(context.Background()); err != nil {
		t.Fatalf("SimulatedBeacon.Initialize: %v", err)
	}
	catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
	stack.RegisterLifecycle(simBeacon)

	tempKeyStore := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	faucetAccount, err := tempKeyStore.ImportECDSA(l1Info.Accounts["Faucet"].PrivateKey, "passphrase")
	if err != nil {
		t.Fatalf("ImportECDSA: %v", err)
	}
	if err := tempKeyStore.Unlock(faucetAccount, "passphrase"); err != nil {
		t.Fatalf("keystore Unlock: %v", err)
	}
	l1Backend.AccountManager().AddBackend(tempKeyStore)

	// eth.New already registered l1Backend as a stack lifecycle; a manual Stop
	// would double-close the peer-dropper channel.
	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filters.NewFilterSystem(l1Backend.APIBackend, filters.Config{})),
	}})
	stack.RegisterAPIs(tracers.APIs(l1Backend.APIBackend))

	if err := stack.Start(); err != nil {
		t.Fatalf("L1 stack.Start: %v", err)
	}

	started = true
	l1Client := ethclient.NewClient(stack.Attach())
	return l1Info, l1Client, l1Backend, stack, containers.Some[daprovider.BlobReader](simBeacon)
}

// deployL1Rollup deploys the legacy (non-BOLD) rollup contracts and returns the
// rollup addresses plus the parsed init message read back from the L1 inbox.
// Mirrors the legacy branch of v1 deployOnParentChain.
func deployL1Rollup(
	t *testing.T,
	ctx context.Context,
	l1Info *arbtest.BlockchainTestInfo,
	l1Client *ethclient.Client,
	chainConfig *params.ChainConfig,
	wasmModuleRoot common.Hash,
	disableValidatorWhitelist bool,
) (*chaininfo.RollupAddresses, *arbostypes.ParsedInitMessage) {
	t.Helper()

	rollupOwnerOpts := l1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	serializedChainConfig, err := json.Marshal(chainConfig)
	if err != nil {
		t.Fatalf("marshal chainConfig: %v", err)
	}

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
	if err != nil {
		t.Fatalf("NewArbSys: %v", err)
	}
	readerCfg := headerreader.TestConfig
	l1Reader, err := headerreader.New(ctx, l1Client, func() *headerreader.Config { return &readerCfg }, arbSys)
	if err != nil {
		t.Fatalf("deploy headerreader.New: %v", err)
	}
	l1Reader.Start(ctx)
	defer l1Reader.StopAndWait()

	addresses, err := deploy.DeployLegacyOnParentChain(
		ctx,
		l1Reader,
		&rollupOwnerOpts,
		[]common.Address{l1Info.GetAddress("Sequencer")},
		l1Info.GetAddress("RollupOwner"),
		0,
		deploy.GenerateLegacyRollupConfig(false, wasmModuleRoot, l1Info.GetAddress("RollupOwner"), chainConfig, serializedChainConfig, common.Address{}),
		common.Address{},
		big.NewInt(maxL1DataSize),
		true,
	)
	if err != nil {
		t.Fatalf("DeployLegacyOnParentChain: %v", err)
	}

	l1Info.SetContract("Bridge", addresses.Bridge)
	l1Info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1Info.SetContract("Inbox", addresses.Inbox)
	l1Info.SetContract("UpgradeExecutor", addresses.UpgradeExecutor)

	// Open the validator whitelist and drop the min assertion period for the staker.
	if disableValidatorWhitelist {
		executeRollupAdmin(t, ctx, l1Info, l1Client, l1Reader, addresses, "setValidatorWhitelistDisabled", true)
		executeRollupAdmin(t, ctx, l1Info, l1Client, l1Reader, addresses, "setMinimumAssertionPeriod", big.NewInt(1))
	}

	initMsg, err := nitroinit.GetConsensusParsedInitMsg(ctx, true, chainConfig.ChainID, l1Client, addresses, chainConfig)
	if err != nil {
		t.Fatalf("GetConsensusParsedInitMsg: %v", err)
	}
	return addresses, initMsg
}

// executeRollupAdmin calls a RollupAdminLogic method through the UpgradeExecutor
// (which owns rollup admin), signed by RollupOwner, and waits for inclusion.
func executeRollupAdmin(t *testing.T, ctx context.Context, l1Info *arbtest.BlockchainTestInfo, l1Client *ethclient.Client, l1Reader *headerreader.HeaderReader, addresses *chaininfo.RollupAddresses, method string, args ...any) {
	t.Helper()
	rollupABI, err := rollup_legacy_gen.RollupAdminLogicMetaData.GetAbi()
	if err != nil {
		t.Fatalf("rollup admin abi: %v", err)
	}
	calldata, err := rollupABI.Pack(method, args...)
	if err != nil {
		t.Fatalf("pack %s: %v", method, err)
	}
	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(addresses.UpgradeExecutor, l1Client)
	if err != nil {
		t.Fatalf("NewUpgradeExecutor: %v", err)
	}
	ownerOpts := l1Info.GetDefaultTransactOpts("RollupOwner", ctx)
	tx, err := upgradeExecutor.ExecuteCall(&ownerOpts, addresses.Rollup, calldata)
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	if _, err := l1Reader.WaitForTxApproval(ctx, tx); err != nil {
		t.Fatalf("%s tx: %v", method, err)
	}
}

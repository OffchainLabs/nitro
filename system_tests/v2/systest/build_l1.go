// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// defaultL1Accounts are funded at L1 genesis: RollupOwner deploys the rollup,
// Sequencer is batch poster + data signer, User drives the delayed inbox.
// Validator stakes in the full-stack topology.
var defaultL1Accounts = []string{"RollupOwner", "Sequencer", "Validator", "User"}

// maxL1DataSize bounds sequencer-inbox batch data on the parent chain.
const maxL1DataSize = 117964

// simulatedParentChainID matches geth's DeveloperGenesisBlock chain id (1337).
var simulatedParentChainID = big.NewInt(1337)

// sequencerCredentials derives the batch-poster transactor and feed data signer
// from the funded Sequencer account on the parent chain.
func sequencerCredentials(ctx context.Context, parentInfo *arbtest.BlockchainTestInfo) (*bind.TransactOpts, signature.DataSignerFunc) {
	opts := parentInfo.GetDefaultTransactOpts("Sequencer", ctx)
	return &opts, signature.DataSignerFromPrivateKey(parentInfo.GetInfoWithPrivKey("Sequencer").PrivateKey)
}

// buildL1L2Node brings up an L1 parent chain, deploys the rollup, and starts a
// sequencer L2 wired to it. Returns Env (with L1 populated) and cleanup.
func buildL1L2Node(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	t.Helper()

	nodeConfig, chainConfig, execCfg, stackCfg := seedConfigs(t, spec, overrides, arbnode.ConfigDefaultL1Test())

	var rb rollbackGuard
	defer rb.run()

	l1Info, l1Client, l1Backend, l1Stack, l1BlobReader := createL1Chain(t)
	closeL1Chain := func() {
		l1Client.Close()
		closeStack("l1", l1Stack)
	}
	rb.stage(closeL1Chain)

	wasmModuleRoot := latestWasmModuleRoot(t)
	if wasmModuleRoot == (common.Hash{}) {
		t.Fatalf("no wasm module root found under target/machines; run `make build-replay-env`")
	}

	addresses, initMsg := deployRollup(t, ctx, l1Info, l1Client, chainConfig, wasmModuleRoot)

	nodeFetcher := newConfigFetcher(nodeConfig)
	// l1Reader's poll loop is never started here; the consensus node builds and
	// owns its own started reader.
	l1Reader := newL1Reader(t, ctx, l1Client, func() *headerreader.Config {
		return &nodeFetcher.Get().ParentChainReader
	})
	parentChain := parent.NewParentChainWithConfig(ctx, simulatedParentChainID, l1Reader,
		func() *parent.Config { return &parent.TestConfig })

	l2Info, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg, initMsg, spec.arbOSInit, overrides.InitData)
	rb.stage(func() { blockchain.Stop(); closeStack("l2", stack); closeL1Chain() })

	execNode, fatalCh := newExecNode(t, ctx, "l2", stack, executionDB, blockchain, execCfg, l1Client, parentChain)
	seqTxOpts, dataSigner := sequencerCredentials(ctx, l1Info)
	consensusNode := newConsensusNode(t, ctx, "l2", stack, execNode, consensusDB, nodeConfig, blockchain.Config(),
		l1Client, addresses, nil, seqTxOpts, dataSigner, fatalCh, l1BlobReader, wasmModuleRoot, parentChain)

	e := newEnv(t, ctx, spec)
	l2Handle, fullCleanup := startNode(t, ctx, &rb, e, "l2", l2Info, stack, execNode, consensusNode, fatalCh, closeL1Chain)
	becomeChainOwner(t, ctx, spec, l2Handle.Client, l2Info)

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

// newL1Reader builds a parent-chain header reader over l1Client.
func newL1Reader(t *testing.T, ctx context.Context, l1Client *ethclient.Client, cfg func() *headerreader.Config) *headerreader.HeaderReader {
	t.Helper()
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, l1Client)
	if err != nil {
		t.Fatalf("NewArbSys: %v", err)
	}
	l1Reader, err := headerreader.New(ctx, l1Client, cfg, arbSys)
	if err != nil {
		t.Fatalf("headerreader.New: %v", err)
	}
	return l1Reader
}

// createL1Chain starts a geth devnet (PoS via SimulatedBeacon) funding the
// rollup accounts at genesis.
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

// deployRollup deploys the legacy (non-BOLD) rollup contracts and returns the
// rollup addresses plus the parsed init message read back from the parent chain's inbox.
func deployRollup(
	t *testing.T,
	ctx context.Context,
	parentInfo *arbtest.BlockchainTestInfo,
	parentClient *ethclient.Client,
	chainConfig *params.ChainConfig,
	wasmModuleRoot common.Hash,
) (*chaininfo.RollupAddresses, *arbostypes.ParsedInitMessage) {
	t.Helper()

	rollupOwnerOpts := parentInfo.GetDefaultTransactOpts("RollupOwner", ctx)
	serializedChainConfig, err := json.Marshal(chainConfig)
	if err != nil {
		t.Fatalf("marshal chainConfig: %v", err)
	}

	readerCfg := headerreader.TestConfig
	reader := newL1Reader(t, ctx, parentClient, func() *headerreader.Config { return &readerCfg })
	reader.Start(ctx)
	defer reader.StopAndWait()

	addresses, err := deploy.DeployLegacyOnParentChain(
		ctx,
		reader,
		&rollupOwnerOpts,
		[]common.Address{parentInfo.GetAddress("Sequencer")},
		parentInfo.GetAddress("RollupOwner"),
		0,
		deploy.GenerateLegacyRollupConfig(false, wasmModuleRoot, parentInfo.GetAddress("RollupOwner"), chainConfig, serializedChainConfig, common.Address{}),
		common.Address{},
		big.NewInt(maxL1DataSize),
		true,
	)
	if err != nil {
		t.Fatalf("DeployLegacyOnParentChain: %v", err)
	}

	parentInfo.SetContract("Bridge", addresses.Bridge)
	parentInfo.SetContract("SequencerInbox", addresses.SequencerInbox)
	parentInfo.SetContract("Inbox", addresses.Inbox)
	parentInfo.SetContract("UpgradeExecutor", addresses.UpgradeExecutor)

	initMsg, err := nitroinit.GetConsensusParsedInitMsg(ctx, true, chainConfig.ChainID, parentClient, addresses, chainConfig)
	if err != nil {
		t.Fatalf("GetConsensusParsedInitMsg: %v", err)
	}
	return addresses, initMsg
}

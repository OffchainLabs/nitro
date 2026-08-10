// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/statetransfer"
	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
	"github.com/offchainlabs/nitro/validator/server_common"
)

// overrides bundles escape-hatch mutators applied to default configs before
// the L2 node is built. Populated by WithExecConfigOverride etc. and carried
// schedule→buildNode. Each slice runs in registration order.
type overrides struct {
	Node        []func(*arbnode.Config)
	Exec        []func(*gethexec.Config)
	Stack       []func(*node.Config)
	InitData    []func(*statetransfer.ArbosInitializationInfo)
	ChainConfig []func(*params.ChainConfig)
}

// buildNode constructs the node layout selected by spec.Topology and returns the
// scenario Env plus a cleanup the runner owns.
func buildNode(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	t.Helper()
	switch spec.Topology {
	case TopologyL2Only:
		return buildL2Node(t, ctx, spec, overrides)
	case TopologyL1L2:
		return buildL1L2Node(t, ctx, spec, overrides)
	case TopologyMultiNode:
		return buildMultiNode(t, ctx, spec, overrides)
	default:
		t.Fatalf("systest: unknown topology %d", spec.Topology)
		return nil, nil
	}
}

// chainConfigForSpec returns the dev-test chain config with the spec's ArbOS
// version pinned (when set).
func chainConfigForSpec(spec Spec) *params.ChainConfig {
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	if spec.ArbOSVersion.IsSome() {
		chainConfig.ArbitrumChainParams.InitialArbOSVersion = spec.ArbOSVersion.Unwrap()
	}
	return chainConfig
}

// testStackConfig builds the L2 geth stack config for a spec.
func testStackConfig(t *testing.T, spec Spec) *node.Config {
	cfg := testhelpers.CreateStackConfigForTest(t.TempDir())
	if spec.DBEngine.IsSome() {
		cfg.DBEngine = string(spec.DBEngine.Unwrap())
	}
	// When the spec opts in, enable HTTP + WS endpoints on auto-assigned ports
	// so tests can dial them via L2Handle.HTTPClient / WSClient. overrides can override.
	if spec.ExposeRPC {
		cfg.HTTPHost = "127.0.0.1"
		cfg.WSHost = "127.0.0.1"
	}
	return cfg
}

func seedConfigs(t *testing.T, spec Spec, o overrides, nodeConfig *arbnode.Config) (*params.ChainConfig, *gethexec.Config, *node.Config) {
	chainConfig := chainConfigForSpec(spec)
	for _, f := range o.ChainConfig {
		f(chainConfig)
	}
	execCfg := defaultExecConfig(t, spec.StateScheme)
	stackCfg := testStackConfig(t, spec)
	applyOverrides(o, nodeConfig, execCfg, stackCfg)
	if *testflag.ConsensusExecutionInSameProcessUseRPC {
		configureConsensusExecutionOverRPC(execCfg, nodeConfig, stackCfg)
	}
	return chainConfig, execCfg, stackCfg
}

// applyOverrides runs the registered config mutators in order.
func applyOverrides(o overrides, nodeCfg *arbnode.Config, execCfg *gethexec.Config, stackCfg *node.Config) {
	for _, f := range o.Node {
		f(nodeCfg)
	}
	for _, f := range o.Exec {
		f(execCfg)
	}
	for _, f := range o.Stack {
		f(stackCfg)
	}
}

// l2Clients attaches the in-process stack client plus dials the HTTP/WS
// endpoints when rpc is set, returning a combined closer.
func l2Clients(t *testing.T, stack *node.Node, rpc bool) (client, httpClient, wsClient *ethclient.Client, closeClients func()) {
	client = ethclient.NewClient(stack.Attach())
	closeClients = func() {
		client.Close()
		if httpClient != nil {
			httpClient.Close()
		}
		if wsClient != nil {
			wsClient.Close()
		}
	}
	if rpc {
		var rb rollbackGuard
		defer rb.run()
		rb.stage(closeClients)
		httpClient = dialOptional(t, stack.HTTPEndpoint())
		wsClient = dialOptional(t, stack.WSEndpoint())
		rb.commit()
	}
	return client, httpClient, wsClient, closeClients
}

// startL2Node starts h's nodes, wires the fatal watcher and clients into h,
// and returns the full cleanup.
func startL2Node(t *testing.T, ctx context.Context, rb *rollbackGuard, h *L2Handle, fatalCh chan error) func() {
	t.Helper()
	cleanup, err := execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, h.Stack, h.ExecNode, h.Consensus)
	if err != nil {
		h.Consensus.StopAndWait()
		h.ExecNode.StopAndWait()
		for len(fatalCh) > 0 {
			t.Logf("%s fatal during InitAndStart: %v", h.name, <-fatalCh)
		}
		t.Fatalf("%s InitAndStart: %v", h.name, err)
	}
	// Start the fatal watcher as soon as the nodes run, so a fatal during client
	// setup or becomeChainOwner is reported rather than buffered then possibly lost.
	stopWatcher := startFatalWatcher(ctx, t, fatalCh)
	rb.stage(func() { cleanup(); stopWatcher() })

	var closeClients func()
	h.Client, h.HTTPClient, h.WSClient, closeClients = l2Clients(t, h.Stack, h.e.Spec.ExposeRPC)
	// Stop nodes (joins fatalCh senders) → drain watcher → close clients.
	h.cleanup = func() {
		cleanup()
		stopWatcher()
		closeClients()
	}
	rb.stage(h.cleanup)
	return h.cleanup
}

// becomeChainOwner makes the Owner account a chain owner — required by most tests.
func becomeChainOwner(t *testing.T, ctx context.Context, client *ethclient.Client, l2Info *arbtest.BlockchainTestInfo) {
	t.Helper()
	debugAuth := l2Info.GetDefaultTransactOpts("Owner", ctx)
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, client)
	if err != nil {
		t.Fatalf("NewArbDebug: %v", err)
	}
	tx, err := arbDebug.BecomeChainOwner(&debugAuth)
	if err != nil {
		t.Fatalf("BecomeChainOwner: %v", err)
	}
	if _, err := ensureTxSucceededWithin(ctx, client, tx, DefaultSetupTxTimeout); err != nil {
		t.Fatal(err)
	}
}

type rollbackGuard struct {
	fn   func()
	done bool
}

func (r *rollbackGuard) stage(fn func()) { r.fn = fn }
func (r *rollbackGuard) commit()         { r.done = true }
func (r *rollbackGuard) run() {
	if !r.done && r.fn != nil {
		r.fn()
	}
}

// closeStack closes a node stack on teardown, logging (not dropping) any close
// error so a leaked port/datadir is diagnosable. Matches InitAndStart's cleanup,
// which logs rather than fails — a benign close shouldn't flake a passing test.
func closeStack(name string, stack *node.Node) {
	if err := stack.Close(); err != nil {
		log.Printf("systest: %s stack close: %v", name, err)
	}
}

// startFatalWatcher drains consensus-node fatals on a goroutine until the
// returned stopper is called.
func startFatalWatcher(ctx context.Context, t *testing.T, fatalCh <-chan error) func() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go watchFatalChan(ctx, t, done, fatalCh, &wg)
	return func() {
		close(done)
		wg.Wait()
	}
}

// buildGenericNode builds and starts one L2 node from explicit wiring; nil L1
// pieces mean L2-only. info nil = the node's own fresh test info.
func buildGenericNode(
	t *testing.T,
	ctx context.Context,
	e *Env,
	spec Spec,
	o overrides,
	name string,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
	execCfg *gethexec.Config,
	stackCfg *node.Config,
	initMsg *arbostypes.ParsedInitMessage,
	info *arbtest.BlockchainTestInfo,
	l1Client *ethclient.Client,
	l1Info *arbtest.BlockchainTestInfo,
	parentChain *parent.ParentChain,
	addresses *chaininfo.RollupAddresses,
	blobReader containers.Option[daprovider.BlobReader],
	wasmRoot common.Hash,
) (*L2Handle, func()) {
	t.Helper()

	var rb rollbackGuard
	defer rb.run()

	l2Info, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg, initMsg, spec.arbOSInit, o.InitData)
	rb.stage(func() { blockchain.Stop(); closeStack(name, stack) })

	fatalCh := make(chan error, 10)
	execFetcher := newConfigFetcher(execCfg)
	l1ClientOpt := containers.None[*ethclient.Client]()
	if l1Client != nil {
		l1ClientOpt = containers.Some(l1Client)
	}
	execNode, err := gethexec.CreateExecutionNode(ctx, stack, executionDB, blockchain,
		l1ClientOpt, execFetcher, 0, parentChain, fatalCh)
	if err != nil {
		t.Fatalf("%s CreateExecutionNode: %v", name, err)
	}

	// The sequencer account posts batches and signs feed data on L1 topologies.
	var seqTxOpts *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	if l1Info != nil {
		opts := l1Info.GetDefaultTransactOpts("Sequencer", ctx)
		seqTxOpts = &opts
		dataSigner = signature.DataSignerFromPrivateKey(l1Info.GetInfoWithPrivKey("Sequencer").PrivateKey)
	}
	nodeFetcher := newConfigFetcher(nodeConfig)
	consensusNode, err := arbnode.CreateConsensusNode(
		ctx, stack, execNode, consensusDB, nodeFetcher, blockchain.Config(), l1Client,
		addresses, nil, seqTxOpts, dataSigner, fatalCh, blobReader, wasmRoot, parentChain)
	if err != nil {
		t.Fatalf("%s CreateConsensusNode: %v", name, err)
	}

	if initMsg == nil {
		if err := consensusNode.TxStreamer.AddFakeInitMessage(); err != nil {
			t.Fatalf("AddFakeInitMessage: %v", err)
		}
	}

	if info == nil {
		info = l2Info
	}
	handle := &L2Handle{
		ChainHandle: ChainHandle{Info: info, e: e, name: name},
		Stack:       stack,
		ExecNode:    execNode,
		Consensus:   consensusNode,
	}
	fullCleanup := startL2Node(t, ctx, &rb, handle, fatalCh)

	rb.commit()
	return handle, fullCleanup
}

// buildL2Node spins up an L2-only node per spec. Returns Env and cleanup.
func buildL2Node(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	t.Helper()

	nodeConfig := cloneConfig(arbnode.ConfigDefaultL2Test())
	chainConfig, execCfg, stackCfg := seedConfigs(t, spec, overrides, nodeConfig)

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("NewMachineLocator: %v", err)
	}

	var rb rollbackGuard
	defer rb.run()

	e := &Env{
		t:    t,
		Ctx:  ctx,
		Spec: spec,
	}
	handle, fullCleanup := buildGenericNode(t, ctx, e, spec, overrides, "l2",
		nodeConfig, chainConfig, execCfg, stackCfg,
		nil, nil, nil, nil, nil, nil,
		containers.None[daprovider.BlobReader](), locator.LatestWasmModuleRoot())
	rb.stage(fullCleanup)

	if !spec.SkipChainOwner {
		becomeChainOwner(t, ctx, handle.Client, handle.Info)
	}

	e.L2 = handle

	rb.commit()
	return e, fullCleanup
}

// createBlockChain builds the L2 stack, DBs, and blockchain. A nil initMsg
// synthesizes the L2-only fake message; L1 topologies pass the real one.
func createBlockChain(
	t *testing.T,
	chainConfig *params.ChainConfig,
	stackCfg *node.Config,
	execCfg *gethexec.Config,
	initMsg *arbostypes.ParsedInitMessage,
	arbOSInit *params.ArbOSInit,
	initData []func(*statetransfer.ArbosInitializationInfo),
) (*arbtest.BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	t.Helper()

	var stack *node.Node
	var rb rollbackGuard
	defer rb.run()

	l2Info := arbtest.NewArbTestInfo(t, chainConfig.ChainID)
	for _, f := range initData {
		f(&l2Info.ArbInitData)
	}

	var err error
	stack, err = node.New(stackCfg)
	if err != nil {
		t.Fatalf("node.New: %v", err)
	}
	rb.stage(func() { closeStack("l2", stack) })

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

	if initMsg == nil {
		serializedChainConfig, err := json.Marshal(chainConfig)
		if err != nil {
			t.Fatalf("marshal chainConfig: %v", err)
		}
		initMsg = &arbostypes.ParsedInitMessage{
			ChainId:               chainConfig.ChainID,
			InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
			ChainConfig:           chainConfig,
			SerializedChainConfig: serializedChainConfig,
		}
	}

	initReader := statetransfer.NewMemoryInitDataReader(&l2Info.ArbInitData)
	coreCacheConfig := gethexec.DefaultCacheConfigWithExtraFor(&execCfg.Caching, false, false)
	blockchain, err := gethexec.WriteOrTestBlockChain(
		executionDB, coreCacheConfig, initReader, chainConfig, arbOSInit, nil, initMsg,
		&gethexec.ConfigDefault.TxIndexer, 0, execCfg.ExposeMultiGas)
	if err != nil {
		t.Fatalf("WriteOrTestBlockChain: %v", err)
	}

	rb.commit()
	return l2Info, stack, executionDB, consensusDB, blockchain
}

// watchFatalChan surfaces consensus-node fatals via t.Errorf. Exits after
// done is closed and ch is drained.
func watchFatalChan(ctx context.Context, t *testing.T, done <-chan struct{}, ch <-chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	report := func(err error) {
		// Suppress ctx errors only during teardown (ctx done); a fatal wrapping a
		// deadline while the test is live is real.
		if suppressedAtShutdown(ctx, err) {
			return
		}
		t.Errorf("fatal error from consensus node: %v", err)
	}
	for {
		select {
		case <-done:
			for {
				select {
				case err := <-ch:
					report(err)
				default:
					return
				}
			}
		case err := <-ch:
			report(err)
		}
	}
}

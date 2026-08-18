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
		return buildMultiNodeStack(t, ctx, spec, overrides)
	default:
		t.Fatalf("systest: unknown topology %d", spec.Topology)
		return nil, nil
	}
}

// buildL2Node spins up an L2-only node, with no parent chain (TopologyL2Only).
func buildL2Node(t *testing.T, ctx context.Context, spec Spec, overrides overrides) (*Env, func()) {
	t.Helper()

	nodeConfig, chainConfig, execCfg, stackCfg := seedConfigs(t, spec, overrides, arbnode.ConfigDefaultL2Test())

	var rb rollbackGuard
	defer rb.run()

	l2Info, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg, nil, spec.arbOSInit, overrides.InitData)
	rb.stage(func() { blockchain.Stop(); closeStack("l2", stack) })

	execNode, fatalCh := newExecNode(t, ctx, "l2", stack, executionDB, blockchain, execCfg, nil, nil)
	consensusNode := newConsensusNode(t, ctx, "l2", stack, execNode, consensusDB, nodeConfig, blockchain.Config(),
		nil, nil, nil, nil, nil, fatalCh, containers.None[daprovider.BlobReader](), latestWasmModuleRoot(t), nil)

	if err := consensusNode.TxStreamer.AddFakeInitMessage(); err != nil {
		t.Fatalf("AddFakeInitMessage: %v", err)
	}

	e := newEnv(t, ctx, spec)
	handle, fullCleanup := startNode(t, ctx, &rb, e, "l2", l2Info, stack, execNode, consensusNode, fatalCh, nil)
	becomeChainOwner(t, ctx, spec, handle.Client, l2Info)
	e.L2 = handle

	rb.commit()
	return e, fullCleanup
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

// stackConfigForSpec builds the L2 geth stack config for a spec.
func stackConfigForSpec(t *testing.T, spec Spec) *node.Config {
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

func seedConfigs(t *testing.T, spec Spec, o overrides, defaultNodeConfig *arbnode.Config) (*arbnode.Config, *params.ChainConfig, *gethexec.Config, *node.Config) {
	nodeConfig := cloneConfig(defaultNodeConfig)
	chainConfig := chainConfigForSpec(spec)
	for _, f := range o.ChainConfig {
		f(chainConfig)
	}
	execCfg := defaultExecConfig(t, spec.StateScheme)
	stackCfg := stackConfigForSpec(t, spec)
	applyOverrides(o, nodeConfig, execCfg, stackCfg)
	if *testflag.ConsensusExecutionInSameProcessUseRPC {
		configureConsensusExecutionOverRPC(execCfg, nodeConfig, stackCfg)
	}
	return nodeConfig, chainConfig, execCfg, stackCfg
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

	memory := stackCfg.DBEngine == env.MemoryDB
	executionDB := rawdb.WrapDatabaseWithWasm(
		openStackDB(t, stack, memory, "l2chaindata", false),
		openStackDB(t, stack, memory, "wasm", true))
	consensusDB := openStackDB(t, stack, memory, "arbitrumdata", true)

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

// openStackDB opens a named stack database, or a fresh in-memory one when memory is set.
func openStackDB(t *testing.T, stack *node.Node, memory bool, name string, noFreezer bool) ethdb.Database {
	t.Helper()
	if memory {
		return rawdb.NewMemoryDatabase()
	}
	db, err := stack.OpenDatabaseWithOptions(name, node.DatabaseOptions{
		MetricsNamespace:   name + "/",
		PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions(name),
		NoFreezer:          noFreezer,
	})
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	return db
}

// newExecNode creates the execution node and its fatal-error channel, wiring
// the optional parent-chain client. Fails the test on error.
func newExecNode(t *testing.T, ctx context.Context, name string, stack *node.Node, executionDB ethdb.Database, blockchain *core.BlockChain, execCfg *gethexec.Config, parentClient *ethclient.Client, parentChain *parent.ParentChain) (*gethexec.ExecutionNode, chan error) {
	t.Helper()
	fatalCh := make(chan error, 10)
	parentClientOpt := containers.None[*ethclient.Client]()
	if parentClient != nil {
		parentClientOpt = containers.Some(parentClient)
	}
	execNode, err := gethexec.CreateExecutionNode(ctx, stack, executionDB, blockchain, newConfigFetcher(execCfg),
		gethexec.WithL1Client(parentClientOpt), gethexec.WithParentChain(parentChain), gethexec.WithFatalErrChan(fatalCh))
	if err != nil {
		t.Fatalf("%s CreateExecutionNode: %v", name, err)
	}
	return execNode, fatalCh
}

// newConsensusNode creates the consensus node with the given parent-chain
// wiring and signing identities. Fails the test on error.
func newConsensusNode(t *testing.T, ctx context.Context, name string, stack *node.Node, execNode *gethexec.ExecutionNode, consensusDB ethdb.Database, nodeConfig *arbnode.Config, chainConfig *params.ChainConfig, parentClient *ethclient.Client, addresses *chaininfo.RollupAddresses, txOptsValidator *bind.TransactOpts, txOptsBatchPoster *bind.TransactOpts, dataSigner signature.DataSignerFunc, fatalCh chan error, blobReader containers.Option[daprovider.BlobReader], wasmRoot common.Hash, parentChain *parent.ParentChain) *arbnode.Node {
	t.Helper()
	consensusNode, err := arbnode.CreateConsensusNode(
		ctx, stack, execNode, consensusDB, newConfigFetcher(nodeConfig), chainConfig, parentClient,
		addresses, txOptsValidator, txOptsBatchPoster, dataSigner, fatalCh, blobReader, wasmRoot, parentChain)
	if err != nil {
		t.Fatalf("%s CreateConsensusNode: %v", name, err)
	}
	return consensusNode
}

// latestWasmModuleRoot returns the newest wasm module root under target/machines.
func latestWasmModuleRoot(t *testing.T) common.Hash {
	t.Helper()
	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("NewMachineLocator: %v", err)
	}
	return locator.LatestWasmModuleRoot()
}

// startNode wires the built pieces into an L2Handle, starts its nodes with the
// fatal watcher and clients, and returns the handle plus the full cleanup;
// closeL1 (nil = none) runs last.
func startNode(t *testing.T, ctx context.Context, rb *rollbackGuard, e *Env, name string, info *arbtest.BlockchainTestInfo, stack *node.Node, execNode *gethexec.ExecutionNode, consensusNode *arbnode.Node, fatalCh chan error, closeL1 func()) (*L2Handle, func()) {
	t.Helper()
	if closeL1 == nil {
		closeL1 = func() {}
	}
	h := &L2Handle{
		ChainHandle: ChainHandle{Info: info, e: e, name: name},
		Stack:       stack,
		ExecNode:    execNode,
		Consensus:   consensusNode,
	}
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
	rb.stage(func() { cleanup(); stopWatcher(); closeL1() })

	var closeClients func()
	h.Client, h.HTTPClient, h.WSClient, closeClients = nodeClients(t, h.Stack, h.e.Spec.ExposeRPC)
	// Stop nodes (joins fatalCh senders) → drain watcher → close clients → close L1.
	fullCleanup := func() {
		cleanup()
		stopWatcher()
		closeClients()
		closeL1()
	}
	rb.stage(fullCleanup)
	return h, fullCleanup
}

// nodeClients attaches the in-process stack client plus dials the HTTP/WS
// endpoints when rpc is set, returning a combined closer.
func nodeClients(t *testing.T, stack *node.Node, rpc bool) (client, httpClient, wsClient *ethclient.Client, closeClients func()) {
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

// becomeChainOwner makes the Owner account a chain owner — required by most
// tests. No-op when the spec opts out via WithoutChainOwner.
func becomeChainOwner(t *testing.T, ctx context.Context, spec Spec, client *ethclient.Client, l2Info *arbtest.BlockchainTestInfo) {
	t.Helper()
	if spec.SkipChainOwner {
		return
	}
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

// rollbackGuard runs the staged cleanup unless commit() was called: builders
// stage partial teardown so a mid-build failure (Fatalf/Goexit) leaks nothing.
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

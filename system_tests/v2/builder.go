// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

// builder.go contains the L2 node lifecycle helpers used by the runner.
// The core logic is ported from system_tests/common_test.go (NodeBuilder.BuildL2
// and createNonL1BlockChainWithStackConfig) into importable, non-test functions.
// Nothing here is specific to any one test — all test-specific setup belongs in
// testConfigX functions.

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution_consensus"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/statetransfer"
	arbtest "github.com/offchainlabs/nitro/system_tests"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
	"github.com/offchainlabs/nitro/validator/server_common"
)

// L2Handle provides access to the running L2 node's client and cleanup function.
// testRunX functions receive this via TestEnv.L2.
type L2Handle struct {
	Client  *ethclient.Client
	cleanup func()
}

// WaitForTx polls until tx appears in a block and returns its receipt.
// Fails the test if not mined within timeout or if the receipt shows a revert.
func (h *L2Handle) WaitForTx(t testing.TB, ctx context.Context, tx *types.Transaction) *types.Receipt {
	t.Helper()
	receipt, err := waitForTxWithTimeout(ctx, h.Client, tx.Hash(), 30*time.Second)
	if err != nil {
		t.Fatalf("WaitForTx %s: %v", tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("transaction %s reverted", tx.Hash())
	}
	return receipt
}

// BalanceAt returns the ETH balance of addr at the latest block.
func (h *L2Handle) BalanceAt(ctx context.Context, addr common.Address) *big.Int {
	bal, err := h.Client.BalanceAt(ctx, addr, nil)
	if err != nil {
		panic("BalanceAt: " + err.Error())
	}
	return bal
}

var defaultForwarderConfig = gethexec.ForwarderConfig{
	ConnectionTimeout:     2 * time.Second,
	IdleConnectionTimeout: 2 * time.Second,
	MaxIdleConnections:    1,
	RedisUrl:              "",
	UpdateInterval:        10 * time.Millisecond,
	RetryInterval:         3 * time.Millisecond,
}

var defaultSequencerConfig = gethexec.SequencerConfig{
	Enable:                       true,
	MaxBlockSpeed:                10 * time.Millisecond,
	ReadFromTxQueueTimeout:       time.Second,
	MaxRevertGasReject:           params.TxGas + 10000,
	MaxAcceptableTimestampDelta:  time.Hour,
	SenderWhitelist:              []string{},
	Forwarder:                    defaultForwarderConfig,
	QueueSize:                    128,
	QueueTimeout:                 5 * time.Second,
	NonceCacheSize:               4,
	MaxTxDataSize:                95000,
	NonceFailureCacheSize:        1024,
	NonceFailureCacheExpiry:      time.Second,
	ExpectedSurplusSoftThreshold: "default",
	ExpectedSurplusHardThreshold: "default",
	ExpectedSurplusGasPriceMode:  "CalldataPrice",
	EnableProfiling:              false,
	TransactionFiltering:         gethexec.DefaultSequencerConfig.TransactionFiltering,
}

func defaultExecConfig(t *testing.T, stateScheme string) *gethexec.Config {
	t.Helper()
	cfg := gethexec.ConfigDefault
	cfg.Caching.StateScheme = stateScheme
	cfg.Sequencer = defaultSequencerConfig
	cfg.ParentChainReader = headerreader.TestConfig
	cfg.ForwardingTarget = "null"
	cfg.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessNone
	cfg.ExposeMultiGas = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("invalid exec config: %v", err)
	}
	return &cfg
}

type configFetcher[T any] interface {
	Get() *T
	Start(context.Context)
	StopAndWait()
	Started() bool
}

type commonConfigFetcher[T any] struct {
	config atomic.Pointer[T]
}

func newConfigFetcher[T any](cfg *T) configFetcher[T] {
	cloned := cloneConfig(cfg)
	f := &commonConfigFetcher[T]{}
	f.config.Store(cloned)
	return f
}

func (f *commonConfigFetcher[T]) Get() *T               { return f.config.Load() }
func (f *commonConfigFetcher[T]) Start(context.Context) {}
func (f *commonConfigFetcher[T]) StopAndWait()          {}
func (f *commonConfigFetcher[T]) Started() bool         { return true }

func cloneConfig[T any](cfg *T) *T {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(cfg); err != nil {
		panic("gob encode: " + err.Error())
	}
	var out T
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		panic("gob decode: " + err.Error())
	}
	// Re-run Validate so unexported fields (e.g. StylusTargetConfig.wasmTargets)
	// are repopulated after the gob round-trip strips them.
	if v, ok := any(&out).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			panic("invalid cloned config: " + err.Error())
		}
	}
	return &out
}

func buildL2Node(t *testing.T, ctx context.Context, spec *BuilderSpec) (*TestEnv, func()) {
	t.Helper()

	stateScheme := spec.StateScheme
	if stateScheme == "" {
		stateScheme = env.GetTestStateScheme()
	}

	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	nodeConfig := arbnode.ConfigDefaultL2Test()
	execCfg := defaultExecConfig(t, stateScheme)
	stackCfg := testhelpers.CreateStackConfigForTest(t.TempDir())

	l2Info, stack, executionDB, consensusDB, blockchain := createBlockChain(
		t, chainConfig, stackCfg, execCfg)

	execFetcher := newConfigFetcher(execCfg)
	execNode, err := gethexec.CreateExecutionNode(ctx, stack, executionDB, blockchain, nil, execFetcher, big.NewInt(1337), 0)
	if err != nil {
		t.Fatalf("CreateExecutionNode: %v", err)
	}

	fatalCh := make(chan error, 10)
	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("NewMachineLocator: %v", err)
	}
	nodeFetcher := newConfigFetcher(nodeConfig)
	consensusNode, err := arbnode.CreateConsensusNode(
		ctx, stack, execNode, consensusDB, nodeFetcher, blockchain.Config(),
		nil, nil, nil, nil, nil, fatalCh, big.NewInt(1337), nil, locator.LatestWasmModuleRoot())
	if err != nil {
		t.Fatalf("CreateConsensusNode: %v", err)
	}

	if err := consensusNode.TxStreamer.AddFakeInitMessage(); err != nil {
		t.Fatalf("AddFakeInitMessage: %v", err)
	}

	cleanup, err := execution_consensus.InitAndStartExecutionAndConsensusNodes(ctx, stack, execNode, consensusNode)
	if err != nil {
		t.Fatalf("InitAndStart: %v", err)
	}

	client := ethclient.NewClient(stack.Attach())

	// Make the genesis Owner a chain owner (required by many tests).
	debugAuth := l2Info.GetDefaultTransactOpts("Owner", ctx)
	arbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), client)
	if err != nil {
		t.Fatalf("NewArbDebug: %v", err)
	}
	tx, err := arbDebug.BecomeChainOwner(&debugAuth)
	if err != nil {
		t.Fatalf("BecomeChainOwner: %v", err)
	}
	if _, err := waitForTxWithTimeout(ctx, client, tx.Hash(), 10*time.Second); err != nil {
		t.Fatalf("BecomeChainOwner tx: %v", err)
	}

	go watchFatalChan(t, ctx, fatalCh)

	handle := &L2Handle{
		Client: client,
		cleanup: func() {
			cleanup()
		},
	}

	env := &TestEnv{
		T:      t,
		Ctx:    ctx,
		L2:     handle,
		L2Info: l2Info,
	}

	return env, func() {
		handle.cleanup()
	}
}

func createBlockChain(
	t *testing.T,
	chainConfig *params.ChainConfig,
	stackCfg *node.Config,
	execCfg *gethexec.Config,
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

	serializedChainConfig, err := json.Marshal(chainConfig)
	if err != nil {
		t.Fatalf("marshal chainConfig: %v", err)
	}
	initMsg := &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
		ChainConfig:           chainConfig,
		SerializedChainConfig: serializedChainConfig,
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

func watchFatalChan(t *testing.T, ctx context.Context, ch <-chan error) {
	select {
	case <-ctx.Done():
		return
	case err := <-ch:
		if ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			return
		}
		t.Errorf("fatal error from consensus node: %v", err)
	}
}

// waitForTxWithTimeout polls until the transaction hash is mined, up to timeout.
// We confirm by checking that the latest block number >= the receipt's block number,
// using HeaderByNumber(nil) rather than PendingCallContract — the latter is
// unreliable on Arbitrum L2 test nodes which have no miner pending state.
func waitForTxWithTimeout(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil && receipt != nil {
			header, herr := client.HeaderByNumber(ctx, nil) // latest
			if herr == nil && header.Number.Cmp(receipt.BlockNumber) >= 0 {
				return receipt, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

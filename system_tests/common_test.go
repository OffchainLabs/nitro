// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/eth/tracers"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	arbosutil "github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	butil "github.com/offchainlabs/nitro/bold/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/deploy"
	"github.com/offchainlabs/nitro/execution/gethexec"
	_ "github.com/offchainlabs/nitro/execution/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/localgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
	"github.com/offchainlabs/nitro/util/testhelpers/github"
	"github.com/offchainlabs/nitro/validator/inputs"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
	rediscons "github.com/offchainlabs/nitro/validator/valnode/redis"
)

type info = *BlockchainTestInfo

type SecondNodeParams struct {
	nodeConfig             *arbnode.Config
	execConfig             *gethexec.Config
	stackConfig            *node.Config
	dasConfig              *das.DataAvailabilityConfig
	initData               *statetransfer.ArbosInitializationInfo
	addresses              *chaininfo.RollupAddresses
	useExecutionClientOnly bool
}

type TestClient struct {
	ctx           context.Context
	Client        *ethclient.Client
	L1Backend     *eth.Ethereum
	Stack         *node.Node
	ConsensusNode *arbnode.Node
	ExecNode      *gethexec.ExecutionNode
	ClientWrapper *ClientWrapper

	// having cleanup() field makes cleanup customizable from default cleanup methods after calling build
	cleanup func()
}

var RollupOwner = "RollupOwner"
var Sequencer = "Sequencer"
var Validator = "Validator"
var User = "User"

var DefaultChainAccounts = []string{RollupOwner, Sequencer, Validator, User}

func NewTestClient(ctx context.Context) *TestClient {
	return &TestClient{ctx: ctx}
}

func (tc *TestClient) SendSignedTx(t *testing.T, l2Client *ethclient.Client, transaction *types.Transaction, lInfo info) *types.Receipt {
	t.Helper()
	return SendSignedTxViaL1(t, tc.ctx, lInfo, tc.Client, l2Client, transaction)
}

func (tc *TestClient) SendUnsignedTx(t *testing.T, l2Client *ethclient.Client, transaction *types.Transaction, lInfo info) *types.Receipt {
	t.Helper()
	return SendUnsignedTxViaL1(t, tc.ctx, lInfo, tc.Client, l2Client, transaction)
}

func (tc *TestClient) TransferBalance(t *testing.T, from string, to string, amount *big.Int, lInfo info) (*types.Transaction, *types.Receipt) {
	t.Helper()
	return TransferBalanceTo(t, from, lInfo.GetAddress(to), amount, lInfo, tc.Client, tc.ctx)
}

func (tc *TestClient) TransferBalanceTo(t *testing.T, from string, to common.Address, amount *big.Int, lInfo info) (*types.Transaction, *types.Receipt) {
	t.Helper()
	return TransferBalanceTo(t, from, to, amount, lInfo, tc.Client, tc.ctx)
}

func (tc *TestClient) GetBalance(t *testing.T, account common.Address) *big.Int {
	t.Helper()
	return GetBalance(t, tc.ctx, tc.Client, account)
}

func (tc *TestClient) GetBaseFee(t *testing.T) *big.Int {
	t.Helper()
	return GetBaseFee(t, tc.Client, tc.ctx)
}

func (tc *TestClient) GetBaseFeeAt(t *testing.T, blockNum *big.Int) *big.Int {
	t.Helper()
	return GetBaseFeeAt(t, tc.Client, tc.ctx, blockNum)
}

func (tc *TestClient) SendWaitTestTransactions(t *testing.T, txs []*types.Transaction) []*types.Receipt {
	t.Helper()
	return SendWaitTestTransactions(t, tc.ctx, tc.Client, txs)
}

func (tc *TestClient) DeployBigMap(t *testing.T, auth bind.TransactOpts) (common.Address, *localgen.BigMap) {
	t.Helper()
	return deployBigMap(t, tc.ctx, auth, tc.Client)
}

func (tc *TestClient) DeploySimple(t *testing.T, auth bind.TransactOpts) (common.Address, *localgen.Simple) {
	t.Helper()
	return deploySimple(t, tc.ctx, auth, tc.Client)
}

func (tc *TestClient) EnsureTxSucceeded(transaction *types.Transaction) (*types.Receipt, error) {
	return tc.EnsureTxSucceededWithTimeout(transaction, time.Second*5)
}

func (tc *TestClient) EnsureTxSucceededWithTimeout(transaction *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	return EnsureTxSucceededWithTimeout(tc.ctx, tc.Client, transaction, timeout)
}

var DefaultTestForwarderConfig = gethexec.ForwarderConfig{
	ConnectionTimeout:     2 * time.Second,
	IdleConnectionTimeout: 2 * time.Second,
	MaxIdleConnections:    1,
	RedisUrl:              "",
	UpdateInterval:        time.Millisecond * 10,
	RetryInterval:         time.Millisecond * 3,
}

var TestSequencerConfig = gethexec.SequencerConfig{
	Enable:                       true,
	MaxBlockSpeed:                time.Millisecond * 10,
	ReadFromTxQueueTimeout:       time.Second, // Dont want this to affect tests
	MaxRevertGasReject:           params.TxGas + 10000,
	MaxAcceptableTimestampDelta:  time.Hour,
	SenderWhitelist:              []string{},
	Forwarder:                    DefaultTestForwarderConfig,
	QueueSize:                    128,
	QueueTimeout:                 time.Second * 5,
	NonceCacheSize:               4,
	MaxTxDataSize:                95000,
	NonceFailureCacheSize:        1024,
	NonceFailureCacheExpiry:      time.Second,
	ExpectedSurplusGasPriceMode:  "CalldataPrice",
	ExpectedSurplusSoftThreshold: "default",
	ExpectedSurplusHardThreshold: "default",
	EnableProfiling:              false,
}

func ExecConfigDefaultNonSequencerTest(t *testing.T, stateScheme string) *gethexec.Config {
	config := gethexec.ConfigDefault
	config.Caching.StateScheme = stateScheme
	config.RPC.StateScheme = stateScheme
	config.ParentChainReader = headerreader.TestConfig
	config.Sequencer.Enable = false
	config.Forwarder = DefaultTestForwarderConfig
	config.ForwardingTarget = "null"
	config.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessNone

	Require(t, config.Validate())

	return &config
}

func ExecConfigDefaultTest(t *testing.T, stateScheme string) *gethexec.Config {
	config := gethexec.ConfigDefault
	config.Caching.StateScheme = stateScheme
	config.RPC.StateScheme = stateScheme
	config.Sequencer = TestSequencerConfig
	config.ParentChainReader = headerreader.TestConfig
	config.ForwardingTarget = "null"
	config.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessNone
	config.ExposeMultiGas = true

	Require(t, config.Validate())

	return &config
}

type NodeBuilder struct {
	// NodeBuilder configuration
	ctx           context.Context
	ctxCancel     context.CancelFunc
	chainConfig   *params.ChainConfig
	arbOSInit     *params.ArbOSInit
	nodeConfig    *arbnode.Config
	execConfig    *gethexec.Config
	l1StackConfig *node.Config
	l2StackConfig *node.Config
	valnodeConfig *valnode.Config
	l3Config      *NitroConfig
	deployBold    bool
	parallelise   bool
	L1Info        info
	L2Info        info
	L3Info        info

	// L1, L2, L3 Node parameters
	dataDir                     string
	isSequencer                 bool
	takeOwnership               bool
	withL1                      bool
	defaultDbScheme             string
	addresses                   *chaininfo.RollupAddresses
	l3Addresses                 *chaininfo.RollupAddresses
	initMessage                 *arbostypes.ParsedInitMessage
	l3InitMessage               *arbostypes.ParsedInitMessage
	withProdConfirmPeriodBlocks bool
	delayBufferThreshold        uint64
	withL1ClientWrapper         bool

	ignoreExecConfigValidationError bool

	// Created nodes
	L1 *TestClient
	L2 *TestClient
	L3 *TestClient
}

type NitroConfig struct {
	chainConfig   *params.ChainConfig
	arbOSConfig   *params.ArbOSInit
	nodeConfig    *arbnode.Config
	execConfig    *gethexec.Config
	stackConfig   *node.Config
	valnodeConfig *valnode.Config

	withProdConfirmPeriodBlocks bool
	isSequencer                 bool
}

func L3NitroConfigDefaultTest(t *testing.T) *NitroConfig {
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(333333),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArbitrumChainParams: chaininfo.ArbitrumDevTestParams(),
		Clique: &params.CliqueConfig{
			Period: 0,
			Epoch:  0,
		},
	}

	valnodeConfig := valnode.TestValidationConfig
	return &NitroConfig{
		chainConfig:   chainConfig,
		nodeConfig:    arbnode.ConfigDefaultL1Test(),
		execConfig:    ExecConfigDefaultTest(t, rawdb.HashScheme),
		stackConfig:   testhelpers.CreateStackConfigForTest(t.TempDir()),
		valnodeConfig: &valnodeConfig,

		withProdConfirmPeriodBlocks: false,
		isSequencer:                 true,
	}
}

func NewNodeBuilder(ctxIn context.Context) *NodeBuilder {
	ctx, cancel := context.WithCancel(ctxIn)
	return &NodeBuilder{ctx: ctx, ctxCancel: cancel}
}

func (b *NodeBuilder) DefaultConfig(t *testing.T, withL1 bool) *NodeBuilder {
	// most used values across current tests are set here as default
	b.withL1 = withL1
	b.parallelise = true
	b.deployBold = true
	if withL1 {
		b.isSequencer = true
		b.nodeConfig = arbnode.ConfigDefaultL1Test()
	} else {
		b.takeOwnership = true
		b.nodeConfig = arbnode.ConfigDefaultL2Test()
	}
	b.chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	b.L1Info = NewL1TestInfo(t)
	b.L2Info = NewArbTestInfo(t, b.chainConfig.ChainID)
	b.dataDir = t.TempDir()
	b.l1StackConfig = testhelpers.CreateStackConfigForTest(b.dataDir)
	b.l2StackConfig = testhelpers.CreateStackConfigForTest(b.dataDir)
	cp := valnode.TestValidationConfig
	b.valnodeConfig = &cp
	b.defaultDbScheme = rawdb.HashScheme
	if *testflag.StateSchemeFlag == rawdb.PathScheme || *testflag.StateSchemeFlag == rawdb.HashScheme {
		b.defaultDbScheme = *testflag.StateSchemeFlag
	}
	b.execConfig = ExecConfigDefaultTest(t, b.defaultDbScheme)
	b.l3Config = L3NitroConfigDefaultTest(t)
	return b
}

func (b *NodeBuilder) DontParalellise() *NodeBuilder {
	b.parallelise = false
	return b
}

func (b *NodeBuilder) WithArbOSVersion(arbosVersion uint64) *NodeBuilder {
	newChainConfig := *b.chainConfig
	newChainConfig.ArbitrumChainParams.InitialArbOSVersion = arbosVersion
	b.chainConfig = &newChainConfig
	return b
}

func (b *NodeBuilder) WithArbOSInit(arbOSInit *params.ArbOSInit) *NodeBuilder {
	b.arbOSInit = arbOSInit
	return b
}

func (b *NodeBuilder) WithProdConfirmPeriodBlocks() *NodeBuilder {
	b.withProdConfirmPeriodBlocks = true
	return b
}

func (b *NodeBuilder) WithPreBoldDeployment() *NodeBuilder {
	b.deployBold = false
	return b
}

func (b *NodeBuilder) WithWasmRootDir(wasmRootDir string) *NodeBuilder {
	b.valnodeConfig.Wasm.RootPath = wasmRootDir
	return b
}

func (b *NodeBuilder) WithExtraArchs(targets []string) *NodeBuilder {
	b.execConfig.StylusTarget.ExtraArchs = targets
	return b
}

// WithDelayBuffer sets the delay-buffer threshold, which is the number of blocks the batch-poster
// is allowed to delay a batch with a delayed message.
// Setting the threshold to zero disabled the delay buffer (default behaviour).
func (b *NodeBuilder) WithDelayBuffer(threshold uint64) *NodeBuilder {
	b.delayBufferThreshold = threshold
	return b
}

func (b *NodeBuilder) RequireScheme(t *testing.T, scheme string) *NodeBuilder {
	if testflag.StateSchemeFlag != nil && *testflag.StateSchemeFlag != "" && *testflag.StateSchemeFlag != scheme {
		t.Skip("skipping because db scheme is set and not ", scheme)
	}
	if b.defaultDbScheme != scheme && b.execConfig != nil {
		b.execConfig.Caching.StateScheme = scheme
		b.execConfig.RPC.StateScheme = scheme
		b.validateExecConfig(t)
	}
	b.defaultDbScheme = scheme
	return b
}

func (b *NodeBuilder) ExecConfigDefaultTest(t *testing.T, sequencer bool) *gethexec.Config {
	if sequencer {
		ExecConfigDefaultTest(t, b.defaultDbScheme)
	}
	return ExecConfigDefaultNonSequencerTest(t, b.defaultDbScheme)
}

// WithL1ClientWrapper creates a ClientWrapper for the L1 RPC client before passing it to the L2 node.
func (b *NodeBuilder) WithL1ClientWrapper(t *testing.T) *NodeBuilder {
	if !b.withL1 {
		Fatal(t, "WithL1ClientWrapper only works when L1 is enabled")
	}
	b.withL1ClientWrapper = true
	return b
}

func (b *NodeBuilder) TakeOwnership() *NodeBuilder {
	b.takeOwnership = true
	return b
}

func (b *NodeBuilder) DontSendL2SetupTxes() *NodeBuilder {
	b.takeOwnership = false // taking ownership requires sequencing arbdebug call
	return b
}

func (b *NodeBuilder) IgnoreExecConfigValidationError() *NodeBuilder {
	b.ignoreExecConfigValidationError = true
	return b
}

func (b *NodeBuilder) validateExecConfig(t *testing.T) {
	validateExecConfig(t, b.execConfig, b.ignoreExecConfigValidationError)
}

func validateExecConfig(t *testing.T, execConfig *gethexec.Config, ignoreExecConfigValidationError bool) {
	t.Helper()
	err := execConfig.Validate()
	if err != nil && ignoreExecConfigValidationError {
		log.Warn("ignoring execution config validation error", "err", err)
		return
	}
	Require(t, err)
}

func (b *NodeBuilder) Build(t *testing.T) func() {
	if b.parallelise {
		b.parallelise = false
		t.Parallel()
	}
	b.CheckConfig(t)
	if b.withL1 {
		b.BuildL1(t)
		return b.BuildL2OnL1(t)
	}
	return b.BuildL2(t)
}

type testCollection struct {
	room    atomic.Int64
	cond    *sync.Cond
	running map[string]int64
	waiting map[string]int64
}

var globalCollection *testCollection

func initTestCollection() {
	if globalCollection != nil {
		panic("trying to init testCollection twice")
	}
	globalCollection = &testCollection{}
	globalCollection.cond = sync.NewCond(&sync.Mutex{})
	room := int64(util.GoMaxProcs())
	if room < 2 {
		room = 2
	}
	globalCollection.running = make(map[string]int64)
	globalCollection.waiting = make(map[string]int64)
	globalCollection.room.Store(room)
}

func runningWithContext(ctx context.Context, weight int64, name string) {
	current := globalCollection.running[name]
	globalCollection.running[name] = current + weight
	globalCollection.cond.L.Unlock()
	go func() {
		<-ctx.Done()
		globalCollection.cond.L.Lock()
		current := globalCollection.running[name]
		if current-weight <= 0 {
			delete(globalCollection.running, name)
		} else {
			globalCollection.running[name] = current - weight
		}
		if globalCollection.room.Add(weight) > 0 {
			globalCollection.cond.Broadcast()
		}
		globalCollection.cond.L.Unlock()
	}()
}

func WaitAndRun(ctx context.Context, weight int64, name string) error {
	globalCollection.cond.L.Lock()
	current := globalCollection.waiting[name]
	globalCollection.waiting[name] = current + weight
	for globalCollection.room.Add(0-weight) < 0 {
		if globalCollection.room.Add(weight) > 0 {
			globalCollection.cond.Broadcast()
		}
		if ctx.Err() != nil {
			return fmt.Errorf("Context cancelled while waiting to launch test: %s", name)
		}
		globalCollection.cond.Wait()
	}
	current = globalCollection.waiting[name]
	if current-weight <= 0 {
		delete(globalCollection.waiting, name)
	} else {
		globalCollection.waiting[name] = current - weight
	}
	runningWithContext(ctx, weight, name)
	return nil
}

func DontWaitAndRun(ctx context.Context, weight int64, name string) {
	globalCollection.room.Add(0 - weight)
	globalCollection.cond.L.Lock()
	runningWithContext(ctx, weight, name)
}

func CurrentlyRunning() (map[string]int64, map[string]int64) {
	running := make(map[string]int64)
	waiting := make(map[string]int64)
	globalCollection.cond.L.Lock()
	for k, v := range globalCollection.running {
		if v > 0 {
			running[k] = v
		}
	}
	for k, v := range globalCollection.waiting {
		if v > 0 {
			waiting[k] = v
		}
	}
	globalCollection.cond.L.Unlock()
	return running, waiting
}

func (b *NodeBuilder) CheckConfig(t *testing.T) {
	if b.chainConfig == nil {
		b.chainConfig = chaininfo.ArbitrumDevTestChainConfig()
	}
	if b.nodeConfig == nil {
		b.nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	if b.nodeConfig.ValidatorRequired() {
		// validation currently requires hash
		b.RequireScheme(t, rawdb.HashScheme)
	}
	if b.defaultDbScheme == "" {
		b.defaultDbScheme = env.GetTestStateScheme()
	}
	if b.execConfig == nil {
		b.execConfig = b.ExecConfigDefaultTest(t, true)
	}
	if b.execConfig.Caching.Archive {
		// archive currently requires hash
		b.RequireScheme(t, rawdb.HashScheme)
	}
	if b.L1Info == nil {
		b.L1Info = NewL1TestInfo(t)
	}
	if b.L2Info == nil {
		b.L2Info = NewArbTestInfo(t, b.chainConfig.ChainID)
	}
	if b.execConfig.RPC.MaxRecreateStateDepth == arbitrum.UninitializedMaxRecreateStateDepth {
		if b.execConfig.Caching.Archive {
			b.execConfig.RPC.MaxRecreateStateDepth = arbitrum.DefaultArchiveNodeMaxRecreateStateDepth
		} else {
			b.execConfig.RPC.MaxRecreateStateDepth = arbitrum.DefaultNonArchiveNodeMaxRecreateStateDepth
		}
	}
}

func (b *NodeBuilder) BuildL1(t *testing.T) {
	if b.parallelise {
		b.parallelise = false
		t.Parallel()
	}
	err := WaitAndRun(b.ctx, 2, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	b.L1 = NewTestClient(b.ctx)
	b.L1Info, b.L1.Client, b.L1.L1Backend, b.L1.Stack, b.L1.ClientWrapper = createTestL1BlockChain(t, b.L1Info, b.withL1ClientWrapper)
	locator, err := server_common.NewMachineLocator(b.valnodeConfig.Wasm.RootPath)
	Require(t, err)
	b.addresses, b.initMessage = deployOnParentChain(
		t,
		b.ctx,
		b.L1Info,
		b.L1.Client,
		&headerreader.TestConfig,
		b.chainConfig,
		locator.LatestWasmModuleRoot(),
		b.withProdConfirmPeriodBlocks,
		true,
		b.deployBold,
		b.delayBufferThreshold,
	)
	b.L1.cleanup = func() { requireClose(t, b.L1.Stack) }
}

func buildOnParentChain(
	t *testing.T,
	ctx context.Context,

	dataDir string,

	parentChainInfo info,
	parentChainTestClient *TestClient,
	parentChainId *big.Int,

	chainConfig *params.ChainConfig,
	arbOSInit *params.ArbOSInit,
	stackConfig *node.Config,
	execConfig *gethexec.Config,
	nodeConfig *arbnode.Config,
	valnodeConfig *valnode.Config,
	isSequencer bool,
	chainInfo info,

	initMessage *arbostypes.ParsedInitMessage,
	addresses *chaininfo.RollupAddresses,

	ignoreExecConfigValidationError bool,
) *TestClient {
	if parentChainTestClient == nil {
		t.Fatal("must build parent chain before building chain")
	}

	chainTestClient := NewTestClient(ctx)

	var chainDb ethdb.Database
	var arbDb ethdb.Database
	var blockchain *core.BlockChain
	_, chainTestClient.Stack, chainDb, arbDb, blockchain = createNonL1BlockChainWithStackConfig(
		t, chainInfo, dataDir, chainConfig, arbOSInit, initMessage, stackConfig, execConfig,
		ignoreExecConfigValidationError)

	var sequencerTxOptsPtr *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	if isSequencer {
		sequencerTxOpts := parentChainInfo.GetDefaultTransactOpts("Sequencer", ctx)
		sequencerTxOptsPtr = &sequencerTxOpts
		dataSigner = signature.DataSignerFromPrivateKey(parentChainInfo.GetInfoWithPrivKey("Sequencer").PrivateKey)
	} else {
		nodeConfig.BatchPoster.Enable = false
		nodeConfig.Sequencer = false
		nodeConfig.DelayedSequencer.Enable = false
		execConfig.Sequencer.Enable = false
	}

	var validatorTxOptsPtr *bind.TransactOpts
	if nodeConfig.Staker.Enable {
		validatorTxOpts := parentChainInfo.GetDefaultTransactOpts("Validator", ctx)
		validatorTxOptsPtr = &validatorTxOpts
	}

	AddValNodeIfNeeded(t, ctx, nodeConfig, true, "", valnodeConfig.Wasm.RootPath)

	validateExecConfig(t, execConfig, ignoreExecConfigValidationError)
	execConfigToBeUsedInConfigFetcher := execConfig
	execConfigFetcher := func() *gethexec.Config { return execConfigToBeUsedInConfigFetcher }
	execNode, err := gethexec.CreateExecutionNode(ctx, chainTestClient.Stack, chainDb, blockchain, parentChainTestClient.Client, execConfigFetcher, parentChainId, 0)
	Require(t, err)

	fatalErrChan := make(chan error, 10)
	locator, err := server_common.NewMachineLocator(valnodeConfig.Wasm.RootPath)
	Require(t, err)
	chainTestClient.ConsensusNode, err = arbnode.CreateNodeFullExecutionClient(
		ctx, chainTestClient.Stack, execNode, execNode, execNode, execNode, arbDb, NewFetcherFromConfig(nodeConfig), blockchain.Config(), parentChainTestClient.Client,
		addresses, validatorTxOptsPtr, sequencerTxOptsPtr, dataSigner, fatalErrChan, parentChainId, nil, locator.LatestWasmModuleRoot())
	Require(t, err)

	err = chainTestClient.ConsensusNode.Start(ctx)
	Require(t, err)

	chainTestClient.Client = ClientForStack(t, chainTestClient.Stack)

	StartWatchChanErr(t, ctx, fatalErrChan, chainTestClient.ConsensusNode)

	chainTestClient.ExecNode = getExecNode(t, chainTestClient.ConsensusNode)
	chainTestClient.cleanup = func() { chainTestClient.ConsensusNode.StopAndWait() }

	return chainTestClient
}

func (b *NodeBuilder) BuildL3OnL2(t *testing.T) func() {
	DontWaitAndRun(b.ctx, 1, t.Name())
	b.L3Info = NewArbTestInfo(t, b.l3Config.chainConfig.ChainID)

	locator, err := server_common.NewMachineLocator(b.l3Config.valnodeConfig.Wasm.RootPath)
	Require(t, err)

	parentChainReaderConfig := headerreader.TestConfig
	parentChainReaderConfig.Dangerous.WaitForTxApprovalSafePoll = 0
	b.l3Addresses, b.l3InitMessage = deployOnParentChain(
		t,
		b.ctx,
		b.L2Info,
		b.L2.Client,
		&parentChainReaderConfig,
		b.l3Config.chainConfig,
		locator.LatestWasmModuleRoot(),
		b.l3Config.withProdConfirmPeriodBlocks,
		false,
		b.deployBold,
		0,
	)

	b.L3 = buildOnParentChain(
		t,
		b.ctx,

		b.dataDir,

		b.L2Info,
		b.L2,
		b.chainConfig.ChainID,

		b.l3Config.chainConfig,
		b.l3Config.arbOSConfig,
		b.l3Config.stackConfig,
		b.l3Config.execConfig,
		b.l3Config.nodeConfig,
		b.l3Config.valnodeConfig,
		b.l3Config.isSequencer,
		b.L3Info,

		b.l3InitMessage,
		b.l3Addresses,

		b.ignoreExecConfigValidationError,
	)

	return func() {
		b.L3.cleanup()
	}
}

func (b *NodeBuilder) BuildL2OnL1(t *testing.T) func() {
	b.L2 = buildOnParentChain(
		t,
		b.ctx,

		b.dataDir,

		b.L1Info,
		b.L1,
		big.NewInt(1337),

		b.chainConfig,
		b.arbOSInit,
		b.l2StackConfig,
		b.execConfig,
		b.nodeConfig,
		b.valnodeConfig,
		b.isSequencer,
		b.L2Info,

		b.initMessage,
		b.addresses,

		b.ignoreExecConfigValidationError,
	)

	if b.takeOwnership {
		debugAuth := b.L2Info.GetDefaultTransactOpts("Owner", b.ctx)

		// make auth a chain owner
		arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), b.L2.Client)
		Require(t, err, "failed to deploy ArbDebug")

		tx, err := arbdebug.BecomeChainOwner(&debugAuth)
		Require(t, err, "failed to deploy ArbDebug")

		_, err = EnsureTxSucceeded(b.ctx, b.L2.Client, tx)
		Require(t, err)
	}

	return func() {
		b.L2.cleanup()
		if b.L1 != nil && b.L1.cleanup != nil {
			b.L1.cleanup()
		}
		b.ctxCancel()
	}
}

// L2 -Only. Enough for tests that needs no interface to L1
// Requires precompiles.AllowDebugPrecompiles = true
func (b *NodeBuilder) BuildL2(t *testing.T) func() {
	if b.parallelise {
		b.parallelise = false
		t.Parallel()
	}
	err := WaitAndRun(b.ctx, 1, t.Name())
	if err != nil {
		Fatal(t, err)
	}
	b.L2 = NewTestClient(b.ctx)

	AddValNodeIfNeeded(t, b.ctx, b.nodeConfig, true, "", b.valnodeConfig.Wasm.RootPath)

	var chainDb ethdb.Database
	var arbDb ethdb.Database
	var blockchain *core.BlockChain
	b.L2Info, b.L2.Stack, chainDb, arbDb, blockchain = createNonL1BlockChainWithStackConfig(
		t, b.L2Info, b.dataDir, b.chainConfig, b.arbOSInit, nil, b.l2StackConfig, b.execConfig,
		b.ignoreExecConfigValidationError)

	b.validateExecConfig(t)
	execConfig := b.execConfig
	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(b.ctx, b.L2.Stack, chainDb, blockchain, nil, execConfigFetcher, big.NewInt(1337), 0)
	Require(t, err)

	fatalErrChan := make(chan error, 10)
	locator, err := server_common.NewMachineLocator(b.valnodeConfig.Wasm.RootPath)
	Require(t, err)
	b.L2.ConsensusNode, err = arbnode.CreateNodeFullExecutionClient(
		b.ctx, b.L2.Stack, execNode, execNode, execNode, execNode, arbDb, NewFetcherFromConfig(b.nodeConfig), blockchain.Config(),
		nil, nil, nil, nil, nil, fatalErrChan, big.NewInt(1337), nil, locator.LatestWasmModuleRoot())
	Require(t, err)

	// Give the node an init message
	err = b.L2.ConsensusNode.TxStreamer.AddFakeInitMessage()
	Require(t, err)

	err = b.L2.ConsensusNode.Start(b.ctx)
	Require(t, err)

	b.L2.Client = ClientForStack(t, b.L2.Stack)

	if b.takeOwnership {
		debugAuth := b.L2Info.GetDefaultTransactOpts("Owner", b.ctx)

		// make auth a chain owner
		arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), b.L2.Client)
		Require(t, err, "failed to deploy ArbDebug")

		tx, err := arbdebug.BecomeChainOwner(&debugAuth)
		Require(t, err, "failed to deploy ArbDebug")

		_, err = EnsureTxSucceeded(b.ctx, b.L2.Client, tx)
		Require(t, err)
	}

	StartWatchChanErr(t, b.ctx, fatalErrChan, b.L2.ConsensusNode)

	b.L2.ExecNode = getExecNode(t, b.L2.ConsensusNode)
	b.L2.cleanup = func() { b.L2.ConsensusNode.StopAndWait() }
	return func() {
		b.L2.cleanup()
		b.ctxCancel()
	}
}

// L2 -Only. RestartL2Node shutdowns the existing l2 node and start it again using the same data dir.
func (b *NodeBuilder) RestartL2Node(t *testing.T) {
	if b.L2 == nil {
		t.Fatalf("L2 was not created")
	}
	b.L2.cleanup()

	l2info, stack, chainDb, arbDb, blockchain := createNonL1BlockChainWithStackConfig(t, b.L2Info, b.dataDir, b.chainConfig, b.arbOSInit, b.initMessage, b.l2StackConfig, b.execConfig, b.ignoreExecConfigValidationError)

	execConfigFetcher := func() *gethexec.Config { return b.execConfig }
	execNode, err := gethexec.CreateExecutionNode(b.ctx, stack, chainDb, blockchain, nil, execConfigFetcher, big.NewInt(1337), 0)
	Require(t, err)

	feedErrChan := make(chan error, 10)
	locator, err := server_common.NewMachineLocator(b.valnodeConfig.Wasm.RootPath)
	Require(t, err)
	var sequencerTxOpts *bind.TransactOpts
	var validatorTxOpts *bind.TransactOpts
	var dataSigner signature.DataSignerFunc
	var l1Client *ethclient.Client
	if b.withL1 {
		sequencerTxOptsNP := b.L1Info.GetDefaultTransactOpts("Sequencer", b.ctx)
		sequencerTxOpts = &sequencerTxOptsNP
		validatorTxOptsNP := b.L1Info.GetDefaultTransactOpts("Validator", b.ctx)
		validatorTxOpts = &validatorTxOptsNP
		dataSigner = signature.DataSignerFromPrivateKey(b.L1Info.GetInfoWithPrivKey("Sequencer").PrivateKey)
		l1Client = b.L1.Client
	}
	chainConfig := blockchain.Config()
	if b.execConfig.Dangerous.DebugBlock.OverwriteChainConfig {
		b.execConfig.Dangerous.DebugBlock.Apply(chainConfig)
	}
	currentNode, err := arbnode.CreateNodeFullExecutionClient(b.ctx, stack, execNode, execNode, execNode, execNode, arbDb, NewFetcherFromConfig(b.nodeConfig), chainConfig, l1Client, b.addresses, validatorTxOpts, sequencerTxOpts, dataSigner, feedErrChan, big.NewInt(1337), nil, locator.LatestWasmModuleRoot())
	Require(t, err)

	Require(t, currentNode.Start(b.ctx))
	client := ClientForStack(t, stack)

	StartWatchChanErr(t, b.ctx, feedErrChan, currentNode)

	l2 := NewTestClient(b.ctx)
	l2.ConsensusNode = currentNode
	l2.Client = client
	l2.ExecNode = execNode
	l2.cleanup = func() { b.L2.ConsensusNode.StopAndWait() }
	l2.Stack = stack

	b.L2 = l2
	b.L2Info = l2info
}

func build2ndNode(
	t *testing.T,
	ctx context.Context,

	firstNodeStackConfig *node.Config,
	firsNodeExecConfig *gethexec.Config,
	firstNodeNodeConfig *arbnode.Config,
	firstNodeInfo info,
	firstNodeTestClient *TestClient,
	valnodeConfig *valnode.Config,

	parentChainTestClient *TestClient,
	parentChainInfo info,

	params *SecondNodeParams,

	addresses *chaininfo.RollupAddresses,
	initMessage *arbostypes.ParsedInitMessage,

	ignoreExecConfigValidationError bool,
) (*TestClient, func()) {
	if params.nodeConfig == nil {
		params.nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if params.dasConfig != nil {
		params.nodeConfig.DataAvailability = *params.dasConfig
	}
	if params.stackConfig == nil {
		params.stackConfig = firstNodeStackConfig
		// should use different dataDir from the previously used ones
		params.stackConfig.DataDir = t.TempDir()
	}
	if params.initData == nil {
		params.initData = &firstNodeInfo.ArbInitData
	}
	if params.execConfig == nil {
		params.execConfig = firsNodeExecConfig
	}
	if params.addresses == nil {
		params.addresses = addresses
	}
	if params.execConfig.RPC.MaxRecreateStateDepth == arbitrum.UninitializedMaxRecreateStateDepth {
		if params.execConfig.Caching.Archive {
			params.execConfig.RPC.MaxRecreateStateDepth = arbitrum.DefaultArchiveNodeMaxRecreateStateDepth
		} else {
			params.execConfig.RPC.MaxRecreateStateDepth = arbitrum.DefaultNonArchiveNodeMaxRecreateStateDepth
		}
	}
	if firstNodeNodeConfig.BatchPoster.Enable && params.nodeConfig.BatchPoster.Enable && params.nodeConfig.BatchPoster.RedisUrl == "" {
		t.Fatal("The batch poster must use Redis when enabled for multiple nodes")
	}

	testClient := NewTestClient(ctx)
	testClient.Client, testClient.ConsensusNode =
		Create2ndNodeWithConfig(t, ctx, firstNodeTestClient.ConsensusNode, parentChainTestClient.Stack, parentChainInfo, params.initData, params.nodeConfig, params.execConfig, params.stackConfig, valnodeConfig, params.addresses, initMessage, params.useExecutionClientOnly, ignoreExecConfigValidationError)
	testClient.ExecNode = getExecNode(t, testClient.ConsensusNode)
	testClient.cleanup = func() { testClient.ConsensusNode.StopAndWait() }
	return testClient, func() { testClient.cleanup() }
}

func (b *NodeBuilder) Build2ndNode(t *testing.T, params *SecondNodeParams) (*TestClient, func()) {
	DontWaitAndRun(b.ctx, 1, t.Name())
	if b.L2 == nil {
		t.Fatal("builder did not previously build an L2 Node")
	}
	if b.L1 == nil {
		t.Fatal("builder did not previously build an L1 Node")
	}
	return build2ndNode(
		t,
		b.ctx,

		b.l2StackConfig,
		b.execConfig,
		b.nodeConfig,
		b.L2Info,
		b.L2,
		b.valnodeConfig,

		b.L1,
		b.L1Info,

		params,

		b.addresses,
		b.initMessage,

		b.ignoreExecConfigValidationError,
	)
}

func (b *NodeBuilder) Build2ndNodeOnL3(t *testing.T, params *SecondNodeParams) (*TestClient, func()) {
	DontWaitAndRun(b.ctx, 1, t.Name())
	if b.L3 == nil {
		t.Fatal("builder did not previously built an L3 Node")
	}
	return build2ndNode(
		t,
		b.ctx,

		b.l3Config.stackConfig,
		b.l3Config.execConfig,
		b.l3Config.nodeConfig,
		b.L3Info,
		b.L3,
		b.l3Config.valnodeConfig,

		b.L2,
		b.L2Info,

		params,

		b.l3Addresses,
		b.l3InitMessage,

		b.ignoreExecConfigValidationError,
	)
}

func (b *NodeBuilder) BridgeBalance(t *testing.T, account string, amount *big.Int) (*types.Transaction, *types.Receipt) {
	return BridgeBalance(t, account, amount, b.L1Info, b.L2Info, b.L1.Client, b.L2.Client, b.ctx)
}

func SendWaitTestTransactions(t *testing.T, ctx context.Context, client *ethclient.Client, txs []*types.Transaction) []*types.Receipt {
	t.Helper()
	receipts := make([]*types.Receipt, len(txs))
	for _, tx := range txs {
		Require(t, client.SendTransaction(ctx, tx))
	}
	for i, tx := range txs {
		var err error
		receipts[i], err = EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}
	return receipts
}

func TransferBalance(
	t *testing.T, from, to string, amount *big.Int, l2info info, client *ethclient.Client, ctx context.Context,
) (*types.Transaction, *types.Receipt) {
	t.Helper()
	return TransferBalanceTo(t, from, l2info.GetAddress(to), amount, l2info, client, ctx)
}

func TransferBalanceTo(
	t *testing.T, from string, to common.Address, amount *big.Int, l2info info, client *ethclient.Client, ctx context.Context,
) (*types.Transaction, *types.Receipt) {
	t.Helper()
	tx := l2info.PrepareTxTo(from, &to, l2info.TransferGas, amount, nil)
	err := client.SendTransaction(ctx, tx)
	Require(t, err)
	res, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	return tx, res
}

// if l2client is not nil - will wait until balance appears in l2
func BridgeBalance(
	t *testing.T, account string, amount *big.Int, l1info info, l2info info, l1client *ethclient.Client, l2client *ethclient.Client, ctx context.Context,
) (*types.Transaction, *types.Receipt) {
	t.Helper()

	// setup or validate the same account on l2info
	l1acct := l1info.GetInfoWithPrivKey(account)
	if l2info.Accounts[account] == nil {
		l2info.SetFullAccountInfo(account, &AccountInfo{
			Address:    l1acct.Address,
			PrivateKey: l1acct.PrivateKey,
		})
	} else {
		l2acct := l2info.GetInfoWithPrivKey(account)
		if l2acct.PrivateKey.X.Cmp(l1acct.PrivateKey.X) != 0 ||
			l2acct.PrivateKey.Y.Cmp(l1acct.PrivateKey.Y) != 0 {
			Fatal(t, "l2 account already exists and not compatible to l1")
		}
	}

	// check previous balance
	var l2Balance *big.Int
	var err error
	if l2client != nil {
		l2Balance, err = l2client.BalanceAt(ctx, l2info.GetAddress("Faucet"), nil)
		Require(t, err)
	}

	// send transaction
	data, err := hex.DecodeString("0f4d14e9000000000000000000000000000000000000000000000000000082f79cd90000")
	Require(t, err)
	tx := l1info.PrepareTx(account, "Inbox", l1info.TransferGas*100, amount, data)
	err = l1client.SendTransaction(ctx, tx)
	Require(t, err)
	res, err := EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)

	// wait for balance to appear in l2
	if l2client != nil {
		l2Balance.Add(l2Balance, amount)
		for i := 0; true; i++ {
			balance, err := l2client.BalanceAt(ctx, l2info.GetAddress("Faucet"), nil)
			Require(t, err)
			if balance.Cmp(l2Balance) >= 0 {
				break
			}
			TransferBalance(t, "Faucet", "User", big.NewInt(1), l1info, l1client, ctx)
			if i > 200 {
				Fatal(t, "bridging failed")
			}
			<-time.After(time.Millisecond * 100)
		}
	}

	return tx, res
}

// AdvanceL1 sends dummy transactions to L1 to create blocks.
func AdvanceL1(
	t *testing.T,
	ctx context.Context,
	l1client *ethclient.Client,
	l1info *BlockchainTestInfo,
	numBlocks int,
) {
	for i := 0; i < numBlocks; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
}

func SendSignedTxesInBatchViaL1(
	t *testing.T,
	ctx context.Context,
	l1info *BlockchainTestInfo,
	l1client *ethclient.Client,
	l2client *ethclient.Client,
	delayedTxes types.Transactions,
) types.Receipts {
	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)
	usertxopts := l1info.GetDefaultTransactOpts("User", ctx)

	wraped, err := l2MessageBatchDataFromTxes(delayedTxes)
	Require(t, err)
	l1tx, err := delayedInboxContract.SendL2Message(&usertxopts, wraped)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)

	AdvanceL1(t, ctx, l1client, l1info, 30)
	var receipts types.Receipts
	for _, tx := range delayedTxes {
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		receipts = append(receipts, receipt)
	}
	return receipts
}

func l2MessageBatchDataFromTxes(txes types.Transactions) ([]byte, error) {
	var l2Message []byte
	l2Message = append(l2Message, arbos.L2MessageKind_Batch)
	sizeBuf := make([]byte, 8)
	for _, tx := range txes {
		txBytes, err := tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint64(sizeBuf, uint64(len(txBytes))+1)
		l2Message = append(l2Message, sizeBuf...)
		l2Message = append(l2Message, arbos.L2MessageKind_SignedTx)
		l2Message = append(l2Message, txBytes...)
	}
	return l2Message, nil
}

func SendSignedTxViaL1(
	t *testing.T,
	ctx context.Context,
	l1info *BlockchainTestInfo,
	l1client *ethclient.Client,
	l2client *ethclient.Client,
	delayedTx *types.Transaction,
) *types.Receipt {
	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)
	usertxopts := l1info.GetDefaultTransactOpts("User", ctx)

	txbytes, err := delayedTx.MarshalBinary()
	Require(t, err)
	txwrapped := append([]byte{arbos.L2MessageKind_SignedTx}, txbytes...)
	l1tx, err := delayedInboxContract.SendL2Message(&usertxopts, txwrapped)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)

	AdvanceL1(t, ctx, l1client, l1info, 30)
	receipt, err := EnsureTxSucceeded(ctx, l2client, delayedTx)
	Require(t, err)
	return receipt
}

func SendUnsignedTxViaL1(
	t *testing.T,
	ctx context.Context,
	l1info *BlockchainTestInfo,
	l1client *ethclient.Client,
	l2client *ethclient.Client,
	templateTx *types.Transaction,
) *types.Receipt {
	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)

	usertxopts := l1info.GetDefaultTransactOpts("User", ctx)
	remapped := arbosutil.RemapL1Address(usertxopts.From)
	nonce, err := l2client.NonceAt(ctx, remapped, nil)
	Require(t, err)

	unsignedTx := types.NewTx(&types.ArbitrumUnsignedTx{
		ChainId:   templateTx.ChainId(),
		From:      remapped,
		Nonce:     nonce,
		GasFeeCap: templateTx.GasFeeCap(),
		Gas:       templateTx.Gas(),
		To:        templateTx.To(),
		Value:     templateTx.Value(),
		Data:      templateTx.Data(),
	})

	l1tx, err := delayedInboxContract.SendUnsignedTransaction(
		&usertxopts,
		arbmath.UintToBig(unsignedTx.Gas()),
		unsignedTx.GasFeeCap(),
		arbmath.UintToBig(unsignedTx.Nonce()),
		*unsignedTx.To(),
		unsignedTx.Value(),
		unsignedTx.Data(),
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, l1tx)
	Require(t, err)

	AdvanceL1(t, ctx, l1client, l1info, 30)
	receipt, err := EnsureTxSucceeded(ctx, l2client, unsignedTx)
	Require(t, err)
	return receipt
}

func GetBaseFee(t *testing.T, client *ethclient.Client, ctx context.Context) *big.Int {
	header, err := client.HeaderByNumber(ctx, nil)
	Require(t, err)
	return header.BaseFee
}

func GetBaseFeeAt(t *testing.T, client *ethclient.Client, ctx context.Context, blockNum *big.Int) *big.Int {
	header, err := client.HeaderByNumber(ctx, blockNum)
	Require(t, err)
	return header.BaseFee
}

type lifecycle struct {
	start func() error
	stop  func() error
}

func (l *lifecycle) Start() error {
	if l.start != nil {
		return l.start()
	}
	return nil
}

func (l *lifecycle) Stop() error {
	if l.start != nil {
		return l.stop()
	}
	return nil
}

type staticNodeConfigFetcher struct {
	config *arbnode.Config
}

func NewFetcherFromConfig(c *arbnode.Config) *staticNodeConfigFetcher {
	err := c.Validate()
	if err != nil {
		panic("invalid static config: " + err.Error())
	}
	return &staticNodeConfigFetcher{c}
}

func (c *staticNodeConfigFetcher) Get() *arbnode.Config {
	return c.config
}

func (c *staticNodeConfigFetcher) Start(context.Context) {}

func (c *staticNodeConfigFetcher) StopAndWait() {}

func (c *staticNodeConfigFetcher) Started() bool {
	return true
}

func createRedisGroup(ctx context.Context, t *testing.T, streamName string, client redis.UniversalClient) {
	t.Helper()
	// Stream name and group name are the same.
	if _, err := client.XGroupCreateMkStream(ctx, streamName, streamName, "$").Result(); err != nil {
		log.Debug("Error creating stream group: %v", err)
	}
}

func destroyRedisGroup(ctx context.Context, t *testing.T, streamName string, client redis.UniversalClient) {
	t.Helper()
	if client == nil {
		return
	}
	if _, err := client.XGroupDestroy(ctx, streamName, streamName).Result(); err != nil {
		log.Debug("Error destroying a stream group", "error", err)
	}
}

func createTestValidationNode(t *testing.T, ctx context.Context, config *valnode.Config, spawnerOpts ...server_arb.SpawnerOption) (*valnode.ValidationNode, *node.Node) {
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{server_api.Namespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""
	stackConf.DBEngine = "leveldb" // TODO Try pebble again in future once iterator race condition issues are fixed

	valnode.EnsureValidationExposedViaAuthRPC(&stackConf)

	stack, err := node.New(&stackConf)
	Require(t, err)

	configFetcher := func() *valnode.Config { return config }
	valnode, err := valnode.CreateValidationNode(configFetcher, stack, nil, spawnerOpts...)
	Require(t, err)

	err = stack.Start()
	Require(t, err)

	err = valnode.Start(ctx)
	Require(t, err)

	t.Cleanup(func() {
		stack.Close()
		valnode.Stop()
	})

	return valnode, stack
}

type validated interface {
	Validate() error
}

func StaticFetcherFrom[T any](t *testing.T, config *T) func() *T {
	t.Helper()
	tCopy := *config
	asEmptyIf := interface{}(&tCopy)
	if asValidtedIf, ok := asEmptyIf.(validated); ok {
		err := asValidtedIf.Validate()
		if err != nil {
			Fatal(t, err)
		}
	}
	return func() *T { return &tCopy }
}

func configByValidationNode(clientConfig *arbnode.Config, valStack *node.Node) {
	clientConfig.BlockValidator.ValidationServerConfigs[0].URL = valStack.WSEndpoint()
	clientConfig.BlockValidator.ValidationServerConfigs[0].JWTSecret = ""
}

func currentRootModule(t *testing.T) common.Hash {
	t.Helper()
	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		t.Fatalf("Error creating machine locator: %v", err)
	}
	return locator.LatestWasmModuleRoot()
}

func AddValNodeIfNeeded(t *testing.T, ctx context.Context, nodeConfig *arbnode.Config, useJit bool, redisURL string, wasmRootDir string) {
	if !nodeConfig.ValidatorRequired() || nodeConfig.BlockValidator.ValidationServerConfigs[0].URL != "" {
		return
	}
	AddValNode(t, ctx, nodeConfig, useJit, redisURL, wasmRootDir)
}

func AddValNode(t *testing.T, ctx context.Context, nodeConfig *arbnode.Config, useJit bool, redisURL string, wasmRootDir string) {
	conf := valnode.TestValidationConfig
	conf.UseJit = useJit
	conf.Wasm.RootPath = wasmRootDir
	DontWaitAndRun(ctx, 2, t.Name())
	// Enable redis streams when URL is specified
	if redisURL != "" {
		conf.Arbitrator.RedisValidationServerConfig = rediscons.TestValidationServerConfig
		redisClient, err := redisutil.RedisClientFromURL(redisURL)
		if err != nil {
			t.Fatalf("Error creating redis coordinator: %v", err)
		}
		redisStream := server_api.RedisStreamForRoot(rediscons.TestValidationServerConfig.StreamPrefix, currentRootModule(t))
		createRedisGroup(ctx, t, redisStream, redisClient)
		conf.Arbitrator.RedisValidationServerConfig.RedisURL = redisURL
		t.Cleanup(func() { destroyRedisGroup(ctx, t, redisStream, redisClient) })
		conf.Arbitrator.RedisValidationServerConfig.ModuleRoots = []string{currentRootModule(t).Hex()}
	}
	_, valStack := createTestValidationNode(t, ctx, &conf)
	configByValidationNode(nodeConfig, valStack)
}

func createTestL1BlockChain(t *testing.T, l1info info, withClientWrapper bool) (info, *ethclient.Client, *eth.Ethereum, *node.Node, *ClientWrapper) {
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	stackConfig := testhelpers.CreateStackConfigForTest("")
	l1info.GenerateAccount("Faucet")
	for _, acct := range DefaultChainAccounts {
		l1info.GenerateAccount(acct)
	}

	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams = params.ArbitrumChainParams{}

	stack, err := node.New(stackConfig)
	Require(t, err)

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = chainConfig.ChainID.Uint64()
	faucetAddr := l1info.GetAddress("Faucet")
	l1Genesis := core.DeveloperGenesisBlock(15_000_000, &faucetAddr)

	// Pre-fund with large values some common accounts
	infoGenesis := l1info.GetGenesisAlloc()
	bigBalance := big.NewInt(0).SetUint64(9223372036854775807)
	for _, acct := range DefaultChainAccounts {
		addr := l1info.GetAddress(acct)
		if l1Genesis.Alloc[addr].Balance == nil {
			l1Genesis.Alloc[addr] = types.Account{Balance: bigBalance}
		} else {
			l1Genesis.Alloc[addr].Balance.Add(l1Genesis.Alloc[addr].Balance, bigBalance)
		}
	}
	for acct, info := range infoGenesis {
		l1Genesis.Alloc[acct] = info
	}
	l1Genesis.BaseFee = big.NewInt(50 * params.GWei)
	nodeConf.Genesis = l1Genesis
	nodeConf.Miner.Etherbase = l1info.GetAddress("Faucet")
	nodeConf.Miner.PendingFeeRecipient = l1info.GetAddress("Faucet")
	nodeConf.SyncMode = ethconfig.FullSync

	l1backend, err := eth.New(stack, &nodeConf)
	Require(t, err)

	simBeacon, err := catalyst.NewSimulatedBeacon(0, common.Address{}, l1backend)
	Require(t, err)
	catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
	stack.RegisterLifecycle(simBeacon)

	tempKeyStore := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	faucetAccount, err := tempKeyStore.ImportECDSA(l1info.Accounts["Faucet"].PrivateKey, "passphrase")
	Require(t, err)
	Require(t, tempKeyStore.Unlock(faucetAccount, "passphrase"))
	l1backend.AccountManager().AddBackend(tempKeyStore)

	stack.RegisterLifecycle(&lifecycle{stop: func() error {
		return l1backend.Stop()
	}})

	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filters.NewFilterSystem(l1backend.APIBackend, filters.Config{})),
	}})
	stack.RegisterAPIs(tracers.APIs(l1backend.APIBackend))

	Require(t, stack.Start())

	var rpcClient rpc.ClientInterface = stack.Attach()
	var clientWrapper *ClientWrapper
	if withClientWrapper {
		clientWrapper = NewClientWrapper(rpcClient, l1info)
		rpcClient = clientWrapper
	}

	l1Client := ethclient.NewClient(rpcClient)

	return l1info, l1Client, l1backend, stack, clientWrapper
}

func getInitMessage(ctx context.Context, t *testing.T, parentChainClient *ethclient.Client, addresses *chaininfo.RollupAddresses) *arbostypes.ParsedInitMessage {
	bridge, err := arbnode.NewDelayedBridge(parentChainClient, addresses.Bridge, addresses.DeployedAt)
	Require(t, err)
	deployedAtBig := arbmath.UintToBig(addresses.DeployedAt)
	messages, err := bridge.LookupMessagesInRange(ctx, deployedAtBig, deployedAtBig, nil)
	Require(t, err)
	if len(messages) == 0 {
		Fatal(t, "No delayed messages found at rollup creation block")
	}
	initMessage, err := messages[0].Message.ParseInitMessage()
	Require(t, err, "Failed to parse rollup init message")

	return initMessage
}

var (
	blockChallengeLeafHeight     = uint64(1 << 5) // 32
	bigStepChallengeLeafHeight   = uint64(1 << 10)
	smallStepChallengeLeafHeight = uint64(1 << 10)
)

func deployOnParentChain(
	t *testing.T,
	ctx context.Context,
	parentChainInfo info,
	parentChainClient *ethclient.Client,
	parentChainReaderConfig *headerreader.Config,
	chainConfig *params.ChainConfig,
	wasmModuleRoot common.Hash,
	prodConfirmPeriodBlocks bool,
	chainSupportsBlobs bool,
	deployBold bool,
	delayBufferThreshold uint64,
) (*chaininfo.RollupAddresses, *arbostypes.ParsedInitMessage) {
	var fundingTxs []*types.Transaction
	for _, acct := range DefaultChainAccounts {
		if !parentChainInfo.HasAccount(acct) {
			parentChainInfo.GenerateAccount(acct)
			fundingTxs = append(fundingTxs, parentChainInfo.PrepareTx("Faucet", acct, parentChainInfo.TransferGas, big.NewInt(9223372036854775807), nil))
		}
	}
	if len(fundingTxs) > 0 {
		// TODO(NIT-3910): Use ArbosInitializationInfo to fund accounts at genesis instead of sending transactions
		SendWaitTestTransactions(t, ctx, parentChainClient, fundingTxs)
	}

	parentChainTransactionOpts := parentChainInfo.GetDefaultTransactOpts("RollupOwner", ctx)
	serializedChainConfig, err := json.Marshal(chainConfig)
	Require(t, err)

	arbSys, _ := precompilesgen.NewArbSys(types.ArbSysAddress, parentChainClient)
	parentChainReader, err := headerreader.New(ctx, parentChainClient, func() *headerreader.Config { return parentChainReaderConfig }, arbSys)
	Require(t, err)
	parentChainReader.Start(ctx)
	defer parentChainReader.StopAndWait()

	nativeToken := common.Address{}
	maxDataSize := big.NewInt(117964)
	var addresses *chaininfo.RollupAddresses
	if deployBold {
		stakeToken, tx, _, err := localgen.DeployTestWETH9(
			&parentChainTransactionOpts,
			parentChainReader.Client(),
			"Weth",
			"WETH",
		)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, parentChainReader.Client(), tx)
		Require(t, err)
		miniStakeValues := []*big.Int{big.NewInt(5), big.NewInt(4), big.NewInt(3), big.NewInt(2), big.NewInt(1)}
		genesisExecutionState := rollupgen.AssertionState{
			GlobalState:    rollupgen.GlobalState{},
			MachineStatus:  1, // Finished
			EndHistoryRoot: [32]byte{},
		}
		bufferConfig := rollupgen.BufferConfig{
			Threshold:            delayBufferThreshold, // number of blocks
			Max:                  14400,                // 2 days of blocks
			ReplenishRateInBasis: 500,                  // 5%
		}
		cfg := rollupgen.Config{
			MiniStakeValues:        miniStakeValues,
			ConfirmPeriodBlocks:    120,
			StakeToken:             stakeToken,
			BaseStake:              big.NewInt(1),
			WasmModuleRoot:         wasmModuleRoot,
			Owner:                  parentChainTransactionOpts.From,
			LoserStakeEscrow:       parentChainTransactionOpts.From,
			MinimumAssertionPeriod: big.NewInt(75),
			ValidatorAfkBlocks:     201600,
			ChainId:                chainConfig.ChainID,
			ChainConfig:            string(serializedChainConfig),
			SequencerInboxMaxTimeVariation: rollupgen.ISequencerInboxMaxTimeVariation{
				DelayBlocks:   big.NewInt(60 * 60 * 24 / 15),
				FutureBlocks:  big.NewInt(12),
				DelaySeconds:  big.NewInt(60 * 60 * 24),
				FutureSeconds: big.NewInt(60 * 60),
			},
			LayerZeroBlockEdgeHeight:     new(big.Int).SetUint64(blockChallengeLeafHeight),
			LayerZeroBigStepEdgeHeight:   new(big.Int).SetUint64(bigStepChallengeLeafHeight),
			LayerZeroSmallStepEdgeHeight: new(big.Int).SetUint64(smallStepChallengeLeafHeight),
			GenesisAssertionState:        genesisExecutionState,
			GenesisInboxCount:            common.Big0,
			AnyTrustFastConfirmer:        common.Address{},
			NumBigStepLevel:              3,
			ChallengeGracePeriodBlocks:   3,
			BufferConfig:                 bufferConfig,
		}
		wrappedClient := butil.NewBackendWrapper(parentChainReader.Client(), rpc.LatestBlockNumber)
		boldAddresses, err := setup.DeployFullRollupStack(
			ctx,
			wrappedClient,
			&parentChainTransactionOpts,
			parentChainInfo.GetAddress("Sequencer"),
			cfg,
			setup.RollupStackConfig{
				UseMockBridge:          false,
				UseMockOneStepProver:   false,
				UseBlobs:               chainSupportsBlobs,
				MinimumAssertionPeriod: 0,
			},
		)
		Require(t, err)
		addresses = &chaininfo.RollupAddresses{
			Bridge:                 boldAddresses.Bridge,
			Inbox:                  boldAddresses.Inbox,
			SequencerInbox:         boldAddresses.SequencerInbox,
			Rollup:                 boldAddresses.Rollup,
			NativeToken:            nativeToken,
			UpgradeExecutor:        boldAddresses.UpgradeExecutor,
			ValidatorUtils:         boldAddresses.ValidatorUtils,
			ValidatorWalletCreator: boldAddresses.ValidatorWalletCreator,
			StakeToken:             stakeToken,
			DeployedAt:             boldAddresses.DeployedAt,
		}
	} else {
		addresses, err = deploy.DeployLegacyOnParentChain(
			ctx,
			parentChainReader,
			&parentChainTransactionOpts,
			[]common.Address{parentChainInfo.GetAddress("Sequencer")},
			parentChainInfo.GetAddress("RollupOwner"),
			0,
			deploy.GenerateLegacyRollupConfig(prodConfirmPeriodBlocks, wasmModuleRoot, parentChainInfo.GetAddress("RollupOwner"), chainConfig, serializedChainConfig, common.Address{}),
			nativeToken,
			maxDataSize,
			chainSupportsBlobs,
		)
	}
	Require(t, err)
	parentChainInfo.SetContract("Bridge", addresses.Bridge)
	parentChainInfo.SetContract("SequencerInbox", addresses.SequencerInbox)
	parentChainInfo.SetContract("Inbox", addresses.Inbox)
	parentChainInfo.SetContract("UpgradeExecutor", addresses.UpgradeExecutor)
	initMessage := getInitMessage(ctx, t, parentChainClient, addresses)
	return addresses, initMessage
}

func createNonL1BlockChainWithStackConfig(
	t *testing.T, info *BlockchainTestInfo, dataDir string, chainConfig *params.ChainConfig, arbOSInit *params.ArbOSInit, initMessage *arbostypes.ParsedInitMessage, stackConfig *node.Config, execConfig *gethexec.Config, ignoreExecConfigValidationError bool,
) (*BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	if info == nil {
		info = NewArbTestInfo(t, chainConfig.ChainID)
	}
	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(dataDir)
	}
	if execConfig == nil {
		execConfig = ExecConfigDefaultTest(t, env.GetTestStateScheme())
	}

	validateExecConfig(t, execConfig, ignoreExecConfigValidationError)

	stack, err := node.New(stackConfig)
	Require(t, err)

	chainData, err := stack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{MetricsNamespace: "l2chaindata/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata")})
	Require(t, err)

	wasmData, err := stack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("wasm")})
	Require(t, err)

	chainDb := rawdb.WrapDatabaseWithWasm(chainData, wasmData)
	arbDb, err := stack.OpenDatabaseWithOptions("arbitrumdata", node.DatabaseOptions{MetricsNamespace: "arbitrumdata/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("arbitrumdata")})
	Require(t, err)

	initReader := statetransfer.NewMemoryInitDataReader(&info.ArbInitData)
	if initMessage == nil {
		serializedChainConfig, err := json.Marshal(chainConfig)
		Require(t, err)
		initMessage = &arbostypes.ParsedInitMessage{
			ChainId:               chainConfig.ChainID,
			InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
			ChainConfig:           chainConfig,
			SerializedChainConfig: serializedChainConfig,
		}
	}
	coreCacheConfig := gethexec.DefaultCacheConfigFor(&execConfig.Caching)
	blockchain, err := gethexec.WriteOrTestBlockChain(chainDb, coreCacheConfig, initReader, chainConfig, arbOSInit, nil, initMessage, &gethexec.ConfigDefault.TxIndexer, 0)
	Require(t, err)

	return info, stack, chainDb, arbDb, blockchain
}

func ClientForStack(t *testing.T, backend *node.Node) *ethclient.Client {
	rpcClient := backend.Attach()
	return ethclient.NewClient(rpcClient)
}

func StartWatchChanErr(t *testing.T, ctx context.Context, feedErrChan chan error, node *arbnode.Node) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case err := <-feedErrChan:
			t.Errorf("error occurred: %v", err)
			if node != nil {
				node.StopAndWait()
			}
		}
	}()
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fatal(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

func CheckEqual[T any](t *testing.T, want T, got T, printables ...interface{}) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		testhelpers.FailImpl(t, "wrong result, want ", want, ", got ", got, printables)
	}
}

func Create2ndNodeWithConfig(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	parentChainStack *node.Node,
	parentChainInfo *BlockchainTestInfo,
	chainInitData *statetransfer.ArbosInitializationInfo,
	nodeConfig *arbnode.Config,
	execConfig *gethexec.Config,
	stackConfig *node.Config,
	valnodeConfig *valnode.Config,
	addresses *chaininfo.RollupAddresses,
	initMessage *arbostypes.ParsedInitMessage,
	useExecutionClientOnly bool,
	ignoreExecConfigValidationError bool,
) (*ethclient.Client, *arbnode.Node) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if execConfig == nil {
		t.Fatal("should not be nil")
	}
	validateExecConfig(t, execConfig, ignoreExecConfigValidationError)

	feedErrChan := make(chan error, 10)
	parentChainRpcClient := parentChainStack.Attach()
	parentChainClient := ethclient.NewClient(parentChainRpcClient)

	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(t.TempDir())
	}
	chainStack, err := node.New(stackConfig)
	Require(t, err)

	chainData, err := chainStack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{MetricsNamespace: "l2chaindata/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata")})
	Require(t, err)
	wasmData, err := chainStack.OpenDatabaseWithOptions("wasm", node.DatabaseOptions{MetricsNamespace: "wasm/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("wasm")})
	Require(t, err)
	chainDb := rawdb.WrapDatabaseWithWasm(chainData, wasmData)

	arbDb, err := chainStack.OpenDatabaseWithOptions("arbitrumdata", node.DatabaseOptions{MetricsNamespace: "arbitrumdata/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("arbitrumdata")})
	Require(t, err)
	initReader := statetransfer.NewMemoryInitDataReader(chainInitData)

	dataSigner := signature.DataSignerFromPrivateKey(parentChainInfo.GetInfoWithPrivKey("Sequencer").PrivateKey)
	sequencerTxOpts := parentChainInfo.GetDefaultTransactOpts("Sequencer", ctx)
	validatorTxOpts := parentChainInfo.GetDefaultTransactOpts("Validator", ctx)
	firstExec := getExecNode(t, first)

	chainConfig := firstExec.ArbInterface.BlockChain().Config()

	coreCacheConfig := gethexec.DefaultCacheConfigFor(&execConfig.Caching)
	var tracer *tracing.Hooks
	if execConfig.VmTrace.TracerName != "" {
		tracer, err = tracers.LiveDirectory.New(execConfig.VmTrace.TracerName, json.RawMessage(execConfig.VmTrace.JSONConfig))
		Require(t, err)
	}
	blockchain, err := gethexec.WriteOrTestBlockChain(chainDb, coreCacheConfig, initReader, chainConfig, nil, tracer, initMessage, &execConfig.TxIndexer, 0)
	Require(t, err)

	AddValNodeIfNeeded(t, ctx, nodeConfig, true, "", valnodeConfig.Wasm.RootPath)

	Require(t, nodeConfig.Validate())
	configFetcher := func() *gethexec.Config { return execConfig }
	currentExec, err := gethexec.CreateExecutionNode(ctx, chainStack, chainDb, blockchain, parentChainClient, configFetcher, big.NewInt(1337), 0)
	Require(t, err)

	var currentNode *arbnode.Node
	locator, err := server_common.NewMachineLocator(valnodeConfig.Wasm.RootPath)
	Require(t, err)
	if useExecutionClientOnly {
		currentNode, err = arbnode.CreateNodeExecutionClient(ctx, chainStack, currentExec, arbDb, NewFetcherFromConfig(nodeConfig), blockchain.Config(), parentChainClient, addresses, &validatorTxOpts, &sequencerTxOpts, dataSigner, feedErrChan, big.NewInt(1337), nil, locator.LatestWasmModuleRoot())
	} else {
		currentNode, err = arbnode.CreateNodeFullExecutionClient(ctx, chainStack, currentExec, currentExec, currentExec, currentExec, arbDb, NewFetcherFromConfig(nodeConfig), blockchain.Config(), parentChainClient, addresses, &validatorTxOpts, &sequencerTxOpts, dataSigner, feedErrChan, big.NewInt(1337), nil, locator.LatestWasmModuleRoot())
	}

	Require(t, err)

	err = currentNode.Start(ctx)
	Require(t, err)
	chainClient := ClientForStack(t, chainStack)

	StartWatchChanErr(t, ctx, feedErrChan, currentNode)

	return chainClient, currentNode
}

func GetBalance(t *testing.T, ctx context.Context, client *ethclient.Client, account common.Address) *big.Int {
	t.Helper()
	balance, err := client.BalanceAt(ctx, account, nil)
	Require(t, err, "could not get balance")
	return balance
}

func requireClose(t *testing.T, s *node.Node, text ...interface{}) {
	t.Helper()
	Require(t, s.Close(), text...)
}

func authorizeDASKeyset(
	t *testing.T,
	ctx context.Context,
	dasSignerKey *blsSignatures.PublicKey,
	l1info info,
	l1client *ethclient.Client,
) {
	if dasSignerKey == nil {
		return
	}
	keyset := &dasutil.DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{*dasSignerKey},
	}
	wr := bytes.NewBuffer([]byte{})
	err := keyset.Serialize(wr)
	Require(t, err, "unable to serialize DAS keyset")
	keysetBytes := wr.Bytes()

	sequencerInboxABI, err := abi.JSON(strings.NewReader(bridgegen.SequencerInboxABI))
	Require(t, err, "unable to parse sequencer inbox ABI")
	setKeysetCalldata, err := sequencerInboxABI.Pack("setValidKeyset", keysetBytes)
	Require(t, err, "unable to generate calldata")

	upgradeExecutor, err := upgrade_executorgen.NewUpgradeExecutor(l1info.Accounts["UpgradeExecutor"].Address, l1client)
	Require(t, err, "unable to bind upgrade executor")

	trOps := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	tx, err := upgradeExecutor.ExecuteCall(&trOps, l1info.Accounts["SequencerInbox"].Address, setKeysetCalldata)
	Require(t, err, "unable to set valid keyset")

	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err, "unable to ensure transaction success for setting valid keyset")
}

func setupConfigWithDAS(
	t *testing.T, ctx context.Context, dasModeString string,
) (*params.ChainConfig, *arbnode.Config, *das.LifecycleManager, string, *blsSignatures.PublicKey) {
	l1NodeConfigA := arbnode.ConfigDefaultL1Test()
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	var dbPath string
	var err error

	enableFileStorage, enableDas := false, true
	switch dasModeString {
	case "files":
		enableFileStorage = true
		chainConfig = chaininfo.ArbitrumDevTestDASChainConfig()
	case "onchain":
		enableDas = false
	default:
		Fatal(t, "unknown storage type")
	}
	dbPath = t.TempDir()
	dasSignerKey, _, err := das.GenerateAndStoreKeys(dbPath)
	Require(t, err)

	dasConfig := &das.DataAvailabilityConfig{
		Enable: enableDas,
		Key: das.KeyConfig{
			KeyDir: dbPath,
		},
		LocalFileStorage: das.LocalFileStorageConfig{
			Enable:  enableFileStorage,
			DataDir: dbPath,
		},
		RequestTimeout:           5 * time.Second,
		ParentChainNodeURL:       "none",
		SequencerInboxAddress:    "none",
		PanicOnError:             true,
		DisableSignatureChecking: true,
	}

	l1NodeConfigA.DataAvailability = das.DefaultDataAvailabilityConfig
	var lifecycleManager *das.LifecycleManager
	var daReader dasutil.DASReader
	var daWriter dasutil.DASWriter
	var daHealthChecker das.DataAvailabilityServiceHealthChecker
	var signatureVerifier *das.SignatureVerifier
	if dasModeString != "onchain" {
		daReader, daWriter, signatureVerifier, daHealthChecker, lifecycleManager, err = das.CreateDAComponentsForDaserver(ctx, dasConfig, nil, nil)

		Require(t, err)
		rpcLis, err := net.Listen("tcp", "localhost:0")
		Require(t, err)
		restLis, err := net.Listen("tcp", "localhost:0")
		Require(t, err)
		_, err = das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, daReader, daWriter, daHealthChecker, signatureVerifier)
		Require(t, err)
		_, err = das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, daReader, daHealthChecker)
		Require(t, err)

		beConfigA := das.BackendConfig{
			URL:    "http://" + rpcLis.Addr().String(),
			Pubkey: blsPubToBase64(dasSignerKey),
		}
		l1NodeConfigA.DataAvailability.RPCAggregator = aggConfigForBackend(beConfigA)
		l1NodeConfigA.DataAvailability.Enable = true
		l1NodeConfigA.DataAvailability.RestAggregator = das.DefaultRestfulClientAggregatorConfig
		l1NodeConfigA.DataAvailability.RestAggregator.Enable = true
		l1NodeConfigA.DataAvailability.RestAggregator.Urls = []string{"http://" + restLis.Addr().String()}
		l1NodeConfigA.DataAvailability.ParentChainNodeURL = "none"
	}

	return chainConfig, l1NodeConfigA, lifecycleManager, dbPath, dasSignerKey
}

func getDeadlineTimeout(t *testing.T, defaultTimeout time.Duration) time.Duration {
	testDeadLine, deadlineExist := t.Deadline()
	var timeout time.Duration
	if deadlineExist {
		timeout = time.Until(testDeadLine) - (time.Second * 10)
		if timeout > time.Second*10 {
			timeout = timeout - (time.Second * 10)
		}
	} else {
		timeout = defaultTimeout
	}

	return timeout
}

func deployBigMap(t *testing.T, ctx context.Context, auth bind.TransactOpts, client *ethclient.Client,
) (common.Address, *localgen.BigMap) {
	addr, tx, bigMap, err := localgen.DeployBigMap(&auth, client)
	Require(t, err, "could not deploy BigMap.sol contract")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	return addr, bigMap
}

func deploySimple(
	t *testing.T, ctx context.Context, auth bind.TransactOpts, client *ethclient.Client,
) (common.Address, *localgen.Simple) {
	addr, tx, simple, err := localgen.DeploySimple(&auth, client)
	Require(t, err, "could not deploy Simple.sol contract")
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	return addr, simple
}

func deployContractInitCode(code []byte, revert bool) []byte {
	// a small prelude to return the given contract code
	last_opcode := vm.RETURN
	if revert {
		last_opcode = vm.REVERT
	}
	deploy := []byte{byte(vm.PUSH32)}
	deploy = append(deploy, math.U256Bytes(big.NewInt(int64(len(code))))...)
	deploy = append(deploy, byte(vm.DUP1))
	deploy = append(deploy, byte(vm.PUSH1))
	deploy = append(deploy, 42) // the prelude length
	deploy = append(deploy, byte(vm.PUSH1))
	deploy = append(deploy, 0)
	deploy = append(deploy, byte(vm.CODECOPY))
	deploy = append(deploy, byte(vm.PUSH1))
	deploy = append(deploy, 0)
	deploy = append(deploy, byte(last_opcode))
	deploy = append(deploy, code...)
	return deploy
}

func deployContract(
	t *testing.T, ctx context.Context, auth bind.TransactOpts, client *ethclient.Client, code []byte,
) common.Address {
	deploy := deployContractInitCode(code, false)
	basefee := arbmath.BigMulByFrac(GetBaseFee(t, client, ctx), 6, 5) // current*1.2
	nonce, err := client.NonceAt(ctx, auth.From, nil)
	Require(t, err)
	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:      auth.From,
		GasPrice:  basefee,
		GasTipCap: auth.GasTipCap,
		Value:     big.NewInt(0),
		Data:      deploy,
	})
	Require(t, err)
	tx := types.NewContractCreation(nonce, big.NewInt(0), gas, basefee, deploy)
	tx, err = auth.Signer(auth.From, tx)
	Require(t, err)
	Require(t, client.SendTransaction(ctx, tx))
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
	return crypto.CreateAddress(auth.From, nonce)
}

func sendContractCall(
	t *testing.T, ctx context.Context, to common.Address, client *ethclient.Client, data []byte,
) []byte {
	t.Helper()
	msg := ethereum.CallMsg{
		To:    &to,
		Value: big.NewInt(0),
		Data:  data,
	}
	res, err := client.CallContract(ctx, msg, nil)
	Require(t, err)
	return res
}

func doUntil(t *testing.T, delay time.Duration, max int, lambda func() bool) {
	t.Helper()
	for i := 0; i < max; i++ {
		if lambda() {
			return
		}
		time.Sleep(delay)
	}
	Fatal(t, "failed to complete after ", delay*time.Duration(max))
}

func initDefaultTestLog() {
	flag.Parse()
	if *testflag.LogLevelFlag != "" {
		logLevel, err := strconv.ParseInt(*testflag.LogLevelFlag, 10, 32)
		if err != nil || logLevel > int64(log.LevelCrit) {
			log.Warn("-test_loglevel exists but out of bound, ignoring", "logLevel", *testflag.LogLevelFlag, "max", log.LvlTrace)
		}
		glogger := log.NewGlogHandler(
			log.NewTerminalHandler(io.Writer(os.Stderr), false))
		glogger.Verbosity(slog.Level(logLevel))
		log.SetDefault(log.NewLogger(glogger))
	}
}

func TestMain(m *testing.M) {
	initDefaultTestLog()
	initTestCollection()
	code := m.Run()
	os.Exit(code)
}

func getExecNode(t *testing.T, node *arbnode.Node) *gethexec.ExecutionNode {
	t.Helper()
	gethExec, ok := node.ExecutionClient.(*gethexec.ExecutionNode)
	if !ok {
		t.Fatal("failed to get exec node from arbnode")
	}
	return gethExec
}

func logParser[T any](t *testing.T, source string, name string) func(*types.Log) *T {
	parser := arbosutil.NewLogParser[T](source, name)
	return func(log *types.Log) *T {
		t.Helper()
		event, err := parser(log)
		Require(t, err, "failed to parse log")
		return event
	}
}

// recordBlock writes a json file with all of the data needed to validate a block.
//
// This can be used as an input to the arbitrator prover to validate a block.
func recordBlock(t *testing.T, block uint64, builder *NodeBuilder, targets ...rawdb.WasmTarget) {
	t.Helper()
	if !*testflag.RecordBlockInputsEnable {
		return
	}
	ctx := builder.ctx
	inboxPos := arbutil.MessageIndex(block)
	for {
		time.Sleep(250 * time.Millisecond)
		batches, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		haveMessages, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMessageCount(batches - 1)
		Require(t, err)
		if haveMessages >= inboxPos {
			break
		}
	}
	var options []inputs.WriterOption
	options = append(options, inputs.WithTimestampDirEnabled(*testflag.RecordBlockInputsWithTimestampDirEnabled))
	options = append(options, inputs.WithBlockIdInFileNameEnabled(*testflag.RecordBlockInputsWithBlockIdInFileNameEnabled))
	if *testflag.RecordBlockInputsWithBaseDir != "" {
		options = append(options, inputs.WithBaseDir(*testflag.RecordBlockInputsWithBaseDir))
	}
	if *testflag.RecordBlockInputsWithSlug != "" {
		options = append(options, inputs.WithSlug(*testflag.RecordBlockInputsWithSlug))
	} else {
		options = append(options, inputs.WithSlug(t.Name()))
	}
	validationInputsWriter, err := inputs.NewWriter(options...)
	Require(t, err)
	inputJson, err := builder.L2.ConsensusNode.StatelessBlockValidator.ValidationInputsAt(ctx, inboxPos, targets...)
	if err != nil {
		Fatal(t, "failed to get validation inputs", block, err)
	}
	if err := validationInputsWriter.Write(&inputJson); err != nil {
		Fatal(t, "failed to write validation inputs", block, err)
	}
}

func populateMachineDir(t *testing.T, cr *github.ConsensusRelease) string {
	baseDir := t.TempDir()
	machineDir := baseDir + "/machines"
	err := os.Mkdir(machineDir, 0755)
	Require(t, err)
	err = os.Mkdir(machineDir+"/latest", 0755)
	Require(t, err)
	mrFile, err := os.Create(machineDir + "/latest/module-root.txt")
	Require(t, err)
	_, err = mrFile.WriteString(cr.WavmModuleRoot)
	Require(t, err)
	machResp, err := http.Get(cr.MachineWavmURL.String())
	Require(t, err)
	defer machResp.Body.Close()
	machineFile, err := os.Create(machineDir + "/latest/machine.wavm.br")
	Require(t, err)
	_, err = io.Copy(machineFile, machResp.Body)
	Require(t, err)
	replayResp, err := http.Get(cr.ReplayWasmURL.String())
	Require(t, err)
	defer replayResp.Body.Close()
	replayFile, err := os.Create(machineDir + "/latest/replay.wasm")
	Require(t, err)
	_, err = io.Copy(replayFile, replayResp.Body)
	Require(t, err)
	return machineDir
}

// will call foo with specified interval, until foo returns true or specified timeout elapses
// if timeout elapses fails with t.Fatal with timeoutMessage appended to the message
// note: use pollWithDeadlineDefault if you don't care much about the interval and timeout, should make it easier to globally tune the tests
func pollWithDeadline(t *testing.T, interval time.Duration, timeout time.Duration, foo func() bool) bool {
	t.Helper()
	deadline := time.After(timeout)
	for !foo() {
		select {
		case <-deadline:
			return false
		case <-time.After(interval):
		}
	}
	return true
}

func pollWithDeadlineDefault(t *testing.T, foo func() bool) bool {
	return pollWithDeadline(t, 20*time.Millisecond, 5*time.Second, foo)
}

// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/deploy"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"
	rediscons "github.com/offchainlabs/nitro/validator/valnode/redis"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
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
	"github.com/offchainlabs/nitro/arbutil"
	_ "github.com/offchainlabs/nitro/execution/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/solgen/go/upgrade_executorgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
	"github.com/offchainlabs/nitro/util/testhelpers/github"
	"golang.org/x/exp/slog"
)

type info = *BlockchainTestInfo
type client = arbutil.L1Interface

type SecondNodeParams struct {
	nodeConfig  *arbnode.Config
	execConfig  *gethexec.Config
	stackConfig *node.Config
	dasConfig   *das.DataAvailabilityConfig
	initData    *statetransfer.ArbosInitializationInfo
	addresses   *chaininfo.RollupAddresses
}

type TestClient struct {
	ctx           context.Context
	Client        *ethclient.Client
	L1Backend     *eth.Ethereum
	Stack         *node.Node
	ConsensusNode *arbnode.Node
	ExecNode      *gethexec.ExecutionNode

	// having cleanup() field makes cleanup customizable from default cleanup methods after calling build
	cleanup func()
}

func NewTestClient(ctx context.Context) *TestClient {
	return &TestClient{ctx: ctx}
}

func (tc *TestClient) SendSignedTx(t *testing.T, l2Client *ethclient.Client, transaction *types.Transaction, lInfo info) *types.Receipt {
	return SendSignedTxViaL1(t, tc.ctx, lInfo, tc.Client, l2Client, transaction)
}

func (tc *TestClient) SendUnsignedTx(t *testing.T, l2Client *ethclient.Client, transaction *types.Transaction, lInfo info) *types.Receipt {
	return SendUnsignedTxViaL1(t, tc.ctx, lInfo, tc.Client, l2Client, transaction)
}

func (tc *TestClient) TransferBalance(t *testing.T, from string, to string, amount *big.Int, lInfo info) (*types.Transaction, *types.Receipt) {
	return TransferBalanceTo(t, from, lInfo.GetAddress(to), amount, lInfo, tc.Client, tc.ctx)
}

func (tc *TestClient) TransferBalanceTo(t *testing.T, from string, to common.Address, amount *big.Int, lInfo info) (*types.Transaction, *types.Receipt) {
	return TransferBalanceTo(t, from, to, amount, lInfo, tc.Client, tc.ctx)
}

func (tc *TestClient) GetBalance(t *testing.T, account common.Address) *big.Int {
	return GetBalance(t, tc.ctx, tc.Client, account)
}

func (tc *TestClient) GetBaseFee(t *testing.T) *big.Int {
	return GetBaseFee(t, tc.Client, tc.ctx)
}

func (tc *TestClient) GetBaseFeeAt(t *testing.T, blockNum *big.Int) *big.Int {
	return GetBaseFeeAt(t, tc.Client, tc.ctx, blockNum)
}

func (tc *TestClient) SendWaitTestTransactions(t *testing.T, txs []*types.Transaction) {
	SendWaitTestTransactions(t, tc.ctx, tc.Client, txs)
}

func (tc *TestClient) DeploySimple(t *testing.T, auth bind.TransactOpts) (common.Address, *mocksgen.Simple) {
	return deploySimple(t, tc.ctx, auth, tc.Client)
}

func (tc *TestClient) EnsureTxSucceeded(transaction *types.Transaction) (*types.Receipt, error) {
	return tc.EnsureTxSucceededWithTimeout(transaction, time.Second*5)
}

func (tc *TestClient) EnsureTxSucceededWithTimeout(transaction *types.Transaction, timeout time.Duration) (*types.Receipt, error) {
	return EnsureTxSucceededWithTimeout(tc.ctx, tc.Client, transaction, timeout)
}

var TestCachingConfig = gethexec.CachingConfig{
	Archive:                            false,
	BlockCount:                         128,
	BlockAge:                           30 * time.Minute,
	TrieTimeLimit:                      time.Hour,
	TrieDirtyCache:                     1024,
	TrieCleanCache:                     600,
	SnapshotCache:                      400,
	DatabaseCache:                      2048,
	SnapshotRestoreGasLimit:            300_000_000_000,
	MaxNumberOfBlocksToSkipStateSaving: 0,
	MaxAmountOfGasToSkipStateSaving:    0,
	StylusLRUCache:                     0,
	StateScheme:                        env.GetTestStateScheme(),
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
	ExpectedSurplusSoftThreshold: "default",
	ExpectedSurplusHardThreshold: "default",
	EnableProfiling:              false,
}

func ExecConfigDefaultNonSequencerTest(t *testing.T) *gethexec.Config {
	config := gethexec.ConfigDefault
	config.Caching = TestCachingConfig
	config.ParentChainReader = headerreader.TestConfig
	config.Sequencer.Enable = false
	config.Forwarder = DefaultTestForwarderConfig
	config.ForwardingTarget = "null"
	config.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessNone

	Require(t, config.Validate())

	return &config
}

func ExecConfigDefaultTest(t *testing.T) *gethexec.Config {
	config := gethexec.ConfigDefault
	config.Caching = TestCachingConfig
	config.Sequencer = TestSequencerConfig
	config.ParentChainReader = headerreader.TestConfig
	config.ForwardingTarget = "null"
	config.TxPreChecker.Strictness = gethexec.TxPreCheckerStrictnessNone

	Require(t, config.Validate())

	return &config
}

type NodeBuilder struct {
	// NodeBuilder configuration
	ctx           context.Context
	chainConfig   *params.ChainConfig
	nodeConfig    *arbnode.Config
	execConfig    *gethexec.Config
	l1StackConfig *node.Config
	l2StackConfig *node.Config
	valnodeConfig *valnode.Config
	l3Config      *NitroConfig
	L1Info        info
	L2Info        info
	L3Info        info

	// L1, L2, L3 Node parameters
	dataDir                     string
	isSequencer                 bool
	takeOwnership               bool
	withL1                      bool
	addresses                   *chaininfo.RollupAddresses
	l3Addresses                 *chaininfo.RollupAddresses
	initMessage                 *arbostypes.ParsedInitMessage
	l3InitMessage               *arbostypes.ParsedInitMessage
	withProdConfirmPeriodBlocks bool

	// Created nodes
	L1 *TestClient
	L2 *TestClient
	L3 *TestClient
}

type NitroConfig struct {
	chainConfig   *params.ChainConfig
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
		ArbitrumChainParams: params.ArbitrumDevTestParams(),
		Clique: &params.CliqueConfig{
			Period: 0,
			Epoch:  0,
		},
	}

	valnodeConfig := valnode.TestValidationConfig
	return &NitroConfig{
		chainConfig:   chainConfig,
		nodeConfig:    arbnode.ConfigDefaultL1Test(),
		execConfig:    ExecConfigDefaultTest(t),
		stackConfig:   testhelpers.CreateStackConfigForTest(t.TempDir()),
		valnodeConfig: &valnodeConfig,

		withProdConfirmPeriodBlocks: false,
		isSequencer:                 true,
	}
}

func NewNodeBuilder(ctx context.Context) *NodeBuilder {
	return &NodeBuilder{ctx: ctx}
}

func (b *NodeBuilder) DefaultConfig(t *testing.T, withL1 bool) *NodeBuilder {
	// most used values across current tests are set here as default
	b.withL1 = withL1
	if withL1 {
		b.isSequencer = true
		b.nodeConfig = arbnode.ConfigDefaultL1Test()
	} else {
		b.takeOwnership = true
		b.nodeConfig = arbnode.ConfigDefaultL2Test()
	}
	b.chainConfig = params.ArbitrumDevTestChainConfig()
	b.L1Info = NewL1TestInfo(t)
	b.L2Info = NewArbTestInfo(t, b.chainConfig.ChainID)
	b.dataDir = t.TempDir()
	b.l1StackConfig = testhelpers.CreateStackConfigForTest(b.dataDir)
	b.l2StackConfig = testhelpers.CreateStackConfigForTest(b.dataDir)
	cp := valnode.TestValidationConfig
	b.valnodeConfig = &cp
	b.execConfig = ExecConfigDefaultTest(t)
	b.l3Config = L3NitroConfigDefaultTest(t)
	return b
}

func (b *NodeBuilder) WithArbOSVersion(arbosVersion uint64) *NodeBuilder {
	newChainConfig := *b.chainConfig
	newChainConfig.ArbitrumChainParams.InitialArbOSVersion = arbosVersion
	b.chainConfig = &newChainConfig
	return b
}

func (b *NodeBuilder) WithProdConfirmPeriodBlocks() *NodeBuilder {
	b.withProdConfirmPeriodBlocks = true
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

func (b *NodeBuilder) Build(t *testing.T) func() {
	b.CheckConfig(t)
	if b.withL1 {
		b.BuildL1(t)
		return b.BuildL2OnL1(t)
	}
	return b.BuildL2(t)
}

func (b *NodeBuilder) CheckConfig(t *testing.T) {
	if b.chainConfig == nil {
		b.chainConfig = params.ArbitrumDevTestChainConfig()
	}
	if b.nodeConfig == nil {
		b.nodeConfig = arbnode.ConfigDefaultL1Test()
	}
	if b.execConfig == nil {
		b.execConfig = ExecConfigDefaultTest(t)
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
	b.L1 = NewTestClient(b.ctx)
	b.L1Info, b.L1.Client, b.L1.L1Backend, b.L1.Stack = createTestL1BlockChain(t, b.L1Info)
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
	stackConfig *node.Config,
	execConfig *gethexec.Config,
	nodeConfig *arbnode.Config,
	valnodeConfig *valnode.Config,
	isSequencer bool,
	chainInfo info,

	initMessage *arbostypes.ParsedInitMessage,
	addresses *chaininfo.RollupAddresses,
) *TestClient {
	if parentChainTestClient == nil {
		t.Fatal("must build parent chain before building chain")
	}

	chainTestClient := NewTestClient(ctx)

	var chainDb ethdb.Database
	var arbDb ethdb.Database
	var blockchain *core.BlockChain
	_, chainTestClient.Stack, chainDb, arbDb, blockchain = createNonL1BlockChainWithStackConfig(
		t, chainInfo, dataDir, chainConfig, initMessage, stackConfig, execConfig)

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

	Require(t, execConfig.Validate())
	execConfigToBeUsedInConfigFetcher := execConfig
	execConfigFetcher := func() *gethexec.Config { return execConfigToBeUsedInConfigFetcher }
	execNode, err := gethexec.CreateExecutionNode(ctx, chainTestClient.Stack, chainDb, blockchain, parentChainTestClient.Client, execConfigFetcher)
	Require(t, err)

	fatalErrChan := make(chan error, 10)
	chainTestClient.ConsensusNode, err = arbnode.CreateNode(
		ctx, chainTestClient.Stack, execNode, arbDb, NewFetcherFromConfig(nodeConfig), blockchain.Config(), parentChainTestClient.Client,
		addresses, validatorTxOptsPtr, sequencerTxOptsPtr, dataSigner, fatalErrChan, parentChainId, nil)
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
	)

	b.L3 = buildOnParentChain(
		t,
		b.ctx,

		b.dataDir,

		b.L2Info,
		b.L2,
		b.chainConfig.ChainID,

		b.l3Config.chainConfig,
		b.l3Config.stackConfig,
		b.l3Config.execConfig,
		b.l3Config.nodeConfig,
		b.l3Config.valnodeConfig,
		b.l3Config.isSequencer,
		b.L3Info,

		b.l3InitMessage,
		b.l3Addresses,
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
		b.l2StackConfig,
		b.execConfig,
		b.nodeConfig,
		b.valnodeConfig,
		b.isSequencer,
		b.L2Info,

		b.initMessage,
		b.addresses,
	)

	return func() {
		b.L2.cleanup()
		if b.L1 != nil && b.L1.cleanup != nil {
			b.L1.cleanup()
		}
	}
}

// L2 -Only. Enough for tests that needs no interface to L1
// Requires precompiles.AllowDebugPrecompiles = true
func (b *NodeBuilder) BuildL2(t *testing.T) func() {
	b.L2 = NewTestClient(b.ctx)

	AddValNodeIfNeeded(t, b.ctx, b.nodeConfig, true, "", b.valnodeConfig.Wasm.RootPath)

	var chainDb ethdb.Database
	var arbDb ethdb.Database
	var blockchain *core.BlockChain
	b.L2Info, b.L2.Stack, chainDb, arbDb, blockchain = createL2BlockChain(
		t, b.L2Info, b.dataDir, b.chainConfig, b.execConfig)

	Require(t, b.execConfig.Validate())
	execConfig := b.execConfig
	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(b.ctx, b.L2.Stack, chainDb, blockchain, nil, execConfigFetcher)
	Require(t, err)

	fatalErrChan := make(chan error, 10)
	b.L2.ConsensusNode, err = arbnode.CreateNode(
		b.ctx, b.L2.Stack, execNode, arbDb, NewFetcherFromConfig(b.nodeConfig), blockchain.Config(),
		nil, nil, nil, nil, nil, fatalErrChan, big.NewInt(1337), nil)
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
	return func() { b.L2.cleanup() }
}

// L2 -Only. RestartL2Node shutdowns the existing l2 node and start it again using the same data dir.
func (b *NodeBuilder) RestartL2Node(t *testing.T) {
	if b.L2 == nil {
		t.Fatalf("L2 was not created")
	}
	b.L2.cleanup()

	l2info, stack, chainDb, arbDb, blockchain := createNonL1BlockChainWithStackConfig(t, b.L2Info, b.dataDir, b.chainConfig, b.initMessage, b.l2StackConfig, b.execConfig)

	execConfigFetcher := func() *gethexec.Config { return b.execConfig }
	execNode, err := gethexec.CreateExecutionNode(b.ctx, stack, chainDb, blockchain, nil, execConfigFetcher)
	Require(t, err)

	feedErrChan := make(chan error, 10)
	currentNode, err := arbnode.CreateNode(b.ctx, stack, execNode, arbDb, NewFetcherFromConfig(b.nodeConfig), blockchain.Config(), nil, nil, nil, nil, nil, feedErrChan, big.NewInt(1337), nil)
	Require(t, err)

	Require(t, currentNode.Start(b.ctx))
	client := ClientForStack(t, stack)

	StartWatchChanErr(t, b.ctx, feedErrChan, currentNode)

	l2 := NewTestClient(b.ctx)
	l2.ConsensusNode = currentNode
	l2.Client = client
	l2.ExecNode = execNode
	l2.cleanup = func() { b.L2.ConsensusNode.StopAndWait() }

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
		Create2ndNodeWithConfig(t, ctx, firstNodeTestClient.ConsensusNode, parentChainTestClient.Stack, parentChainInfo, params.initData, params.nodeConfig, params.execConfig, params.stackConfig, valnodeConfig, params.addresses, initMessage)
	testClient.ExecNode = getExecNode(t, testClient.ConsensusNode)
	testClient.cleanup = func() { testClient.ConsensusNode.StopAndWait() }
	return testClient, func() { testClient.cleanup() }
}

func (b *NodeBuilder) Build2ndNode(t *testing.T, params *SecondNodeParams) (*TestClient, func()) {
	if b.L2 == nil {
		t.Fatal("builder did not previously built an L2 Node")
	}
	if b.withL1 && b.L1 == nil {
		t.Fatal("builder did not previously built an L1 Node")
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
	)
}

func (b *NodeBuilder) Build2ndNodeOnL3(t *testing.T, params *SecondNodeParams) (*TestClient, func()) {
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
	)
}

func (b *NodeBuilder) BridgeBalance(t *testing.T, account string, amount *big.Int) (*types.Transaction, *types.Receipt) {
	return BridgeBalance(t, account, amount, b.L1Info, b.L2Info, b.L1.Client, b.L2.Client, b.ctx)
}

func SendWaitTestTransactions(t *testing.T, ctx context.Context, client client, txs []*types.Transaction) {
	t.Helper()
	for _, tx := range txs {
		Require(t, client.SendTransaction(ctx, tx))
	}
	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}
}

func TransferBalance(
	t *testing.T, from, to string, amount *big.Int, l2info info, client client, ctx context.Context,
) (*types.Transaction, *types.Receipt) {
	t.Helper()
	return TransferBalanceTo(t, from, l2info.GetAddress(to), amount, l2info, client, ctx)
}

func TransferBalanceTo(
	t *testing.T, from string, to common.Address, amount *big.Int, l2info info, client client, ctx context.Context,
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
	t *testing.T, account string, amount *big.Int, l1info info, l2info info, l1client client, l2client client, ctx context.Context,
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

func SendSignedTxesInBatchViaL1(
	t *testing.T,
	ctx context.Context,
	l1info *BlockchainTestInfo,
	l1client arbutil.L1Interface,
	l2client arbutil.L1Interface,
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

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
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
	l1client arbutil.L1Interface,
	l2client arbutil.L1Interface,
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

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
	receipt, err := EnsureTxSucceeded(ctx, l2client, delayedTx)
	Require(t, err)
	return receipt
}

func SendUnsignedTxViaL1(
	t *testing.T,
	ctx context.Context,
	l1info *BlockchainTestInfo,
	l1client arbutil.L1Interface,
	l2client arbutil.L1Interface,
	templateTx *types.Transaction,
) *types.Receipt {
	delayedInboxContract, err := bridgegen.NewInbox(l1info.GetAddress("Inbox"), l1client)
	Require(t, err)

	usertxopts := l1info.GetDefaultTransactOpts("User", ctx)
	remapped := util.RemapL1Address(usertxopts.From)
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

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
			l1info.PrepareTx("Faucet", "Faucet", 30000, big.NewInt(1e12), nil),
		})
	}
	receipt, err := EnsureTxSucceeded(ctx, l2client, unsignedTx)
	Require(t, err)
	return receipt
}

func GetBaseFee(t *testing.T, client client, ctx context.Context) *big.Int {
	header, err := client.HeaderByNumber(ctx, nil)
	Require(t, err)
	return header.BaseFee
}

func GetBaseFeeAt(t *testing.T, client client, ctx context.Context, blockNum *big.Int) *big.Int {
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

func createTestValidationNode(t *testing.T, ctx context.Context, config *valnode.Config) (*valnode.ValidationNode, *node.Node) {
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
	valnode, err := valnode.CreateValidationNode(configFetcher, stack, nil)
	Require(t, err)

	err = stack.Start()
	Require(t, err)

	err = valnode.Start(ctx)
	Require(t, err)

	go func() {
		<-ctx.Done()
		stack.Close()
	}()

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

func createTestL1BlockChain(t *testing.T, l1info info) (info, *ethclient.Client, *eth.Ethereum, *node.Node) {
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	stackConfig := testhelpers.CreateStackConfigForTest(t.TempDir())
	l1info.GenerateAccount("Faucet")

	chainConfig := params.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams = params.ArbitrumChainParams{}

	stack, err := node.New(stackConfig)
	Require(t, err)

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = chainConfig.ChainID.Uint64()
	faucetAddr := l1info.GetAddress("Faucet")
	l1Genesis := core.DeveloperGenesisBlock(15_000_000, &faucetAddr)
	infoGenesis := l1info.GetGenesisAlloc()
	for acct, info := range infoGenesis {
		l1Genesis.Alloc[acct] = info
	}
	l1Genesis.BaseFee = big.NewInt(50 * params.GWei)
	nodeConf.Genesis = l1Genesis
	nodeConf.Miner.Etherbase = l1info.GetAddress("Faucet")
	nodeConf.SyncMode = downloader.FullSync

	l1backend, err := eth.New(stack, &nodeConf)
	Require(t, err)

	simBeacon, err := catalyst.NewSimulatedBeacon(0, l1backend)
	Require(t, err)
	catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
	stack.RegisterLifecycle(simBeacon)

	tempKeyStore := keystore.NewPlaintextKeyStore(t.TempDir())
	faucetAccount, err := tempKeyStore.ImportECDSA(l1info.Accounts["Faucet"].PrivateKey, "passphrase")
	Require(t, err)
	Require(t, tempKeyStore.Unlock(faucetAccount, "passphrase"))
	l1backend.AccountManager().AddBackend(tempKeyStore)
	l1backend.SetEtherbase(l1info.GetAddress("Faucet"))

	stack.RegisterLifecycle(&lifecycle{stop: func() error {
		l1backend.StopMining()
		return nil
	}})

	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filters.NewFilterSystem(l1backend.APIBackend, filters.Config{}), false),
	}})
	stack.RegisterAPIs(tracers.APIs(l1backend.APIBackend))

	Require(t, stack.Start())
	Require(t, l1backend.StartMining())

	rpcClient := stack.Attach()

	l1Client := ethclient.NewClient(rpcClient)

	return l1info, l1Client, l1backend, stack
}

func getInitMessage(ctx context.Context, t *testing.T, parentChainClient client, addresses *chaininfo.RollupAddresses) *arbostypes.ParsedInitMessage {
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

func deployOnParentChain(
	t *testing.T,
	ctx context.Context,
	parentChainInfo info,
	parentChainClient client,
	parentChainReaderConfig *headerreader.Config,
	chainConfig *params.ChainConfig,
	wasmModuleRoot common.Hash,
	prodConfirmPeriodBlocks bool,
	chainSupportsBlobs bool,
) (*chaininfo.RollupAddresses, *arbostypes.ParsedInitMessage) {
	parentChainInfo.GenerateAccount("RollupOwner")
	parentChainInfo.GenerateAccount("Sequencer")
	parentChainInfo.GenerateAccount("Validator")
	parentChainInfo.GenerateAccount("User")

	SendWaitTestTransactions(t, ctx, parentChainClient, []*types.Transaction{
		parentChainInfo.PrepareTx("Faucet", "RollupOwner", parentChainInfo.TransferGas, big.NewInt(9223372036854775807), nil),
		parentChainInfo.PrepareTx("Faucet", "Sequencer", parentChainInfo.TransferGas, big.NewInt(9223372036854775807), nil),
		parentChainInfo.PrepareTx("Faucet", "Validator", parentChainInfo.TransferGas, big.NewInt(9223372036854775807), nil),
		parentChainInfo.PrepareTx("Faucet", "User", parentChainInfo.TransferGas, big.NewInt(9223372036854775807), nil)})

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
	addresses, err := deploy.DeployOnParentChain(
		ctx,
		parentChainReader,
		&parentChainTransactionOpts,
		[]common.Address{parentChainInfo.GetAddress("Sequencer")},
		parentChainInfo.GetAddress("RollupOwner"),
		0,
		arbnode.GenerateRollupConfig(prodConfirmPeriodBlocks, wasmModuleRoot, parentChainInfo.GetAddress("RollupOwner"), chainConfig, serializedChainConfig, common.Address{}),
		nativeToken,
		maxDataSize,
		chainSupportsBlobs,
	)
	Require(t, err)
	parentChainInfo.SetContract("Bridge", addresses.Bridge)
	parentChainInfo.SetContract("SequencerInbox", addresses.SequencerInbox)
	parentChainInfo.SetContract("Inbox", addresses.Inbox)
	parentChainInfo.SetContract("UpgradeExecutor", addresses.UpgradeExecutor)
	initMessage := getInitMessage(ctx, t, parentChainClient, addresses)
	return addresses, initMessage
}

func createL2BlockChain(
	t *testing.T, l2info *BlockchainTestInfo, dataDir string, chainConfig *params.ChainConfig, execConfig *gethexec.Config,
) (*BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	return createNonL1BlockChainWithStackConfig(t, l2info, dataDir, chainConfig, nil, nil, execConfig)
}

func createNonL1BlockChainWithStackConfig(
	t *testing.T, info *BlockchainTestInfo, dataDir string, chainConfig *params.ChainConfig, initMessage *arbostypes.ParsedInitMessage, stackConfig *node.Config, execConfig *gethexec.Config,
) (*BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	if info == nil {
		info = NewArbTestInfo(t, chainConfig.ChainID)
	}
	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(dataDir)
	}
	if execConfig == nil {
		execConfig = ExecConfigDefaultTest(t)
	}

	stack, err := node.New(stackConfig)
	Require(t, err)

	chainData, err := stack.OpenDatabaseWithExtraOptions("l2chaindata", 0, 0, "l2chaindata/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata"))
	Require(t, err)
	wasmData, err := stack.OpenDatabaseWithExtraOptions("wasm", 0, 0, "wasm/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("wasm"))
	Require(t, err)
	chainDb := rawdb.WrapDatabaseWithWasm(chainData, wasmData, 0, execConfig.StylusTarget.WasmTargets())
	arbDb, err := stack.OpenDatabaseWithExtraOptions("arbitrumdata", 0, 0, "arbitrumdata/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("arbitrumdata"))
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
	coreCacheConfig := gethexec.DefaultCacheConfigFor(stack, &execConfig.Caching)
	blockchain, err := gethexec.WriteOrTestBlockChain(chainDb, coreCacheConfig, initReader, chainConfig, initMessage, ExecConfigDefaultTest(t).TxLookupLimit, 0)
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
) (*ethclient.Client, *arbnode.Node) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if execConfig == nil {
		execConfig = ExecConfigDefaultNonSequencerTest(t)
	}
	feedErrChan := make(chan error, 10)
	parentChainRpcClient := parentChainStack.Attach()
	parentChainClient := ethclient.NewClient(parentChainRpcClient)

	if stackConfig == nil {
		stackConfig = testhelpers.CreateStackConfigForTest(t.TempDir())
	}
	chainStack, err := node.New(stackConfig)
	Require(t, err)

	chainData, err := chainStack.OpenDatabaseWithExtraOptions("l2chaindata", 0, 0, "l2chaindata/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata"))
	Require(t, err)
	wasmData, err := chainStack.OpenDatabaseWithExtraOptions("wasm", 0, 0, "wasm/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("wasm"))
	Require(t, err)
	chainDb := rawdb.WrapDatabaseWithWasm(chainData, wasmData, 0, execConfig.StylusTarget.WasmTargets())

	arbDb, err := chainStack.OpenDatabaseWithExtraOptions("arbitrumdata", 0, 0, "arbitrumdata/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("arbitrumdata"))
	Require(t, err)
	initReader := statetransfer.NewMemoryInitDataReader(chainInitData)

	dataSigner := signature.DataSignerFromPrivateKey(parentChainInfo.GetInfoWithPrivKey("Sequencer").PrivateKey)
	sequencerTxOpts := parentChainInfo.GetDefaultTransactOpts("Sequencer", ctx)
	validatorTxOpts := parentChainInfo.GetDefaultTransactOpts("Validator", ctx)
	firstExec := getExecNode(t, first)

	chainConfig := firstExec.ArbInterface.BlockChain().Config()

	coreCacheConfig := gethexec.DefaultCacheConfigFor(chainStack, &execConfig.Caching)
	blockchain, err := gethexec.WriteOrTestBlockChain(chainDb, coreCacheConfig, initReader, chainConfig, initMessage, ExecConfigDefaultTest(t).TxLookupLimit, 0)
	Require(t, err)

	AddValNodeIfNeeded(t, ctx, nodeConfig, true, "", valnodeConfig.Wasm.RootPath)

	Require(t, execConfig.Validate())
	Require(t, nodeConfig.Validate())
	configFetcher := func() *gethexec.Config { return execConfig }
	currentExec, err := gethexec.CreateExecutionNode(ctx, chainStack, chainDb, blockchain, parentChainClient, configFetcher)
	Require(t, err)

	currentNode, err := arbnode.CreateNode(ctx, chainStack, currentExec, arbDb, NewFetcherFromConfig(nodeConfig), blockchain.Config(), parentChainClient, addresses, &validatorTxOpts, &sequencerTxOpts, dataSigner, feedErrChan, big.NewInt(1337), nil)
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
	l1client arbutil.L1Interface,
) {
	if dasSignerKey == nil {
		return
	}
	keyset := &daprovider.DataAvailabilityKeyset{
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
	chainConfig := params.ArbitrumDevTestChainConfig()
	var dbPath string
	var err error

	enableFileStorage, enableDbStorage, enableDas := false, false, true
	switch dasModeString {
	case "db":
		enableDbStorage = true
		chainConfig = params.ArbitrumDevTestDASChainConfig()
	case "files":
		enableFileStorage = true
		chainConfig = params.ArbitrumDevTestDASChainConfig()
	case "onchain":
		enableDas = false
	default:
		Fatal(t, "unknown storage type")
	}
	dbPath = t.TempDir()
	dasSignerKey, _, err := das.GenerateAndStoreKeys(dbPath)
	Require(t, err)

	dbConfig := das.DefaultLocalDBStorageConfig
	dbConfig.Enable = enableDbStorage
	dbConfig.DataDir = dbPath

	dasConfig := &das.DataAvailabilityConfig{
		Enable: enableDas,
		Key: das.KeyConfig{
			KeyDir: dbPath,
		},
		LocalFileStorage: das.LocalFileStorageConfig{
			Enable:  enableFileStorage,
			DataDir: dbPath,
		},
		LocalDBStorage:           dbConfig,
		RequestTimeout:           5 * time.Second,
		ParentChainNodeURL:       "none",
		SequencerInboxAddress:    "none",
		PanicOnError:             true,
		DisableSignatureChecking: true,
	}

	l1NodeConfigA.DataAvailability = das.DefaultDataAvailabilityConfig
	var lifecycleManager *das.LifecycleManager
	var daReader das.DataAvailabilityServiceReader
	var daWriter das.DataAvailabilityServiceWriter
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

func deploySimple(
	t *testing.T, ctx context.Context, auth bind.TransactOpts, client *ethclient.Client,
) (common.Address, *mocksgen.Simple) {
	addr, tx, simple, err := mocksgen.DeploySimple(&auth, client)
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

func TestMain(m *testing.M) {
	logLevelEnv := os.Getenv("TEST_LOGLEVEL")
	if logLevelEnv != "" {
		logLevel, err := strconv.ParseInt(logLevelEnv, 10, 32)
		if err != nil || logLevel > int64(log.LevelCrit) {
			log.Warn("TEST_LOGLEVEL exists but out of bound, ignoring", "logLevel", logLevelEnv, "max", log.LvlTrace)
		}
		glogger := log.NewGlogHandler(
			log.NewTerminalHandler(io.Writer(os.Stderr), false))
		glogger.Verbosity(slog.Level(logLevel))
		log.SetDefault(log.NewLogger(glogger))
	}
	code := m.Run()
	os.Exit(code)
}

func getExecNode(t *testing.T, node *arbnode.Node) *gethexec.ExecutionNode {
	t.Helper()
	gethExec, ok := node.Execution.(*gethexec.ExecutionNode)
	if !ok {
		t.Fatal("failed to get exec node from arbnode")
	}
	return gethExec
}

func logParser[T any](t *testing.T, source string, name string) func(*types.Log) *T {
	parser := util.NewLogParser[T](source, name)
	return func(log *types.Log) *T {
		t.Helper()
		event, err := parser(log)
		Require(t, err, "failed to parse log")
		return event
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

// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/execclient"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/valnode"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbutil"
	_ "github.com/offchainlabs/nitro/execution/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

type info = *BlockchainTestInfo
type client = arbutil.L1Interface

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
			Nonce:      0,
		})
	} else {
		l2acct := l2info.GetInfoWithPrivKey(account)
		if l2acct.PrivateKey.X.Cmp(l1acct.PrivateKey.X) != 0 ||
			l2acct.PrivateKey.Y.Cmp(l1acct.PrivateKey.Y) != 0 {
			Fail(t, "l2 account already exists and not compatible to l1")
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
			if i > 20 {
				Fail(t, "bridging failed")
			}
			<-time.After(time.Millisecond * 100)
		}
	}

	return tx, res
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

func createTestL1BlockChain(t *testing.T, l1info info) (info, *ethclient.Client, *eth.Ethereum, *node.Node) {
	return createTestL1BlockChainWithConfig(t, l1info, nil)
}

func getTestStackConfig(t *testing.T) *node.Config {
	stackConfig := node.DefaultConfig
	stackConfig.HTTPPort = 0
	stackConfig.WSPort = 0
	stackConfig.WSHost = "127.0.0.1"
	stackConfig.WSModules = []string{server_api.Namespace, consensus.RPCNamespace, execution.RPCNamespace}
	stackConfig.UseLightweightKDF = true
	stackConfig.P2P.ListenAddr = ""
	stackConfig.P2P.NoDial = true
	stackConfig.P2P.NoDiscovery = true
	stackConfig.P2P.NAT = nil
	stackConfig.DataDir = t.TempDir()
	return &stackConfig
}

func createDefaultStackForTest(dataDir string) (*node.Node, error) {
	stackConf := node.DefaultConfig
	var err error
	stackConf.DataDir = dataDir
	stackConf.HTTPHost = ""
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stackConf.WSPort = 0
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSModules = []string{server_api.Namespace, consensus.RPCNamespace, execution.RPCNamespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	if err != nil {
		return nil, fmt.Errorf("error creating protocol stack: %w", err)
	}
	return stack, nil
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

	stack, err := node.New(&stackConf)
	Require(t, err)

	node.DefaultAuthModules = []string{server_api.Namespace}

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

func StaticFetcherFrom[T any](config T) func() T {
	return func() T { return config }
}

func configByValidationNode(t *testing.T, clientConfig *arbnode.Config, valStack *node.Node) {
	clientConfig.BlockValidator.ValidationServer.URL = valStack.WSEndpoint()
	clientConfig.BlockValidator.ValidationServer.JWTSecret = ""
}

func AddDefaultValNode(t *testing.T, ctx context.Context, nodeConfig *arbnode.Config, useJit bool) {
	if !nodeConfig.ValidatorRequired() {
		return
	}
	if nodeConfig.BlockValidator.ValidationServer.URL != "" {
		return
	}
	conf := valnode.TestValidationConfig
	conf.UseJit = useJit
	_, valStack := createTestValidationNode(t, ctx, &conf)
	configByValidationNode(t, nodeConfig, valStack)
}

func createTestL1BlockChainWithConfig(t *testing.T, l1info info, stackConfig *node.Config) (info, *ethclient.Client, *eth.Ethereum, *node.Node) {
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	if stackConfig == nil {
		stackConfig = getTestStackConfig(t)
	}
	l1info.GenerateAccount("Faucet")

	chainConfig := params.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams = params.ArbitrumChainParams{}

	stack, err := node.New(stackConfig)
	Require(t, err)

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = chainConfig.ChainID.Uint64()
	l1Genesis := core.DeveloperGenesisBlock(0, 15_000_000, l1info.GetAddress("Faucet"))
	infoGenesis := l1info.GetGenesisAlloc()
	for acct, info := range infoGenesis {
		l1Genesis.Alloc[acct] = info
	}
	l1Genesis.BaseFee = big.NewInt(50 * params.GWei)
	nodeConf.Genesis = l1Genesis
	nodeConf.Miner.Etherbase = l1info.GetAddress("Faucet")

	l1backend, err := eth.New(stack, &nodeConf)
	Require(t, err)
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

	Require(t, stack.Start())
	Require(t, l1backend.StartMining(1))

	rpcClient, err := stack.Attach()
	Require(t, err)

	l1Client := ethclient.NewClient(rpcClient)

	return l1info, l1Client, l1backend, stack
}

func DeployOnTestL1(
	t *testing.T, ctx context.Context, l1info info, l1client client, chainId *big.Int,
) *arbnode.RollupAddresses {
	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil)})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	config := arbnode.GenerateRollupConfig(false, locator.LatestWasmModuleRoot(), l1info.GetAddress("RollupOwner"), chainId, common.Address{})
	addresses, err := arbnode.DeployOnL1(
		ctx,
		l1client,
		&l1TransactionOpts,
		l1info.GetAddress("Sequencer"),
		0,
		func() *headerreader.Config { return &headerreader.TestConfig },
		config,
	)
	Require(t, err)
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)
	return addresses
}

func createL2BlockChain(
	t *testing.T, l2info *BlockchainTestInfo, dataDir string, chainConfig *params.ChainConfig,
) (*BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	return createL2BlockChainWithStackConfig(t, l2info, dataDir, chainConfig, nil)
}

func createL2BlockChainWithStackConfig(
	t *testing.T, l2info *BlockchainTestInfo, dataDir string, chainConfig *params.ChainConfig, stackConfig *node.Config,
) (*BlockchainTestInfo, *node.Node, ethdb.Database, ethdb.Database, *core.BlockChain) {
	if l2info == nil {
		l2info = NewArbTestInfo(t, chainConfig.ChainID)
	}
	var stack *node.Node
	var err error
	if stackConfig == nil {
		stack, err = createDefaultStackForTest(dataDir)
		Require(t, err)
	} else {
		stack, err = node.New(stackConfig)
		Require(t, err)
	}

	chainDb, err := stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	arbDb, err := stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	initReader := statetransfer.NewMemoryInitDataReader(&l2info.ArbInitData)
	blockchain, err := gethexec.WriteOrTestBlockChain(chainDb, nil, initReader, chainConfig, gethexec.ConfigDefaultTest().TxLookupLimit, 0)
	Require(t, err)

	return l2info, stack, chainDb, arbDb, blockchain
}

func ClientForStack(t *testing.T, backend *node.Node) *ethclient.Client {
	rpcClient, err := backend.Attach()
	Require(t, err)
	return ethclient.NewClient(rpcClient)
}

// Create and deploy L1 and arbnode for L2
func createTestNodeOnL1(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
) (
	l2info info, node *arbnode.Node, l2client *ethclient.Client, l1info info,
	l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
) {
	return createTestNodeOnL1WithConfig(t, ctx, isSequencer, nil, nil, nil, nil)
}

func createTestNodeOnL1WithConfig(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	execConfig *gethexec.Config,
	chainConfig *params.ChainConfig,
	stackConfig *node.Config,
) (
	l2info info, currentNode *arbnode.Node, l2client *ethclient.Client, l1info info,
	l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
) {
	l2info, currentNode, l2client, _, l1info, l1backend, l1client, l1stack = createTestNodeOnL1WithConfigImpl(t, ctx, isSequencer, nodeConfig, execConfig, chainConfig, stackConfig, nil)
	return
}

func createTestNodeOnL1WithConfigImpl(
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
	addresses := DeployOnTestL1(t, ctx, l1info, l1client, chainConfig.ChainID)
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
	execNode, err := gethexec.CreateExecutionNode(l2stack, l2chainDb, l2blockchain, l1client, execConfigFetcher)
	Require(t, err)

	execclient := execclient.NewClient(StaticFetcherFrom[*rpcclient.ClientConfig](&rpcclient.TestClientConfig), l2stack)
	currentNode, err = arbnode.CreateNode(
		ctx, l2stack, execclient, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client,
		addresses, sequencerTxOptsPtr, dataSigner, fatalErrChan,
	)
	Require(t, err)

	Require(t, execNode.Initialize(ctx))

	Require(t, currentNode.Start(ctx))

	Require(t, execNode.Start(ctx))

	l2client = ClientForStack(t, l2stack)

	StartWatchChanErr(t, ctx, fatalErrChan, currentNode, execNode)

	return
}

type execNodesInfo struct {
	execNodes []*gethexec.ExecutionNode
	lock      sync.Mutex
}

var execNodes map[string]*execNodesInfo = make(map[string]*execNodesInfo)
var execNodeLock sync.Mutex

// assumptions: it is possible that multiple tests will use the same endpoint, but not concurrently
// if multiple tests registered for the same endpoint, only the last one is active
func addExecNode(endpoint string, node *gethexec.ExecutionNode) {
	execNodeLock.Lock()
	defer execNodeLock.Unlock()
	nodeInfo := execNodes[endpoint]
	if nodeInfo == nil {
		nodeInfo = &execNodesInfo{}
		execNodes[endpoint] = nodeInfo
	}
	nodeInfo.lock.Lock()
	defer nodeInfo.lock.Unlock()
	nodeInfo.execNodes = append(nodeInfo.execNodes, node)
}

func rmExecNode(t *testing.T, endpoint string, node *gethexec.ExecutionNode) {
	execNodeLock.Lock()
	defer execNodeLock.Unlock()
	nodeInfo := execNodes[endpoint]
	if nodeInfo == nil {
		return
	}
	nodeInfo.lock.Lock()
	defer nodeInfo.lock.Unlock()
	if len(nodeInfo.execNodes) == 1 && nodeInfo.execNodes[0] == node {
		delete(execNodes, endpoint)
		return
	}
	newNodes := []*gethexec.ExecutionNode{}
	for _, storedNode := range nodeInfo.execNodes {
		if storedNode != node {
			newNodes = append(newNodes, storedNode)
		}
	}
	nodeInfo.execNodes = newNodes
}

func getExecNodeFromEndpoint(t *testing.T, endpoint string) *gethexec.ExecutionNode {
	execNodeLock.Lock()
	defer execNodeLock.Unlock()
	nodeInfo := execNodes[endpoint]
	if nodeInfo == nil {
		Fail(t, "didn't find execnode for endpoint: ", endpoint)
		return nil
	}
	nodeInfo.lock.Lock()
	defer nodeInfo.lock.Unlock()
	return nodeInfo.execNodes[len(nodeInfo.execNodes)-1]
}

// L2 -Only. Enough for tests that needs no interface to L1
// Requires precompiles.AllowDebugPrecompiles = true
func CreateTestL2(t *testing.T, ctx context.Context) (*BlockchainTestInfo, *arbnode.Node, *ethclient.Client) {
	return CreateTestL2WithConfig(t, ctx, nil, nil, nil, true)
}

func CreateTestL2WithConfig(
	t *testing.T, ctx context.Context, l2Info *BlockchainTestInfo, nodeConfig *arbnode.Config, execConfig *gethexec.Config, takeOwnership bool,
) (*BlockchainTestInfo, *arbnode.Node, *ethclient.Client) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL2Test()
	}
	if execConfig == nil {
		execConfig = gethexec.ConfigDefaultTest()
	}

	feedErrChan := make(chan error, 10)

	AddDefaultValNode(t, ctx, nodeConfig, true)

	l2info, stack, chainDb, arbDb, blockchain := createL2BlockChain(t, l2Info, "", params.ArbitrumDevTestChainConfig())

	Require(t, execConfig.Validate())
	execConfigFetcher := func() *gethexec.Config { return execConfig }
	execNode, err := gethexec.CreateExecutionNode(stack, chainDb, blockchain, nil, execConfigFetcher)
	Require(t, err)

	execclient := execclient.NewClient(StaticFetcherFrom[*rpcclient.ClientConfig](&rpcclient.TestClientConfig), stack)

	currentNode, err := arbnode.CreateNode(ctx, stack, execclient, arbDb, NewFetcherFromConfig(nodeConfig), blockchain.Config(), nil, nil, nil, nil, feedErrChan)
	Require(t, err)

	// Give the node an init message
	err = currentNode.TxStreamer.AddFakeInitMessage()
	Require(t, err)

	Require(t, execNode.Initialize(ctx))

	Require(t, currentNode.Start(ctx))
	client := ClientForStack(t, stack)

	Require(t, execNode.Start(ctx))

	if takeOwnership {
		debugAuth := l2info.GetDefaultTransactOpts("Owner", ctx)

		// make auth a chain owner
		arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), client)
		Require(t, err, "failed to deploy ArbDebug")

		tx, err := arbdebug.BecomeChainOwner(&debugAuth)
		Require(t, err, "failed to deploy ArbDebug")

		_, err = EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}

	StartWatchChanErr(t, ctx, feedErrChan, currentNode, execNode)

	return l2info, currentNode, client
}

func StartWatchChanErr(t *testing.T, ctx context.Context, feedErrChan chan error, node *arbnode.Node, exec *gethexec.ExecutionNode) {
	endpoint := node.Stack.WSEndpoint()
	addExecNode(endpoint, exec)
	go func() {
		select {
		case err := <-feedErrChan:
			t.Errorf("error occurred: %v", err)
			if node != nil {
				node.StopAndWait()
			}
			if exec != nil {
				exec.StopAndWait()
			}
		case <-ctx.Done():
		}
		rmExecNode(t, endpoint, exec)
	}()
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

func Create2ndNode(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	dasConfig *das.DataAvailabilityConfig,
) (*ethclient.Client, *arbnode.Node) {
	nodeConf := arbnode.ConfigDefaultL1NonSequencerTest()
	if dasConfig == nil {
		nodeConf.DataAvailability.Enable = false
	} else {
		nodeConf.DataAvailability = *dasConfig
	}
	return Create2ndNodeWithConfig(t, ctx, first, l1stack, l1info, l2InitData, nodeConf, nil, nil)
}

func Create2ndNodeWithConfig(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l1info *BlockchainTestInfo,
	l2InitData *statetransfer.ArbosInitializationInfo,
	nodeConfig *arbnode.Config,
	execConfig *gethexec.Config,
	stackConfig *node.Config,
) (*ethclient.Client, *arbnode.Node) {
	if nodeConfig == nil {
		nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if execConfig == nil {
		execConfig = gethexec.ConfigDefaultNonSequencerTest()
	}
	feedErrChan := make(chan error, 10)
	l1rpcClient, err := l1stack.Attach()
	if err != nil {
		Fail(t, err)
	}
	l1client := ethclient.NewClient(l1rpcClient)

	if stackConfig == nil {
		stackConfig = getTestStackConfig(t)
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

	l2blockchain, err := gethexec.WriteOrTestBlockChain(l2chainDb, nil, initReader, firstExec.ArbInterface.BlockChain().Config(), gethexec.ConfigDefaultTest().TxLookupLimit, 0)
	Require(t, err)

	AddDefaultValNode(t, ctx, nodeConfig, true)

	configFetcher := func() *gethexec.Config { return execConfig }
	currentExec, err := gethexec.CreateExecutionNode(l2stack, l2chainDb, l2blockchain, l1client, configFetcher)
	Require(t, err)

	execclient := execclient.NewClient(StaticFetcherFrom[*rpcclient.ClientConfig](&rpcclient.TestClientConfig), l2stack)

	currentNode, err := arbnode.CreateNode(ctx, l2stack, execclient, l2arbDb, NewFetcherFromConfig(nodeConfig), l2blockchain.Config(), l1client, first.DeployInfo, &txOpts, dataSigner, feedErrChan)
	Require(t, err)

	Require(t, currentExec.Initialize(ctx))

	err = currentNode.Start(ctx)
	Require(t, err, nodeConfig.BlockValidator.ValidationServer.URL, " l2: ", l2stack.WSEndpoint())
	l2client := ClientForStack(t, l2stack)

	Require(t, currentExec.Start(ctx))

	StartWatchChanErr(t, ctx, feedErrChan, currentNode, currentExec)

	return l2client, currentNode
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
	keyset := &arbstate.DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{*dasSignerKey},
	}
	wr := bytes.NewBuffer([]byte{})
	err := keyset.Serialize(wr)
	Require(t, err, "unable to serialize DAS keyset")
	keysetBytes := wr.Bytes()
	sequencerInbox, err := bridgegen.NewSequencerInbox(l1info.Accounts["SequencerInbox"].Address, l1client)
	Require(t, err, "unable to create sequencer inbox")
	trOps := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	tx, err := sequencerInbox.SetValidKeyset(&trOps, keysetBytes)
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
		Fail(t, "unknown storage type")
	}
	dbPath = t.TempDir()
	dasSignerKey, _, err := das.GenerateAndStoreKeys(dbPath)
	Require(t, err)

	dasConfig := &das.DataAvailabilityConfig{
		Enable: enableDas,
		KeyConfig: das.KeyConfig{
			KeyDir: dbPath,
		},
		LocalFileStorageConfig: das.LocalFileStorageConfig{
			Enable:  enableFileStorage,
			DataDir: dbPath,
		},
		LocalDBStorageConfig: das.LocalDBStorageConfig{
			Enable:  enableDbStorage,
			DataDir: dbPath,
		},
		RequestTimeout:           5 * time.Second,
		L1NodeURL:                "none",
		SequencerInboxAddress:    "none",
		PanicOnError:             true,
		DisableSignatureChecking: true,
	}

	l1NodeConfigA.DataAvailability = das.DefaultDataAvailabilityConfig
	var lifecycleManager *das.LifecycleManager
	var daReader das.DataAvailabilityServiceReader
	var daWriter das.DataAvailabilityServiceWriter
	var daHealthChecker das.DataAvailabilityServiceHealthChecker
	if dasModeString != "onchain" {
		daReader, daWriter, daHealthChecker, lifecycleManager, err = das.CreateDAComponentsForDaserver(ctx, dasConfig, nil, nil)

		Require(t, err)
		rpcLis, err := net.Listen("tcp", "localhost:0")
		Require(t, err)
		restLis, err := net.Listen("tcp", "localhost:0")
		Require(t, err)
		_, err = das.StartDASRPCServerOnListener(ctx, rpcLis, genericconf.HTTPServerTimeoutConfigDefault, daReader, daWriter, daHealthChecker)
		Require(t, err)
		_, err = das.NewRestfulDasServerOnListener(restLis, genericconf.HTTPServerTimeoutConfigDefault, daReader, daHealthChecker)
		Require(t, err)

		beConfigA := das.BackendConfig{
			URL:                 "http://" + rpcLis.Addr().String(),
			PubKeyBase64Encoded: blsPubToBase64(dasSignerKey),
			SignerMask:          1,
		}
		l1NodeConfigA.DataAvailability.AggregatorConfig = aggConfigForBackend(t, beConfigA)
		l1NodeConfigA.DataAvailability.Enable = true
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig = das.DefaultRestfulClientAggregatorConfig
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Enable = true
		l1NodeConfigA.DataAvailability.RestfulClientAggregatorConfig.Urls = []string{"http://" + restLis.Addr().String()}
		l1NodeConfigA.DataAvailability.L1NodeURL = "none"
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

func getExecNode(t *testing.T, node *arbnode.Node) *gethexec.ExecutionNode {
	t.Helper()
	gethExec, ok := node.Execution.(*gethexec.ExecutionNode)
	if ok {
		return gethExec
	}
	endpoint := node.Stack.WSEndpoint() // this assumes both use the same address
	return getExecNodeFromEndpoint(t, endpoint)
}

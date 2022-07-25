// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/das"

	"github.com/offchainlabs/nitro/blsSignatures"

	"github.com/offchainlabs/nitro/arbstate"

	"github.com/offchainlabs/nitro/util/headerreader"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbutil"
	_ "github.com/offchainlabs/nitro/nodeInterface"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator"
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
	tx := l2info.PrepareTx(from, to, l2info.TransferGas, amount, nil)
	err := client.SendTransaction(ctx, tx)
	Require(t, err)
	res, err := EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
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
	receipt, err := WaitForTx(ctx, l2client, delayedTx.Hash(), time.Second*5)
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

func CreateTestL1BlockChain(t *testing.T, l1info info) (info, *ethclient.Client, *eth.Ethereum, *node.Node) {
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	l1info.GenerateAccount("Faucet")

	chainConfig := params.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams = params.ArbitrumChainParams{}

	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.WSPort = 0
	stackConf.UseLightweightKDF = true
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.NAT = nil
	var err error
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	Require(t, err)

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = chainConfig.ChainID.Uint64()
	l1Genesys := core.DeveloperGenesisBlock(0, 15_000_000, l1info.GetAddress("Faucet"))
	infoGenesys := l1info.GetGenesysAlloc()
	for acct, info := range infoGenesys {
		l1Genesys.Alloc[acct] = info
	}
	l1Genesys.BaseFee = big.NewInt(50 * params.GWei)
	nodeConf.Genesis = l1Genesys
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
	addresses, err := arbnode.DeployOnL1(
		ctx,
		l1client,
		&l1TransactionOpts,
		l1info.GetAddress("Sequencer"),
		l1info.GetAddress("RollupOwner"),
		0,
		common.Hash{},
		chainId,
		headerreader.TestConfig,
		validator.DefaultNitroMachineConfig,
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
	if l2info == nil {
		l2info = NewArbTestInfo(t, chainConfig.ChainID)
	}
	stack, err := arbnode.CreateDefaultStackForTest(dataDir)
	Require(t, err)
	chainDb, err := stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	arbDb, err := stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)

	initReader := statetransfer.NewMemoryInitDataReader(&l2info.ArbInitData)
	blockchain, err := arbnode.WriteOrTestBlockChain(chainDb, nil, initReader, chainConfig, 0)
	Require(t, err)

	return l2info, stack, chainDb, arbDb, blockchain
}

func ClientForStack(t *testing.T, backend *node.Node) *ethclient.Client {
	rpcClient, err := backend.Attach()
	Require(t, err)
	return ethclient.NewClient(rpcClient)
}

// Create and deploy L1 and arbnode for L2
func CreateTestNodeOnL1(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
) (
	l2info info, node *arbnode.Node, l2client *ethclient.Client, l2stack *node.Node, l1info info,
	l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
) {
	conf := arbnode.ConfigDefaultL1Test()
	return CreateTestNodeOnL1WithConfig(t, ctx, isSequencer, conf, params.ArbitrumDevTestChainConfig())
}

func CreateTestNodeOnL1WithConfig(
	t *testing.T,
	ctx context.Context,
	isSequencer bool,
	nodeConfig *arbnode.Config,
	chainConfig *params.ChainConfig,
) (
	l2info info, node *arbnode.Node, l2client *ethclient.Client, l2stack *node.Node, l1info info,
	l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node,
) {
	l1info, l1client, l1backend, l1stack = CreateTestL1BlockChain(t, nil)
	var l2chainDb ethdb.Database
	var l2arbDb ethdb.Database
	var l2blockchain *core.BlockChain
	l2info, l2stack, l2chainDb, l2arbDb, l2blockchain = createL2BlockChain(t, nil, "", chainConfig)
	addresses := DeployOnTestL1(t, ctx, l1info, l1client, chainConfig.ChainID)
	var sequencerTxOptsPtr *bind.TransactOpts
	if isSequencer {
		sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer", ctx)
		sequencerTxOptsPtr = &sequencerTxOpts
	}

	if !isSequencer {
		nodeConfig.BatchPoster.Enable = false
		nodeConfig.Sequencer.Enable = false
		nodeConfig.DelayedSequencer.Enable = false
	}
	node, err := arbnode.CreateNode(ctx, l2stack, l2chainDb, l2arbDb, nodeConfig, l2blockchain, l1client, addresses, sequencerTxOptsPtr, nil)

	Require(t, err)
	Require(t, l2stack.Start())

	l2client = ClientForStack(t, l2stack)
	return
}

// L2 -Only. Enough for tests that needs no interface to L1
// Requires precompiles.AllowDebugPrecompiles = true
func CreateTestL2(t *testing.T, ctx context.Context) (*BlockchainTestInfo, *arbnode.Node, *ethclient.Client, *node.Node) {
	return CreateTestL2WithConfig(t, ctx, nil, arbnode.ConfigDefaultL2Test(), true)
}

func CreateTestL2WithConfig(
	t *testing.T, ctx context.Context, l2Info *BlockchainTestInfo, nodeConfig *arbnode.Config, takeOwnership bool,
) (*BlockchainTestInfo, *arbnode.Node, *ethclient.Client, *node.Node) {
	l2info, stack, chainDb, arbDb, blockchain := createL2BlockChain(t, l2Info, "", params.ArbitrumDevTestChainConfig())
	node, err := arbnode.CreateNode(ctx, stack, chainDb, arbDb, nodeConfig, blockchain, nil, nil, nil, nil)
	Require(t, err)

	// Give the node an init message
	err = node.TxStreamer.AddFakeInitMessage()
	Require(t, err)

	Require(t, stack.Start())
	client := ClientForStack(t, stack)

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

	return l2info, node, client, stack
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
	l2InitData *statetransfer.ArbosInitializationInfo,
	dasConfig *das.DataAvailabilityConfig,
) (*ethclient.Client, *arbnode.Node, *node.Node) {
	nodeConf := arbnode.ConfigDefaultL1NonSequencerTest()
	if dasConfig == nil {
		nodeConf.DataAvailability.Enable = false
	} else {
		nodeConf.DataAvailability = *dasConfig
	}
	return Create2ndNodeWithConfig(t, ctx, first, l1stack, l2InitData, nodeConf)
}

func Create2ndNodeWithConfig(
	t *testing.T,
	ctx context.Context,
	first *arbnode.Node,
	l1stack *node.Node,
	l2InitData *statetransfer.ArbosInitializationInfo,
	nodeConfig *arbnode.Config,
) (*ethclient.Client, *arbnode.Node, *node.Node) {
	l1rpcClient, err := l1stack.Attach()
	if err != nil {
		Fail(t, err)
	}
	l1client := ethclient.NewClient(l1rpcClient)
	l2stack, err := arbnode.CreateDefaultStackForTest("")
	Require(t, err)

	l2chainDb, err := l2stack.OpenDatabase("chaindb", 0, 0, "", false)
	Require(t, err)
	l2arbDb, err := l2stack.OpenDatabase("arbdb", 0, 0, "", false)
	Require(t, err)
	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)

	l2blockchain, err := arbnode.WriteOrTestBlockChain(l2chainDb, nil, initReader, first.ArbInterface.BlockChain().Config(), 0)
	Require(t, err)

	node, err := arbnode.CreateNode(ctx, l2stack, l2chainDb, l2arbDb, nodeConfig, l2blockchain, l1client, first.DeployInfo, nil, nil)
	Require(t, err)

	err = l2stack.Start()
	Require(t, err)
	l2client := ClientForStack(t, l2stack)
	return l2client, node, l2stack
}

func GetBalance(t *testing.T, ctx context.Context, client *ethclient.Client, account common.Address) *big.Int {
	t.Helper()
	balance, err := client.BalanceAt(ctx, account, nil)
	Require(t, err, "could not get balance")
	return balance
}

func requireClose(t *testing.T, s *node.Node) {
	Require(t, s.Close())
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
	Require(t, err)
	keysetBytes := wr.Bytes()
	sequencerInbox, err := bridgegen.NewSequencerInbox(l1info.Accounts["SequencerInbox"].Address, l1client)
	Require(t, err)
	trOps := l1info.GetDefaultTransactOpts("RollupOwner", ctx)
	tx, err := sequencerInbox.SetValidKeyset(&trOps, keysetBytes)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1client, tx)
	Require(t, err)
}

func setupConfigWithDAS(
	t *testing.T, dasModeString string,
) (*params.ChainConfig, *arbnode.Config, string, *blsSignatures.PublicKey) {
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

	dasConfig := das.DataAvailabilityConfig{
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
		PanicOnError:             true,
		DisableSignatureChecking: true,
	}

	l1NodeConfigA.DataAvailability = dasConfig
	return chainConfig, l1NodeConfigA, dbPath, dasSignerKey
}

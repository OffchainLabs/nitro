//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/arbstate/arbos/l2pricing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/solgen/go/precompilesgen"
	"github.com/offchainlabs/arbstate/util/testhelpers"
)

type info = *BlockchainTestInfo
type client = arbnode.L1Interface

var (
	l1Genesys, l2Genesys *core.Genesis
)

func SendWaitTestTransactions(t *testing.T, ctx context.Context, client client, txs []*types.Transaction) {
	t.Helper()
	for _, tx := range txs {
		Require(t, client.SendTransaction(ctx, tx))
	}
	for _, tx := range txs {
		_, err := arbnode.EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}
}

func TransferBalance(t *testing.T, from, to string, amount *big.Int, l2info info, ctx context.Context) {
	tx := l2info.PrepareTx(from, to, 100000, amount, nil)
	err := l2info.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = arbnode.EnsureTxSucceeded(ctx, l2info.Client, tx)
	Require(t, err)
}

func GetBaseFee(t *testing.T, client client, ctx context.Context) *big.Int {
	header, err := client.HeaderByNumber(ctx, nil)
	Require(t, err)
	return header.BaseFee
}

func CreateTestL1BlockChain(t *testing.T, l1info info) (info, *eth.Ethereum, *node.Node) {
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	l1info.GenerateAccount("Faucet")

	chainConfig := params.ArbitrumTestChainConfig()

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
	l1Genesys = core.DeveloperGenesisBlock(0, l2pricing.PerBlockGasLimit, l1info.GetAddress("Faucet"))
	infoGenesys := l1info.GetGenesysAlloc()
	for acct, info := range infoGenesys {
		l1Genesys.Alloc[acct] = info
	}
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
	Require(t, stack.Start())
	Require(t, l1backend.StartMining(1))

	rpcClient, err := stack.Attach()
	Require(t, err)

	l1Client := ethclient.NewClient(rpcClient)

	l1info.Client = l1Client

	return l1info, l1backend, stack
}

func DeployOnTestL1(t *testing.T, ctx context.Context, l1info info) *arbnode.RollupAddresses {
	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	SendWaitTestTransactions(t, ctx, l1info.Client, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil)})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner")
	addresses, err := arbnode.DeployOnL1(ctx, l1info.Client, &l1TransactionOpts, l1info.GetAddress("Sequencer"), time.Second)
	Require(t, err)
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)
	return addresses
}

func createL2BlockChain(t *testing.T) (info, *node.Node, ethdb.Database, *core.BlockChain) {
	l2info := NewArbTestInfo(t)
	l2info.GenerateAccount("Owner")
	l2info.GenerateAccount("Faucet")
	l2GenesysAlloc := make(map[common.Address]core.GenesisAccount)
	l2GenesysAlloc[l2info.GetAddress("Owner")] = core.GenesisAccount{
		Balance:    big.NewInt(9223372036854775807),
		Nonce:      0,
		PrivateKey: nil,
	}
	l2GenesysAlloc[l2info.GetAddress("Faucet")] = core.GenesisAccount{
		Balance:    new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9)),
		Nonce:      0,
		PrivateKey: nil,
	}
	l2Genesys = &core.Genesis{
		Config:     params.ArbitrumTestChainConfig(),
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      l2GenesysAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(l2pricing.InitialGasPriceWei),
	}
	stack, err := arbnode.CreateDefaultStack()
	Require(t, err)
	chainDb, blockchain, err := arbnode.CreateDefaultBlockChain(stack, l2Genesys)
	Require(t, err)
	return l2info, stack, chainDb, blockchain
}

func ClientForArbBackend(t *testing.T, backend *arbitrum.Backend) *ethclient.Client {
	apis := backend.APIBackend().GetAPIs()

	inproc := rpc.NewServer()
	for _, api := range apis {
		err := inproc.RegisterName(api.Namespace, api.Service)
		Require(t, err)
	}

	return ethclient.NewClient(rpc.DialInProc(inproc))
}

// Create and deploy L1 and arbnode for L2
func CreateTestNodeOnL1(t *testing.T, ctx context.Context, isSequencer bool) (info, *arbnode.Node, info, *eth.Ethereum, *node.Node) {
	conf := arbnode.NodeConfigL1Test
	return CreateTestNodeOnL1WithConfig(t, ctx, isSequencer, &conf)
}

func CreateTestNodeOnL1WithConfig(t *testing.T, ctx context.Context, isSequencer bool, nodeConfig *arbnode.NodeConfig) (info, *arbnode.Node, info, *eth.Ethereum, *node.Node) {
	l1info, l1backend, l1stack := CreateTestL1BlockChain(t, nil)
	l2info, l2stack, l2chainDb, l2blockchain := createL2BlockChain(t)
	addresses := DeployOnTestL1(t, ctx, l1info)
	var sequencerTxOptsPtr *bind.TransactOpts
	if isSequencer {
		sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer")
		sequencerTxOptsPtr = &sequencerTxOpts
	}

	if !isSequencer {
		nodeConfig.BatchPoster = false
	}
	node, err := arbnode.CreateNode(l2stack, l2chainDb, nodeConfig, l2blockchain, l1info.Client, addresses, sequencerTxOptsPtr)

	Require(t, err)
	Require(t, node.Start(ctx))

	l2info.Client = ClientForArbBackend(t, node.Backend)
	return l2info, node, l1info, l1backend, l1stack
}

// L2 -Only. Enough for tests that needs no interface to L1
// Requires precompiles.AllowDebugPrecompiles = true
func CreateTestL2(t *testing.T, ctx context.Context) (info, *arbnode.Node, *ethclient.Client) {
	return CreateTestL2WithConfig(t, ctx, &arbnode.NodeConfigL2Test)
}

func CreateTestL2WithConfig(t *testing.T, ctx context.Context, nodeConfig *arbnode.NodeConfig) (info, *arbnode.Node, *ethclient.Client) {
	l2info, stack, chainDb, blockchain := createL2BlockChain(t)
	node, err := arbnode.CreateNode(stack, chainDb, nodeConfig, blockchain, nil, nil, nil)
	Require(t, err)
	Require(t, node.Start(ctx))
	l2info.Client = ClientForArbBackend(t, node.Backend)

	client := l2info.Client
	debugAuth := l2info.GetDefaultTransactOpts("Owner")

	// make auth a chain owner
	arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), client)
	Require(t, err, "failed to deploy ArbDebug")

	tx, err := arbdebug.BecomeChainOwner(&debugAuth)
	Require(t, err, "failed to deploy ArbDebug")

	_, err = arbnode.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)

	return l2info, node, client
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

func Create2ndNode(t *testing.T, ctx context.Context, first *arbnode.Node, l1stack *node.Node, blockValidator bool) (*ethclient.Client, *arbnode.Node) {
	nodeConf := arbnode.NodeConfigL1Test
	nodeConf.BatchPoster = false
	nodeConf.BlockValidator = blockValidator
	return Create2ndNodeWithConfig(t, ctx, first, l1stack, &nodeConf)
}

func Create2ndNodeWithConfig(t *testing.T, ctx context.Context, first *arbnode.Node, l1stack *node.Node, nodeConfig *arbnode.NodeConfig) (*ethclient.Client, *arbnode.Node) {
	l1rpcClient, err := l1stack.Attach()
	if err != nil {
		t.Fatal(err)
	}
	l1client := ethclient.NewClient(l1rpcClient)
	l2stack, err := arbnode.CreateDefaultStack()
	Require(t, err)

	l2chainDb, l2blockchain, err := arbnode.CreateDefaultBlockChain(l2stack, l2Genesys)
	Require(t, err)

	node, err := arbnode.CreateNode(l2stack, l2chainDb, nodeConfig, l2blockchain, l1client, first.DeployInfo, nil)
	Require(t, err)

	err = node.Start(ctx)
	Require(t, err)
	l2client := ClientForArbBackend(t, node.Backend)
	return l2client, node
}

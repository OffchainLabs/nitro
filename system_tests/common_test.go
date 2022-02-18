//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/statetransfer"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
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
type client = arbutil.L1Interface

func SendWaitTestTransactions(t *testing.T, ctx context.Context, client client, txs []*types.Transaction) {
	t.Helper()
	for _, tx := range txs {
		Require(t, client.SendTransaction(ctx, tx))
	}
	for _, tx := range txs {
		_, err := arbutil.EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}
}

func TransferBalance(t *testing.T, from, to string, amount *big.Int, l2info info, client client, ctx context.Context) {
	tx := l2info.PrepareTx(from, to, l2info.TransferGas, amount, nil)
	err := client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = arbutil.EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
}

func GetBaseFee(t *testing.T, client client, ctx context.Context) *big.Int {
	header, err := client.HeaderByNumber(ctx, nil)
	Require(t, err)
	return header.BaseFee
}

func CreateTestL1BlockChain(t *testing.T, l1info info) (info, *ethclient.Client, *eth.Ethereum, *node.Node) {
	if l1info == nil {
		l1info = NewL1TestInfo(t)
	}
	l1info.GenerateAccount("Faucet")

	chainConfig := params.ArbitrumTestChainConfig()
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
	Require(t, stack.Start())
	Require(t, l1backend.StartMining(1))

	rpcClient, err := stack.Attach()
	Require(t, err)

	l1Client := ethclient.NewClient(rpcClient)

	return l1info, l1Client, l1backend, stack
}

func DeployOnTestL1(t *testing.T, ctx context.Context, l1info info, l1client client) *arbnode.RollupAddresses {
	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	SendWaitTestTransactions(t, ctx, l1client, []*types.Transaction{
		l1info.PrepareTx("Faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("Faucet", "User", 30000, big.NewInt(9223372036854775807), nil)})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner")
	addresses, err := arbnode.DeployOnL1(ctx, l1client, &l1TransactionOpts, l1info.GetAddress("Sequencer"), common.Hash{}, 5*time.Second)
	Require(t, err)
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)
	return addresses
}

func createL2BlockChain(t *testing.T, l2info *BlockchainTestInfo) (*BlockchainTestInfo, *node.Node, ethdb.Database, *core.BlockChain) {
	if l2info == nil {
		l2info = NewArbTestInfo(t)
	}
	stack, err := arbnode.CreateDefaultStack()
	Require(t, err)
	chainDb := rawdb.NewMemoryDatabase()

	initReader := statetransfer.NewMemoryInitDataReader(&l2info.ArbInitData)
	blockchain, err := arbnode.WriteOrTestBlockChain(chainDb, nil, initReader, 0, params.ArbitrumTestChainConfig())
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
func CreateTestNodeOnL1(t *testing.T, ctx context.Context, isSequencer bool) (l2info info, node *arbnode.Node, l2client *ethclient.Client, l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node) {
	conf := arbnode.NodeConfigL1Test
	return CreateTestNodeOnL1WithConfig(t, ctx, isSequencer, &conf)
}

func CreateTestNodeOnL1WithConfig(t *testing.T, ctx context.Context, isSequencer bool, nodeConfig *arbnode.NodeConfig) (l2info info, node *arbnode.Node, l2client *ethclient.Client, l1info info, l1backend *eth.Ethereum, l1client *ethclient.Client, l1stack *node.Node) {
	l1info, l1client, l1backend, l1stack = CreateTestL1BlockChain(t, nil)
	l2info, l2stack, l2chainDb, l2blockchain := createL2BlockChain(t, nil)
	addresses := DeployOnTestL1(t, ctx, l1info, l1client)
	var sequencerTxOptsPtr *bind.TransactOpts
	if isSequencer {
		sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer")
		sequencerTxOptsPtr = &sequencerTxOpts
	}

	if !isSequencer {
		nodeConfig.BatchPoster = false
	}
	node, err := arbnode.CreateNode(l2stack, l2chainDb, nodeConfig, l2blockchain, l1client, addresses, sequencerTxOptsPtr)

	Require(t, err)
	Require(t, node.Start(ctx))

	l2client = ClientForArbBackend(t, node.Backend)
	return
}

// L2 -Only. Enough for tests that needs no interface to L1
// Requires precompiles.AllowDebugPrecompiles = true
func CreateTestL2(t *testing.T, ctx context.Context) (*BlockchainTestInfo, *arbnode.Node, *ethclient.Client) {
	return CreateTestL2WithConfig(t, ctx, nil, &arbnode.NodeConfigL2Test, true)
}

func CreateTestL2WithConfig(t *testing.T, ctx context.Context, l2Info *BlockchainTestInfo, nodeConfig *arbnode.NodeConfig, takeOwnership bool) (*BlockchainTestInfo, *arbnode.Node, *ethclient.Client) {
	l2info, stack, chainDb, blockchain := createL2BlockChain(t, l2Info)
	node, err := arbnode.CreateNode(stack, chainDb, nodeConfig, blockchain, nil, nil, nil)
	Require(t, err)

	// Give the node an init message
	err = node.TxStreamer.AddMessages(0, false, []arbstate.MessageWithMetadata{{
		Message: &arbos.L1IncomingMessage{
			Header: &arbos.L1IncomingMessageHeader{
				Kind: arbos.L1MessageType_SetChainParams,
			},
			L2msg: math.U256Bytes(l2info.Signer.ChainID()),
		},
		DelayedMessagesRead: 0,
	}})
	Require(t, err)

	Require(t, node.Start(ctx))
	client := ClientForArbBackend(t, node.Backend)

	if takeOwnership {
		debugAuth := l2info.GetDefaultTransactOpts("Owner")

		// make auth a chain owner
		arbdebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), client)
		Require(t, err, "failed to deploy ArbDebug")

		tx, err := arbdebug.BecomeChainOwner(&debugAuth)
		Require(t, err, "failed to deploy ArbDebug")

		_, err = arbutil.EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
	}

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

func Create2ndNode(t *testing.T, ctx context.Context, first *arbnode.Node, l1stack *node.Node, l2InitData *statetransfer.ArbosInitializationInfo, blockValidator bool) (*ethclient.Client, *arbnode.Node) {
	nodeConf := arbnode.NodeConfigL1Test
	nodeConf.BatchPoster = false
	nodeConf.BlockValidator = blockValidator
	return Create2ndNodeWithConfig(t, ctx, first, l1stack, l2InitData, &nodeConf)
}

func Create2ndNodeWithConfig(t *testing.T, ctx context.Context, first *arbnode.Node, l1stack *node.Node, l2InitData *statetransfer.ArbosInitializationInfo, nodeConfig *arbnode.NodeConfig) (*ethclient.Client, *arbnode.Node) {
	l1rpcClient, err := l1stack.Attach()
	if err != nil {
		t.Fatal(err)
	}
	l1client := ethclient.NewClient(l1rpcClient)
	l2stack, err := arbnode.CreateDefaultStack()
	Require(t, err)
	l2chainDb := rawdb.NewMemoryDatabase()
	initReader := statetransfer.NewMemoryInitDataReader(l2InitData)

	l2blockchain, err := arbnode.WriteOrTestBlockChain(l2chainDb, nil, initReader, 0, params.ArbitrumTestChainConfig())
	Require(t, err)

	node, err := arbnode.CreateNode(l2stack, l2chainDb, nodeConfig, l2blockchain, l1client, first.DeployInfo, nil)
	Require(t, err)

	err = node.Start(ctx)
	Require(t, err)
	l2client := ClientForArbBackend(t, node.Backend)
	return l2client, node
}

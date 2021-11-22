//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"os"
	"testing"

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
	"github.com/offchainlabs/arbstate/arbos"
)

var simulatedChainID = big.NewInt(1337)

var (
	l1Genesys, l2Genesys *core.Genesis
)

func SendWaitTestTransactions(t *testing.T, ctx context.Context, client arbnode.L1Interface, txs []*types.Transaction) {
	t.Helper()
	for _, tx := range txs {
		err := client.SendTransaction(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(txs) > 0 {
		_, err := arbnode.EnsureTxSucceeded(ctx, client, txs[len(txs)-1])
		if err != nil {
			t.Fatal(err)
		}
	}
}

func CreateTestL1BlockChain(t *testing.T) (*BlockchainTestInfo, *eth.Ethereum, *node.Node) {
	l1info := NewBlockChainTestInfo(t, types.NewLondonSigner(simulatedChainID), 0)
	l1info.GenerateAccount("faucet")

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
	if err != nil {
		t.Fatal(err)
	}

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()
	l1Genesys = core.DeveloperGenesisBlock(0, l1info.GetAddress("faucet"))
	nodeConf.Genesis = l1Genesys
	nodeConf.Miner.Etherbase = l1info.GetAddress("faucet")

	l1backend, err := eth.New(stack, &nodeConf)
	if err != nil {
		t.Fatal(err)
	}
	tempKeyStore := keystore.NewPlaintextKeyStore(t.TempDir())
	faucetAccount, err := tempKeyStore.ImportECDSA(l1info.Accounts["faucet"].PrivateKey, "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	err = tempKeyStore.Unlock(faucetAccount, "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	l1backend.AccountManager().AddBackend(tempKeyStore)
	l1backend.SetEtherbase(l1info.GetAddress("faucet"))
	err = stack.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = l1backend.StartMining(1)
	if err != nil {
		t.Fatal(err)
	}

	rpcClient, err := stack.Attach()
	if err != nil {
		t.Fatal(err)
	}

	l1Client := ethclient.NewClient(rpcClient)

	l1info.Client = l1Client

	return l1info, l1backend, stack
}

func TestDeployOnL1(t *testing.T, ctx context.Context, l1info *BlockchainTestInfo) *arbnode.RollupAddresses {
	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	SendWaitTestTransactions(t, ctx, l1info.Client, []*types.Transaction{
		l1info.PrepareTx("faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("faucet", "User", 30000, big.NewInt(9223372036854775807), nil)})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner")
	addresses, err := arbnode.DeployOnL1(ctx, l1info.Client, &l1TransactionOpts, l1info.GetAddress("Sequencer"))
	if err != nil {
		t.Fatal(err)
	}
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)
	return addresses
}

func createL2BlockChain(t *testing.T) (*BlockchainTestInfo, *node.Node, ethdb.Database, *core.BlockChain) {
	l2info := NewBlockChainTestInfo(t, types.NewArbitrumSigner(types.NewLondonSigner(arbos.ChainConfig.ChainID)), 1e6)
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
		Config:     arbos.ChainConfig,
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
		BaseFee:    big.NewInt(params.InitialBaseFee / 100),
	}
	stack, err := arbnode.CreateDefaultStack()
	if err != nil {
		t.Fatal(err)
	}
	chainDb, blockchain, err := arbnode.CreateDefaultBlockChain(stack, l2Genesys)
	if err != nil {
		t.Fatal(err)
	}
	return l2info, stack, chainDb, blockchain
}

func ClientForArbBackend(t *testing.T, backend *arbitrum.Backend) *ethclient.Client {
	apis := backend.APIBackend().GetAPIs()

	inproc := rpc.NewServer()
	for _, api := range apis {
		if err := inproc.RegisterName(api.Namespace, api.Service); err != nil {
			t.Fatal(err)
		}
	}

	return ethclient.NewClient(rpc.DialInProc(inproc))
}

// Create and deploy L1 and arbnode for L2
func CreateTestNodeOnL1(t *testing.T, ctx context.Context, isSequencer bool) (*BlockchainTestInfo, *arbnode.Node, *BlockchainTestInfo, *eth.Ethereum, *node.Node) {
	l1info, l1backend, l1stack := CreateTestL1BlockChain(t)
	l2info, l2stack, l2chainDb, l2blockchain := createL2BlockChain(t)
	addresses := TestDeployOnL1(t, ctx, l1info)
	var sequencerTxOptsPtr *bind.TransactOpts
	if isSequencer {
		sequencerTxOpts := l1info.GetDefaultTransactOpts("Sequencer")
		sequencerTxOptsPtr = &sequencerTxOpts
	}
	nodeconf := arbnode.NodeConfigL1Test
	nodeconf.RecordPath = os.ExpandEnv("${ARBRECORDPATH}")

	node, err := arbnode.CreateNode(l2stack, l2chainDb, &nodeconf, l2blockchain, l1info.Client, addresses, sequencerTxOptsPtr)
	if err != nil {
		t.Fatal(err)
	}
	err = node.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	l2info.Client = ClientForArbBackend(t, node.Backend)
	return l2info, node, l1info, l1backend, l1stack
}

// L2 -Only. Enough for tests that needs no interface to L1
func CreateTestL2(t *testing.T, ctx context.Context) (*BlockchainTestInfo, *arbnode.Node) {
	l2info, stack, chainDb, blockchain := createL2BlockChain(t)
	node, err := arbnode.CreateNode(stack, chainDb, &arbnode.NodeConfigL2Test, blockchain, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = node.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	l2info.Client = ClientForArbBackend(t, node.Backend)
	return l2info, node
}

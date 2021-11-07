//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
)

var simulatedChainID = big.NewInt(1337)

func SendWaitTestTransactions(t *testing.T, client arbnode.L1Interface, txs []*types.Transaction) {
	t.Helper()
	ctx := context.Background()
	for _, tx := range txs {
		err := client.SendTransaction(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tx := range txs {
		_, err := arbnode.EnsureTxSucceeded(client, tx)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func CreateTestL1(t *testing.T, l2backend *arbitrum.Backend) (arbnode.L1Interface, *core.BlockChain, *BlockchainTestInfo) {
	l1info := NewBlockChainTestInfo(t, types.NewLondonSigner(simulatedChainID), 0)
	l1info.GenerateAccount("faucet")

	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.WSPort = 0
	stackConf.P2P.ListenAddr = ":0"
	var err error
	stackConf.DataDir = t.TempDir()
	stack, err := node.New(&stackConf)
	if err != nil {
		t.Fatal(err)
	}

	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()
	nodeConf.Genesis = core.DeveloperGenesisBlock(0, l1info.GetAddress("faucet"))
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

	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	SendWaitTestTransactions(t, l1Client, []*types.Transaction{
		l1info.PrepareTx("faucet", "RollupOwner", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("faucet", "Sequencer", 30000, big.NewInt(9223372036854775807), nil),
		l1info.PrepareTx("faucet", "User", 30000, big.NewInt(9223372036854775807), nil)})

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner")
	sequencerTxOpt := l1info.GetDefaultTransactOpts("Sequencer")
	addresses, err := arbnode.CreateL1WithInbox(l1Client, l2backend, &l1TransactionOpts, l1info.GetAddress("Sequencer"), &sequencerTxOpt, true)
	if err != nil {
		t.Fatal(err)
	}
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	return l1Client, l1backend.BlockChain(), l1info
}

func CreateTestL2(t *testing.T) (*arbitrum.Backend, *BlockchainTestInfo) {
	l2info := NewBlockChainTestInfo(t, types.NewArbitrumSigner(types.NewLondonSigner(arbos.ChainConfig.ChainID)), 1e6)
	l2info.GenerateAccount("Owner")
	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[l2info.GetAddress("Owner")] = core.GenesisAccount{
		Balance:    big.NewInt(9223372036854775807),
		Nonce:      0,
		PrivateKey: nil,
	}
	stack, err := arbnode.CreateStack()
	if err != nil {
		t.Fatal(err)
	}
	backend, err := arbnode.CreateArbBackend(stack, genesisAlloc)
	if err != nil {
		t.Fatal(err)
	}

	return backend, l2info
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

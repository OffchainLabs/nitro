//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/solgen/go/bridgegen"
)

type AccountInfo struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

type BlockchainTestInfo struct {
	T        *testing.T
	Signer   types.Signer
	Accounts map[string]AccountInfo
}

func NewBlocChainTestInfo(t *testing.T, signer types.Signer) *BlockchainTestInfo {
	return &BlockchainTestInfo{
		T:        t,
		Signer:   signer,
		Accounts: make(map[string]AccountInfo),
	}
}

func (b *BlockchainTestInfo) GenerateAccount(name string) {
	b.T.Helper()

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		b.T.Fatal(err)
	}
	b.Accounts[name] = AccountInfo{
		PrivateKey: privateKey,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
	}
}

func (b *BlockchainTestInfo) SetContract(name string, address common.Address) {
	b.Accounts[name] = AccountInfo{
		PrivateKey: nil,
		Address:    address,
	}
}

func (b *BlockchainTestInfo) GetAddress(name string) common.Address {
	b.T.Helper()
	info, ok := b.Accounts[name]
	if !ok {
		b.T.Fatal("not found account: ", name)
	}
	return info.Address
}

func (b *BlockchainTestInfo) GetDefaultTransactOpts(name string) bind.TransactOpts {
	b.T.Helper()
	info, ok := b.Accounts[name]
	if !ok {
		b.T.Fatal("not found account: ", name)
	}
	if info.PrivateKey == nil {
		b.T.Fatal("no private key for account: ", name)
	}
	return bind.TransactOpts{
		From:      info.Address,
		Nonce:     nil,
		GasLimit:  30000,
		GasFeeCap: big.NewInt(5e+09),
		GasTipCap: big.NewInt(2),
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != info.Address {
				return nil, errors.New("bad address")
			}
			signature, err := crypto.Sign(b.Signer.Hash(tx).Bytes(), info.PrivateKey)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(b.Signer, signature)
		},
	}
}

func (b *BlockchainTestInfo) SignTxAs(name string, data types.TxData) *types.Transaction {
	b.T.Helper()
	info, ok := b.Accounts[name]
	if !ok {
		b.T.Fatal("not found account: ", name)
	}
	if info.PrivateKey == nil {
		b.T.Fatal("no private key for account: ", name)
	}
	tx := types.NewTx(data)
	tx, err := types.SignTx(tx, b.Signer, info.PrivateKey)
	if err != nil {
		b.T.Fatal(err)
	}
	return tx
}

func CreateL1WithInbox(t *testing.T) (*backends.SimulatedBackend, *BlockchainTestInfo) {
	var gasLimit uint64 = 8000029
	l1info := NewBlocChainTestInfo(t, types.NewLondonSigner(big.NewInt(1337)))
	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")

	l1genAlloc := make(core.GenesisAlloc)
	l1genAlloc[l1info.GetAddress("RollupOwner")] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}
	l1genAlloc[l1info.GetAddress("Sequencer")] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}

	l1sim := backends.NewSimulatedBackend(l1genAlloc, gasLimit)

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner")
	bridgeAddr, _, bridgeContract, err := bridgegen.DeployBridge(&l1TransactionOpts, l1sim)
	if err != nil {
		t.Fatal(err)
	}
	l1info.SetContract("Bridge", bridgeAddr)

	inboxAddr, _, inboxContract, err := bridgegen.DeployInbox(&l1TransactionOpts, l1sim)
	if err != nil {
		t.Fatal(err)
	}
	l1info.SetContract("Inbox", inboxAddr)

	_, err = inboxContract.Initialize(&l1TransactionOpts, bridgeAddr)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bridgeContract.SetInbox(&l1TransactionOpts, inboxAddr, true)
	if err != nil {
		t.Fatal(err)
	}

	sequencerInboxAddr, _, _, err := bridgegen.DeploySequencerInbox(&l1TransactionOpts, l1sim, bridgeAddr, l1info.GetAddress("Sequencer"))
	if err != nil {
		t.Fatal(err)
	}

	l1info.SetContract("SequencerInbox", sequencerInboxAddr)

	return l1sim, l1info
}

func CreateTestBackendWithBalance(t *testing.T) (*arbitrum.Backend, *BlockchainTestInfo, *backends.SimulatedBackend, *BlockchainTestInfo) {
	arbstate.RequireHookedGeth()
	stackConf := node.DefaultConfig
	var err error
	stackConf.DataDir = t.TempDir()
	stackConf.HTTPHost = "localhost"
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stack, err := node.New(&stackConf)
	if err != nil {
		utils.Fatalf("Error creating protocol stack: %v\n", err)
	}
	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()

	l1backend, l1info := CreateL1WithInbox(t)

	l2info := NewBlocChainTestInfo(t, types.NewArbitrumSigner(types.NewLondonSigner(arbos.ChainConfig.ChainID)))
	l2info.GenerateAccount("Owner")

	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[l2info.GetAddress("Owner")] = core.GenesisAccount{
		Balance:    big.NewInt(params.Ether * 2),
		Nonce:      0,
		PrivateKey: nil,
	}
	nodeConf.Genesis = &core.Genesis{
		Config:     arbos.ChainConfig,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", nodeConf.DatabaseCache, nodeConf.DatabaseHandles, nodeConf.DatabaseFreezer, "eth/db/chaindata/", false)
	if err != nil {
		t.Fatal(err)
	}

	delayedBridge, err := arbnode.NewDelayedBridge(l1backend, l1info.GetAddress("Bridge"), 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = arbnode.NewInboxReader(chainDb, l1backend, &big.Int{}, delayedBridge)
	if err != nil {
		t.Fatal(err)
	}

	engine := arbos.Engine{
		IsSequencer: true,
	}
	chainConfig, _, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, nodeConf.Genesis, nodeConf.OverrideLondon)
	var configCompatError *params.ConfigCompatError
	if errors.As(genesisErr, &configCompatError) {
		t.Fatal(genesisErr)
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: nodeConf.EnablePreimageRecording,
	}
	cacheConfig := &core.CacheConfig{
		TrieCleanLimit:      nodeConf.TrieCleanCache,
		TrieCleanJournal:    stack.ResolvePath(nodeConf.TrieCleanCacheJournal),
		TrieCleanRejournal:  nodeConf.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch: nodeConf.NoPrefetch,
		TrieDirtyLimit:      nodeConf.TrieDirtyCache,
		TrieDirtyDisabled:   nodeConf.NoPruning,
		TrieTimeLimit:       nodeConf.TrieTimeout,
		SnapshotLimit:       nodeConf.SnapshotCache,
		Preimages:           nodeConf.Preimages,
	}

	blockChain, err := core.NewBlockChain(chainDb, cacheConfig, chainConfig, engine, vmConfig, shouldPreserveFalse, &nodeConf.TxLookupLimit)
	if err != nil {
		t.Fatal(err)
	}

	inbox, err := arbnode.NewInboxState(chainDb, blockChain)
	if err != nil {
		t.Fatal(err)
	}

	inbox.Start(context.Background())

	sequencer := arbnode.NewSequencer(inbox)

	backend, err := arbitrum.NewBackend(stack, &nodeConf, chainDb, blockChain, arbos.ChainConfig.ChainID, sequencer)
	if err != nil {
		t.Fatal(err)
	}

	return backend, l2info, l1backend, l1info
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

// will wait untill tx is in the blockchain. attempts = 0 is infinite
func WaitForTx(t *testing.T, txhash common.Hash, backend *arbitrum.Backend, client *ethclient.Client, attempts int) {
	ctx := context.Background()
	chanHead := make(chan *types.Header, 20)
	headSubscribe, err := client.SubscribeNewHead(ctx, chanHead)
	if err != nil {
		t.Fatal(err)
	}
	defer headSubscribe.Unsubscribe()

	for {
		reciept, _ := client.TransactionReceipt(ctx, txhash)
		if reciept != nil {
			fmt.Println("Reciept: ", reciept)
			break
		}
		if attempts == 1 {
			t.Fatal("timeout waiting for Tx ", txhash)
		}
		if attempts > 1 {
			attempts -= 1
		}
		select {
		case <-chanHead:
		case <-time.After(time.Second / 100):
		}
	}
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}

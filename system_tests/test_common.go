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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
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

func NewBlockChainTestInfo(t *testing.T, signer types.Signer) *BlockchainTestInfo {
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
		From:     info.Address,
		GasLimit: 4000000,
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

func CreateTestL1(t *testing.T) (*backends.SimulatedBackend, *BlockchainTestInfo) {
	var gasLimit uint64 = 8000029
	l1info := NewBlockChainTestInfo(t, types.NewLondonSigner(big.NewInt(1337)))
	l1info.GenerateAccount("RollupOwner")
	l1info.GenerateAccount("Sequencer")
	l1info.GenerateAccount("User")

	l1genAlloc := make(core.GenesisAlloc)
	l1genAlloc[l1info.GetAddress("RollupOwner")] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}
	l1genAlloc[l1info.GetAddress("Sequencer")] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}
	l1genAlloc[l1info.GetAddress("User")] = core.GenesisAccount{Balance: big.NewInt(9223372036854775807)}

	l1TransactionOpts := l1info.GetDefaultTransactOpts("RollupOwner")

	chainDb := rawdb.NewMemoryDatabase()
	l1sim := backends.NewSimulatedBackendWithDatabase(chainDb, l1genAlloc, gasLimit)
	addresses, err := arbnode.CreateL1WithInbox(l1sim, &l1TransactionOpts, l1info.GetAddress("Sequencer"))
	if err != nil {
		t.Fatal(err)
	}
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	return l1sim, l1info
}

func CreateTestL2(t *testing.T) (*arbitrum.Backend, *BlockchainTestInfo) {
	l2info := NewBlockChainTestInfo(t, types.NewArbitrumSigner(types.NewLondonSigner(arbos.ChainConfig.ChainID)))
	l2info.GenerateAccount("Owner")
	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[l2info.GetAddress("Owner")] = core.GenesisAccount{
		Balance:    big.NewInt(params.Ether * 2),
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

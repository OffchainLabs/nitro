//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbnode"
	"github.com/offchainlabs/arbstate/arbos"
)

var simulatedChainID = big.NewInt(1337)

type AccountInfo struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
	Nonce      uint64
}

type BlockchainTestInfo struct {
	T        *testing.T
	Signer   types.Signer
	Accounts map[string]*AccountInfo
	keySeed  int64
}

func NewBlockChainTestInfo(t *testing.T, signer types.Signer, keySeed int64) *BlockchainTestInfo {
	return &BlockchainTestInfo{
		T:        t,
		Signer:   signer,
		Accounts: make(map[string]*AccountInfo),
		keySeed:  keySeed,
	}
}

func (b *BlockchainTestInfo) GenerateAccount(name string) {
	b.T.Helper()

	seedBytes := common.BigToHash(big.NewInt(b.keySeed)).Bytes()
	seedBytes = append(seedBytes, seedBytes...)
	seedReader := bytes.NewReader(seedBytes)
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), seedReader)
	if err != nil {
		b.T.Fatal(err)
	}
	b.keySeed += 1
	b.Accounts[name] = &AccountInfo{
		PrivateKey: privateKey,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		Nonce:      0,
	}
	log.Info("New Key ", "name", name, "Address", b.Accounts[name].Address)
}

func (b *BlockchainTestInfo) SetContract(name string, address common.Address) {
	b.Accounts[name] = &AccountInfo{
		PrivateKey: nil,
		Address:    address,
	}
}

func (b *BlockchainTestInfo) GetAddress(name string) common.Address {
	b.T.Helper()
	info := b.Accounts[name]
	if info == nil {
		b.T.Fatal("not found account: ", name)
	}
	return info.Address
}

func (b *BlockchainTestInfo) GetInfoWithPrivKey(name string) *AccountInfo {
	b.T.Helper()
	info := b.Accounts[name]
	if info == nil {
		b.T.Fatal("not found account: ", name)
	}
	if info.PrivateKey == nil {
		b.T.Fatal("no private key for account: ", name)
	}
	return info
}

func (b *BlockchainTestInfo) GetDefaultTransactOpts(name string) bind.TransactOpts {
	b.T.Helper()
	info := b.GetInfoWithPrivKey(name)
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
			info.Nonce += 1 // we don't set Nonce, but try to keep track..
			return tx.WithSignature(b.Signer, signature)
		},
	}
}

func (b *BlockchainTestInfo) SignTxAs(name string, data types.TxData) *types.Transaction {
	b.T.Helper()
	info := b.GetInfoWithPrivKey(name)
	tx := types.NewTx(data)
	tx, err := types.SignTx(tx, b.Signer, info.PrivateKey)
	if err != nil {
		b.T.Fatal(err)
	}
	return tx
}

func (b *BlockchainTestInfo) PrepareTx(from, to string, gas uint64, value *big.Int, data []byte) *types.Transaction {
	b.T.Helper()
	addr := b.GetAddress(to)
	info := b.GetInfoWithPrivKey(from)
	txData := &types.DynamicFeeTx{
		To:        &addr,
		Gas:       gas,
		GasFeeCap: big.NewInt(params.InitialBaseFee * 2),
		Value:     value,
		Nonce:     info.Nonce,
	}
	info.Nonce += 1
	return b.SignTxAs(from, txData)
}

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
		err := arbnode.EnsureTxSucceeded(client, tx)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func CreateTestL1(t *testing.T, l2backend *arbitrum.Backend) (arbnode.L1Interface, *BlockchainTestInfo) {
	l1info := NewBlockChainTestInfo(t, types.NewLondonSigner(simulatedChainID), 0)
	l1info.GenerateAccount("faucet")

	stackConf := node.DefaultConfig
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

	addresses, err := arbnode.CreateL1WithInbox(l1Client, &l1TransactionOpts, l1info.GetAddress("Sequencer"))
	if err != nil {
		t.Fatal(err)
	}
	l1info.SetContract("Bridge", addresses.Bridge)
	l1info.SetContract("SequencerInbox", addresses.SequencerInbox)
	l1info.SetContract("Inbox", addresses.Inbox)

	return l1Client, l1info
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

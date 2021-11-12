package arbtest

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

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
	Client   *ethclient.Client
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
		Data:      data,
	}
	info.Nonce += 1
	return b.SignTxAs(from, txData)
}

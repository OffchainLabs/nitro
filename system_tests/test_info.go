// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/statetransfer"
)

var simulatedChainID = big.NewInt(1337)

type AccountInfo struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
	Nonce      uint64
}

type BlockchainTestInfo struct {
	T           *testing.T
	Signer      types.Signer
	Accounts    map[string]*AccountInfo
	ArbInitData statetransfer.ArbosInitializationInfo
	GasPrice    *big.Int
	// The amount of gas needed for a simple transfer tx.
	TransferGas uint64
}

func NewBlockChainTestInfo(t *testing.T, signer types.Signer, gasPrice *big.Int, transferGas uint64) *BlockchainTestInfo {
	return &BlockchainTestInfo{
		T:           t,
		Signer:      signer,
		Accounts:    make(map[string]*AccountInfo),
		GasPrice:    new(big.Int).Set(gasPrice),
		TransferGas: transferGas,
	}
}

func NewArbTestInfo(t *testing.T, chainId *big.Int) *BlockchainTestInfo {
	var transferGas = util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for aggregator L1 costs
	arbinfo := NewBlockChainTestInfo(
		t,
		types.NewArbitrumSigner(types.NewLondonSigner(chainId)), big.NewInt(l2pricing.InitialBaseFeeWei*2),
		transferGas,
	)
	arbinfo.GenerateGenesisAccount("Owner", new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9)))
	arbinfo.GenerateGenesisAccount("Faucet", new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9)))
	return arbinfo
}

func NewL1TestInfo(t *testing.T) *BlockchainTestInfo {
	return NewBlockChainTestInfo(t, types.NewLondonSigner(simulatedChainID), big.NewInt(params.GWei*100), params.TxGas)
}

func GetTestKeyForAccountName(t *testing.T, name string) *ecdsa.PrivateKey {
	nameBytes := []byte(name)
	seedBytes := make([]byte, 0, 128)
	for len(seedBytes) < 64 {
		seedBytes = append(seedBytes, nameBytes...)
	}
	seedReader := bytes.NewReader(seedBytes)
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), seedReader)
	if err != nil {
		t.Fatal(err)
	}
	return privateKey
}

func GetTestAddressForAccountName(t *testing.T, name string) common.Address {
	privateKey := GetTestKeyForAccountName(t, name)
	return crypto.PubkeyToAddress(privateKey.PublicKey)
}

func (b *BlockchainTestInfo) GenerateAccount(name string) {
	b.T.Helper()

	privateKey := GetTestKeyForAccountName(b.T, name)
	if b.Accounts[name] != nil {
		b.T.Fatal("account already exists")
	}
	b.Accounts[name] = &AccountInfo{
		PrivateKey: privateKey,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		Nonce:      0,
	}
	log.Info("New Key ", "name", name, "Address", b.Accounts[name].Address)
}

func (b *BlockchainTestInfo) HasAccount(name string) bool {
	return b.Accounts[name] != nil
}

func (b *BlockchainTestInfo) GenerateGenesisAccount(name string, balance *big.Int) {
	b.GenerateAccount(name)
	b.ArbInitData.Accounts = append(b.ArbInitData.Accounts, statetransfer.AccountInitializationInfo{
		Addr:       b.Accounts[name].Address,
		EthBalance: new(big.Int).Set(balance),
	})
}

func (b *BlockchainTestInfo) GetGenesisAlloc() core.GenesisAlloc {
	alloc := make(core.GenesisAlloc)
	for _, info := range b.ArbInitData.Accounts {
		var contractCode []byte
		contractStorage := make(map[common.Hash]common.Hash)
		if info.ContractInfo != nil {
			contractCode = append([]byte{}, info.ContractInfo.Code...)
			for k, v := range info.ContractInfo.ContractStorage {
				contractStorage[k] = v
			}
		}
		alloc[info.Addr] = core.GenesisAccount{
			Balance: new(big.Int).Set(info.EthBalance),
			Nonce:   info.Nonce,
			Code:    contractCode,
			Storage: contractStorage,
		}
	}
	return alloc
}

func (b *BlockchainTestInfo) SetContract(name string, address common.Address) {
	b.Accounts[name] = &AccountInfo{
		PrivateKey: nil,
		Address:    address,
	}
}

func (b *BlockchainTestInfo) SetFullAccountInfo(name string, info *AccountInfo) {
	infoCopy := *info
	b.Accounts[name] = &infoCopy
}

func (b *BlockchainTestInfo) GetAddress(name string) common.Address {
	b.T.Helper()
	info, ok := b.Accounts[name]
	if !ok {
		b.T.Fatal("not found account: ", name)
	}
	return info.Address
}

func (b *BlockchainTestInfo) GetInfoWithPrivKey(name string) *AccountInfo {
	b.T.Helper()
	info, ok := b.Accounts[name]
	if !ok {
		b.T.Fatal("not found account: ", name)
	}
	if info.PrivateKey == nil {
		b.T.Fatal("no private key for account: ", name)
	}
	return info
}

func (b *BlockchainTestInfo) GetDefaultTransactOpts(name string, ctx context.Context) bind.TransactOpts {
	b.T.Helper()
	info := b.GetInfoWithPrivKey(name)
	return bind.TransactOpts{
		From: info.Address,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != info.Address {
				return nil, errors.New("bad address")
			}
			signature, err := crypto.Sign(b.Signer.Hash(tx).Bytes(), info.PrivateKey)
			if err != nil {
				return nil, err
			}
			atomic.AddUint64(&info.Nonce, 1) // we don't set Nonce, but try to keep track..
			return tx.WithSignature(b.Signer, signature)
		},
		GasMargin: 2000, // adjust by 20%
		Context:   ctx,
	}
}

func (b *BlockchainTestInfo) GetDefaultCallOpts(name string, ctx context.Context) *bind.CallOpts {
	b.T.Helper()
	auth := b.GetDefaultTransactOpts(name, ctx)
	return &bind.CallOpts{
		From: auth.From,
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
	return b.PrepareTxTo(from, &addr, gas, value, data)
}

func (b *BlockchainTestInfo) PrepareTxTo(
	from string, to *common.Address, gas uint64, value *big.Int, data []byte,
) *types.Transaction {
	b.T.Helper()
	info := b.GetInfoWithPrivKey(from)
	txNonce := atomic.AddUint64(&info.Nonce, 1) - 1
	txData := &types.DynamicFeeTx{
		To:        to,
		Gas:       gas,
		GasFeeCap: new(big.Int).Set(b.GasPrice),
		Value:     value,
		Nonce:     txNonce,
		Data:      data,
	}
	return b.SignTxAs(from, txData)
}

func (b *BlockchainTestInfo) PrepareTippingTx(from, to string, gas uint64, tipCap *big.Int, value *big.Int, data []byte) *types.Transaction {
	b.T.Helper()
	addr := b.GetAddress(to)
	return b.PrepareTippingTxTo(from, &addr, gas, tipCap, value, data)
}

func (b *BlockchainTestInfo) PrepareTippingTxTo(
	from string, to *common.Address, gas uint64, tipCap *big.Int, value *big.Int, data []byte,
) *types.Transaction {
	b.T.Helper()
	info := b.GetInfoWithPrivKey(from)
	txNonce := atomic.AddUint64(&info.Nonce, 1) - 1
	feeCap := arbmath.BigAdd(b.GasPrice, tipCap)
	dynamic := types.DynamicFeeTx{
		To:        to,
		Gas:       gas,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		Value:     value,
		Nonce:     txNonce,
		Data:      data,
	}
	txData := &types.ArbitrumSubtypedTx{TxData: &types.ArbitrumTippingTx{DynamicFeeTx: dynamic}}
	return b.SignTxAs(from, txData)
}

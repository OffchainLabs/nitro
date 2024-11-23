// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package txbuilder

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ValidatorWalletInterface interface {
	// Address must be able to be called concurrently with other functions
	Address() *common.Address
	TestTransactions(ctx context.Context, txs []*types.Transaction) error
	ExecuteTransactions(ctx context.Context, txs []*types.Transaction, gasRefunder common.Address) (*types.Transaction, error)
	AuthIfEoa() *bind.TransactOpts
}

// Builder combines any transactions signed via it into one batch,
// which is then sent to the validator wallet.
// This lets the validator make multiple atomic transactions.
type Builder struct {
	transactions []*types.Transaction
	singleTxAuth bind.TransactOpts
	multiTxAuth  bind.TransactOpts
	isAuthFake   bool
	authMutex    sync.Mutex
	wallet       ValidatorWalletInterface
	gasRefunder  common.Address
}

func NewBuilder(wallet ValidatorWalletInterface, gasRefunder common.Address) (*Builder, error) {
	var builderAuth bind.TransactOpts
	var isAuthFake bool
	if authIfEoa := wallet.AuthIfEoa(); authIfEoa != nil {
		builderAuth = *authIfEoa
	} else {
		isAuthFake = true
		var addressOrZero common.Address
		if addr := wallet.Address(); addr != nil {
			addressOrZero = *addr
		}
		builderAuth = bind.TransactOpts{
			From:     addressOrZero,
			GasLimit: 123, // don't gas estimate, that's done when the real tx is created
			Signer: func(_ common.Address, tx *types.Transaction) (*types.Transaction, error) {
				return tx, nil
			},
		}
	}
	builderAuth.NoSend = true
	builder := &Builder{
		singleTxAuth: builderAuth,
		multiTxAuth:  builderAuth,
		wallet:       wallet,
		isAuthFake:   isAuthFake,
		gasRefunder:  gasRefunder,
	}
	originalSigner := builderAuth.Signer
	builder.multiTxAuth.Signer = func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		tx, err := originalSigner(addr, tx)
		if err != nil {
			return nil, err
		}
		// Append the transaction to the builder's queue of transactions
		builder.transactions = append(builder.transactions, tx)
		err = builder.wallet.TestTransactions(context.TODO(), builder.transactions)
		if err != nil {
			// Remove the bad tx
			builder.transactions = builder.transactions[:len(builder.transactions)-1]
			return nil, err
		}
		return tx, nil
	}
	builder.singleTxAuth.Signer = func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if !isAuthFake {
			return originalSigner(addr, tx)
		}
		// Try to process the transaction on its own
		ctx := context.TODO()
		txs := []*types.Transaction{tx}
		err := builder.wallet.TestTransactions(ctx, txs)
		if err != nil {
			return nil, fmt.Errorf("failed to test builder transaction: %w", err)
		}
		signedTx, err := builder.wallet.ExecuteTransactions(ctx, txs, gasRefunder)
		if err != nil {
			return nil, fmt.Errorf("failed to execute builder transaction: %w", err)
		}
		return signedTx, nil
	}
	return builder, nil
}

func (b *Builder) BuildingTransactionCount() int {
	return len(b.transactions)
}

func (b *Builder) ClearTransactions() {
	b.transactions = nil
}

func (b *Builder) tryToFillAuthAddress() {
	if b.multiTxAuth.From == (common.Address{}) {
		if addr := b.wallet.Address(); addr != nil {
			b.multiTxAuth.From = *addr
			b.singleTxAuth.From = *addr
		}
	}
}

func (b *Builder) AuthWithAmount(ctx context.Context, amount *big.Int) *bind.TransactOpts {
	b.authMutex.Lock()
	defer b.authMutex.Unlock()
	b.tryToFillAuthAddress()
	auth := b.multiTxAuth
	auth.Context = ctx
	auth.Value = amount
	return &auth
}

// Auth is the same as AuthWithAmount with a 0 amount specified.
func (b *Builder) Auth(ctx context.Context) *bind.TransactOpts {
	return b.AuthWithAmount(ctx, common.Big0)
}

// SingleTxAuth should be used if you need an auth without the transaction batching of the builder.
func (b *Builder) SingleTxAuth() *bind.TransactOpts {
	b.authMutex.Lock()
	defer b.authMutex.Unlock()
	b.tryToFillAuthAddress()
	auth := b.singleTxAuth
	return &auth
}

func (b *Builder) WalletAddress() *common.Address {
	return b.wallet.Address()
}

func (b *Builder) ExecuteTransactions(ctx context.Context) (*types.Transaction, error) {
	tx, err := b.wallet.ExecuteTransactions(ctx, b.transactions, b.gasRefunder)
	b.ClearTransactions()
	return tx, err
}

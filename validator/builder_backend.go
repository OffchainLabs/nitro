//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/pkg/errors"
)

type BuilderBackend struct {
	transactions []*types.Transaction
	builderAuth  *bind.TransactOpts
	realSender   common.Address
	wallet       *ValidatorWallet

	arbutil.L1Interface
}

func NewBuilderBackend(wallet *ValidatorWallet) (*BuilderBackend, error) {
	randKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	fakeAuth, err := bind.NewKeyedTransactorWithChainID(randKey, big.NewInt(9999999))
	if err != nil {
		return nil, err
	}
	return &BuilderBackend{
		builderAuth: fakeAuth,
		realSender:  wallet.From(),
		wallet:      wallet,
		L1Interface: wallet.client,
	}, nil
}

func (b *BuilderBackend) BuilderTransactionCount() int {
	return len(b.transactions)
}

func (b *BuilderBackend) ClearTransactions() {
	b.transactions = nil
}

func (b *BuilderBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	if call.From == b.builderAuth.From {
		if b.wallet.Address() == nil {
			return 0, nil
		}
		call.From = *b.wallet.Address()
	}
	return b.EstimateGas(ctx, call)
}

func (b *BuilderBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	b.transactions = append(b.transactions, tx)
	data, dest, amount, totalAmount := combineTxes(b.transactions)
	if b.wallet.Address() == nil {
		return nil
	}
	realData, err := validatorABI.Pack("executeTransactions", data, dest, amount)
	if err != nil {
		return err
	}
	msg := ethereum.CallMsg{
		From:  b.realSender,
		To:    b.wallet.Address(),
		Value: totalAmount,
		Data:  realData,
	}
	_, err = b.EstimateGas(ctx, msg)
	return errors.WithStack(err)
}

func (b *BuilderBackend) AuthWithAmount(ctx context.Context, amount *big.Int) *bind.TransactOpts {
	return &bind.TransactOpts{
		From:     b.builderAuth.From,
		Nonce:    b.builderAuth.Nonce,
		Signer:   b.builderAuth.Signer,
		Value:    amount,
		GasPrice: b.builderAuth.GasPrice,
		GasLimit: b.builderAuth.GasLimit,
		Context:  ctx,
	}
}

func (b *BuilderBackend) Auth(ctx context.Context) *bind.TransactOpts {
	return b.AuthWithAmount(ctx, big.NewInt(0))
}

//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
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
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/pkg/errors"
)

// ValidatorTxBuilder combines any transactions sent to it via SendTransaction into one batch,
// which is then sent to the validator wallet.
// This lets the validator make multiple atomic transactions.
// This inherits from an eth client so it can be used as an L1Interface,
// where it transparently intercepts calls to SendTransaction and queues them for the next batch.
type ValidatorTxBuilder struct {
	arbutil.L1Interface
	transactions []*types.Transaction
	builderAuth  *bind.TransactOpts
	realSender   common.Address
	wallet       *ValidatorWallet
}

func NewValidatorTxBuilder(wallet *ValidatorWallet) (*ValidatorTxBuilder, error) {
	randKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	fakeAuth, err := bind.NewKeyedTransactorWithChainID(randKey, big.NewInt(9999999))
	if err != nil {
		return nil, err
	}
	return &ValidatorTxBuilder{
		builderAuth: fakeAuth,
		realSender:  wallet.From(),
		wallet:      wallet,
		L1Interface: wallet.client,
	}, nil
}

func (b *ValidatorTxBuilder) BuildingTransactionCount() int {
	return len(b.transactions)
}

func (b *ValidatorTxBuilder) ClearTransactions() {
	b.transactions = nil
}

func (b *ValidatorTxBuilder) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return 0, nil
}

func (b *ValidatorTxBuilder) SendTransaction(ctx context.Context, tx *types.Transaction) error {
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
	_, err = b.L1Interface.PendingCallContract(ctx, msg)
	return errors.WithStack(err)
}

func (b *ValidatorTxBuilder) AuthWithAmount(ctx context.Context, amount *big.Int) *bind.TransactOpts {
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

func (b *ValidatorTxBuilder) Auth(ctx context.Context) *bind.TransactOpts {
	return b.AuthWithAmount(ctx, big.NewInt(0))
}

// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package validatorwallet

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/solgen/go/challenge_legacy_gen"
)

// EOA is a ValidatorWallet that uses an Externally Owned Account to sign transactions.
// An Ethereum Externally Owned Account is directly represented by a private key,
// as opposed to a smart contract wallet where the smart contract authorizes transactions.
type EOA struct {
	auth        *bind.TransactOpts
	client      *ethclient.Client
	dataPoster  *dataposter.DataPoster
	getExtraGas func() uint64
}

func NewEOA(dataPoster *dataposter.DataPoster, l1Client *ethclient.Client, getExtraGas func() uint64) (*EOA, error) {
	return &EOA{
		auth:        dataPoster.Auth(),
		client:      l1Client,
		dataPoster:  dataPoster,
		getExtraGas: getExtraGas,
	}, nil
}

func (w *EOA) Initialize(ctx context.Context) error {
	return nil
}

func (w *EOA) InitializeAndCreateSCW(ctx context.Context) error {
	return nil
}

func (w *EOA) Address() *common.Address {
	return &w.auth.From
}

func (w *EOA) AddressOrZero() common.Address {
	return w.auth.From
}

func (w *EOA) TxSenderAddress() *common.Address {
	return &w.auth.From
}

func (w *EOA) L1Client() *ethclient.Client {
	return w.client
}

func (w *EOA) TestTransactions(context.Context, []*types.Transaction) error {
	// We only use the first tx which is checked implicitly by gas estimation
	return nil
}

func (w *EOA) ExecuteTransactions(ctx context.Context, txes []*types.Transaction, _ common.Address) (*types.Transaction, error) {
	if len(txes) == 0 {
		return nil, nil
	}
	tx := txes[0] // we ignore future txs and only execute the first
	return w.postTransaction(ctx, tx)
}

func (w *EOA) postTransaction(ctx context.Context, baseTx *types.Transaction) (*types.Transaction, error) {
	gas := baseTx.Gas() + w.getExtraGas()
	newTx, err := w.dataPoster.PostSimpleTransaction(ctx, *baseTx.To(), baseTx.Data(), gas, baseTx.Value())
	if err != nil {
		return nil, fmt.Errorf("post transaction: %w", err)
	}
	return newTx, nil
}

func (w *EOA) TimeoutChallenges(ctx context.Context, timeouts []uint64, challengeManagerAddress common.Address) (*types.Transaction, error) {
	if len(timeouts) == 0 {
		return nil, nil
	}
	auth := *w.auth
	auth.Context = ctx
	auth.NoSend = true
	challengeManager, err := challenge_legacy_gen.NewChallengeManager(challengeManagerAddress, w.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge manager: %w", err)
	}
	tx, err := challengeManager.Timeout(&auth, timeouts[0])
	if err != nil {
		return nil, err
	}
	return w.postTransaction(ctx, tx)
}

func (w *EOA) CanBatchTxs() bool {
	return false
}

func (w *EOA) AuthIfEoa() *bind.TransactOpts {
	return w.auth
}

func (w *EOA) Start(ctx context.Context) {
	w.dataPoster.Start(ctx)
}

func (b *EOA) StopAndWait() {
	b.dataPoster.StopAndWait()
}

func (b *EOA) DataPoster() *dataposter.DataPoster {
	return b.dataPoster
}

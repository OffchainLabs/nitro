// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validatorwallet

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/txbuilder"
)

type EOA struct {
	auth                    *bind.TransactOpts
	client                  arbutil.L1Interface
	rollupAddress           common.Address
	challengeManager        *challengegen.ChallengeManager
	challengeManagerAddress common.Address
	dataPoster              *dataposter.DataPoster
	getExtraGas             func() uint64
}

func NewEOA(dataPoster *dataposter.DataPoster, rollupAddress common.Address, l1Client arbutil.L1Interface, getExtraGas func() uint64) (*EOA, error) {
	return &EOA{
		auth:          dataPoster.Auth(),
		client:        l1Client,
		rollupAddress: rollupAddress,
		dataPoster:    dataPoster,
		getExtraGas:   getExtraGas,
	}, nil
}

func (w *EOA) Initialize(ctx context.Context) error {
	rollup, err := rollupgen.NewRollupUserLogic(w.rollupAddress, w.client)
	if err != nil {
		return err
	}
	callOpts := &bind.CallOpts{Context: ctx}
	w.challengeManagerAddress, err = rollup.ChallengeManager(callOpts)
	if err != nil {
		return err
	}
	w.challengeManager, err = challengegen.NewChallengeManager(w.challengeManagerAddress, w.client)
	return err
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

func (w *EOA) L1Client() arbutil.L1Interface {
	return w.client
}

func (w *EOA) RollupAddress() common.Address {
	return w.rollupAddress
}

func (w *EOA) ChallengeManagerAddress() common.Address {
	return w.challengeManagerAddress
}

func (w *EOA) TestTransactions(context.Context, []*types.Transaction) error {
	// We only use the first tx which is checked implicitly by gas estimation
	return nil
}

func (w *EOA) ExecuteTransactions(ctx context.Context, builder *txbuilder.Builder, _ common.Address) (*types.Transaction, error) {
	if len(builder.Transactions()) == 0 {
		return nil, nil
	}
	tx := builder.Transactions()[0] // we ignore future txs and only execute the first
	return w.postTransaction(ctx, tx)
}

func (w *EOA) postTransaction(ctx context.Context, baseTx *types.Transaction) (*types.Transaction, error) {
	nonce, err := w.L1Client().NonceAt(ctx, w.auth.From, nil)
	if err != nil {
		return nil, err
	}
	gas := baseTx.Gas() + w.getExtraGas()
	newTx, err := w.dataPoster.PostSimpleTransaction(ctx, nonce, *baseTx.To(), baseTx.Data(), gas, baseTx.Value())
	if err != nil {
		return nil, fmt.Errorf("post transaction: %w", err)
	}
	return newTx, nil
}

func (w *EOA) TimeoutChallenges(ctx context.Context, timeouts []uint64) (*types.Transaction, error) {
	if len(timeouts) == 0 {
		return nil, nil
	}
	auth := *w.auth
	auth.Context = ctx
	auth.NoSend = true
	tx, err := w.challengeManager.Timeout(&auth, timeouts[0])
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

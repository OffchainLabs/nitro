// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validatorwallet

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/staker/txbuilder"
)

// NoOp validator wallet is used for watchtower mode.
type NoOp struct {
	l1Client arbutil.L1Interface
}

func NewNoOp(l1Client arbutil.L1Interface) *NoOp {
	return &NoOp{l1Client: l1Client}
}

func (*NoOp) Initialize(context.Context) error { return nil }

func (*NoOp) Address() *common.Address { return nil }

func (*NoOp) AddressOrZero() common.Address { return common.Address{} }

func (*NoOp) TxSenderAddress() *common.Address { return nil }

func (*NoOp) From() common.Address { return common.Address{} }

func (*NoOp) ExecuteTransactions(context.Context, *txbuilder.Builder, common.Address) (*types.Transaction, error) {
	return nil, errors.New("no op validator wallet cannot execute transactions")
}

func (*NoOp) TimeoutChallenges(ctx context.Context, challenges []uint64) (*types.Transaction, error) {
	return nil, errors.New("no op validator wallet cannot timeout challenges")
}

func (n *NoOp) L1Client() arbutil.L1Interface { return n.l1Client }

func (*NoOp) RollupAddress() common.Address { return common.Address{} }

func (*NoOp) ChallengeManagerAddress() common.Address { return common.Address{} }

func (*NoOp) TestTransactions(ctx context.Context, txs []*types.Transaction) error {
	return nil
}

func (*NoOp) CanBatchTxs() bool { return false }

func (*NoOp) AuthIfEoa() *bind.TransactOpts { return nil }

func (w *NoOp) Start(ctx context.Context) {}

func (b *NoOp) StopAndWait() {}

func (b *NoOp) DataPoster() *dataposter.DataPoster { return nil }

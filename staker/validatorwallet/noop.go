// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package validatorwallet

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
)

// NoOp validator wallet is used for watchtower mode.
type NoOp struct {
	l1Client *ethclient.Client
}

func NewNoOp(l1Client *ethclient.Client) *NoOp {
	return &NoOp{
		l1Client: l1Client,
	}
}

func (*NoOp) Initialize(context.Context) error { return nil }

func (*NoOp) InitializeAndCreateSCW(context.Context) error { return nil }

func (*NoOp) Address() *common.Address { return nil }

func (*NoOp) AddressOrZero() common.Address { return common.Address{} }

func (*NoOp) TxSenderAddress() *common.Address { return nil }

func (*NoOp) From() common.Address { return common.Address{} }

func (*NoOp) ExecuteTransactions(context.Context, []*types.Transaction, common.Address) (*types.Transaction, error) {
	return nil, errors.New("no op validator wallet cannot execute transactions")
}

func (*NoOp) TimeoutChallenges(ctx context.Context, challenges []uint64, challengeManagerAddress common.Address) (*types.Transaction, error) {
	return nil, errors.New("no op validator wallet cannot timeout challenges")
}

func (n *NoOp) L1Client() *ethclient.Client { return n.l1Client }

func (*NoOp) TestTransactions(ctx context.Context, txs []*types.Transaction) error {
	return nil
}

func (*NoOp) CanBatchTxs() bool { return false }

func (*NoOp) AuthIfEoa() *bind.TransactOpts { return nil }

func (w *NoOp) Start(ctx context.Context) {}

func (b *NoOp) StopAndWait() {}

func (b *NoOp) DataPoster() *dataposter.DataPoster { return nil }

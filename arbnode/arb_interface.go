//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransactionPublisher interface {
	PublishTransaction(ctx context.Context, tx *types.Transaction) error
	Initialize(context.Context) error
	Start(context.Context) (*Stopper, error)
}

type ArbInterface struct {
	txStreamer  *TransactionStreamer
	txPublisher TransactionPublisher
}

func NewArbInterface(txStreamer *TransactionStreamer, txPublisher TransactionPublisher) (*ArbInterface, error) {
	return &ArbInterface{
		txStreamer:  txStreamer,
		txPublisher: txPublisher,
	}, nil
}

func (a *ArbInterface) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	return a.txPublisher.PublishTransaction(ctx, tx)
}

func (a *ArbInterface) TransactionStreamer() *TransactionStreamer {
	return a.txStreamer
}

func (a *ArbInterface) BlockChain() *core.BlockChain {
	return a.txStreamer.bc
}

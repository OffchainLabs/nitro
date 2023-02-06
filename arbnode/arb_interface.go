// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransactionPublisher interface {
	PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error
	CheckHealth(ctx context.Context) error
	Initialize(context.Context) error
	Start(context.Context) error
	StopAndWait()
	Started() bool
}

type ArbInterface struct {
	txStreamer  *TransactionStreamer
	txPublisher TransactionPublisher
	arbNode     *Node
}

func NewArbInterface(txStreamer *TransactionStreamer, txPublisher TransactionPublisher) (*ArbInterface, error) {
	return &ArbInterface{
		txStreamer:  txStreamer,
		txPublisher: txPublisher,
	}, nil
}

func (a *ArbInterface) Initialize(n *Node) {
	a.arbNode = n
}

func (a *ArbInterface) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return a.txPublisher.PublishTransaction(ctx, tx, options)
}

func (a *ArbInterface) TransactionStreamer() *TransactionStreamer {
	return a.txStreamer
}

func (a *ArbInterface) BlockChain() *core.BlockChain {
	return a.txStreamer.bc
}

func (a *ArbInterface) ArbNode() interface{} {
	return a.arbNode
}

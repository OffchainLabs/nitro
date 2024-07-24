// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package gethexec

import (
	"context"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

type TransactionPublisher interface {
	PublishExpressLaneTransaction(ctx context.Context, msg *arbitrum_types.ExpressLaneSubmission) error
	PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error
	CheckHealth(ctx context.Context) error
	Initialize(context.Context) error
	Start(context.Context) error
	StopAndWait()
	Started() bool
}

type ArbInterface struct {
	blockchain  *core.BlockChain
	node        *ExecutionNode
	txPublisher TransactionPublisher
}

func NewArbInterface(blockchain *core.BlockChain, txPublisher TransactionPublisher) (*ArbInterface, error) {
	return &ArbInterface{
		blockchain:  blockchain,
		txPublisher: txPublisher,
	}, nil
}

func (a *ArbInterface) Initialize(node *ExecutionNode) {
	a.node = node
}

func (a *ArbInterface) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return a.txPublisher.PublishTransaction(ctx, tx, options)
}

func (a *ArbInterface) PublishExpressLaneTransaction(ctx context.Context, msg *arbitrum_types.ExpressLaneSubmission) error {
	return a.txPublisher.PublishExpressLaneTransaction(ctx, msg)
}

// might be used before Initialize
func (a *ArbInterface) BlockChain() *core.BlockChain {
	return a.blockchain
}

func (a *ArbInterface) ArbNode() interface{} {
	return a.node
}

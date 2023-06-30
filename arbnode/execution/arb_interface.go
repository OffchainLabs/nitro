// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package execution

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
	exec        *ExecutionEngine
	txPublisher TransactionPublisher
	arbNode     interface{}
}

func NewArbInterface(exec *ExecutionEngine, txPublisher TransactionPublisher) (*ArbInterface, error) {
	return &ArbInterface{
		exec:        exec,
		txPublisher: txPublisher,
	}, nil
}

func (a *ArbInterface) Initialize(arbnode interface{}) {
	a.arbNode = arbnode
}

func (a *ArbInterface) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return a.txPublisher.PublishTransaction(ctx, tx, options)
}

func (a *ArbInterface) BlockChain() *core.BlockChain {
	return a.exec.bc
}

func (a *ArbInterface) ArbNode() interface{} {
	return a.arbNode
}

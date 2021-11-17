//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TxForwarder struct {
	target string
	client *ethclient.Client
}

func NewForwarder(target string) (*TxForwarder, error) {
	return &TxForwarder{
		target: target,
	}, nil
}

func (f *TxForwarder) PublishTransaction(ctx context.Context, tx *types.Transaction) error {
	return f.client.SendTransaction(ctx, tx)
}

func (f *TxForwarder) Initialize(ctx context.Context) error {
	client, err := ethclient.DialContext(ctx, f.target)
	if err != nil {
		return err
	}
	f.client = client
	return nil
}

func (f *TxForwarder) Start(ctx context.Context) (*Stopper, error) {
	return nil, nil
}

// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type TxFiltererManagerAPI struct {
}

func (t *TxFiltererManagerAPI) Filter(ctx context.Context, txHash common.Hash) error {
	log.Info("Received request to filter transaction", "txHash", txHash.Hex())
	return nil
}

func RegisterAPI(stack *node.Node) {
	api := &TxFiltererManagerAPI{}
	apis := []rpc.API{{
		Namespace: "txfilterermanager",
		Version:   "1.0",
		Service:   api,
		Public:    true,
	}}
	stack.RegisterAPIs(apis)
}

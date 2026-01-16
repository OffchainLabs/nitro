// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

const namespace = "transactionfilterer"

type TransactionFiltererAPI struct {
	sequencerClient *ethclient.Client
	txOpts          *bind.TransactOpts
}

func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHash common.Hash) error {
	log.Info("Received request to filter transaction", "txHash", txHash.Hex())
	return nil
}

var DefaultStackConfig = node.Config{
	DataDir:             "", // ephemeral
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{namespace},
	HTTPHost:            node.DefaultHTTPHost,
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSHost:              node.DefaultWSHost,
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{namespace},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr:  "",
		NoDiscovery: true,
		NoDial:      true,
	},
}

func NewStack(
	ctx context.Context,
	stackConfig *node.Config,
	txOpts *bind.TransactOpts,
	sequencerClient *ethclient.Client,
) (*node.Node, error) {
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, err
	}

	api := &TransactionFiltererAPI{
		sequencerClient: sequencerClient,
		txOpts:          txOpts,
	}
	apis := []rpc.API{{
		Namespace: namespace,
		Version:   "1.0",
		Service:   api,
		Public:    true,
	}}
	stack.RegisterAPIs(apis)

	return stack, nil
}

// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

const namespace = "transactionfilterer"

type TransactionFiltererAPI struct {
	arbFilteredTransactionsManager *precompilesgen.ArbFilteredTransactionsManager
	txOpts                         *bind.TransactOpts
}

func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHashToFilter common.Hash) (common.Hash, error) {
	txOpts := *t.txOpts
	txOpts.Context = ctx

	log.Info("Received call to filter transaction", "txHashToFilter", txHashToFilter.Hex())
	tx, err := t.arbFilteredTransactionsManager.AddFilteredTransaction(&txOpts, txHashToFilter)
	if err != nil {
		log.Warn("Failed to filter transaction", "txHashToFilter", txHashToFilter.Hex(), "err", err)
		return common.Hash{}, err
	} else {
		log.Info("Submitted filter transaction", "txHashToFilter", txHashToFilter.Hex(), "txHash", tx.Hash().Hex())
		return tx.Hash(), nil
	}
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
	stackConfig *node.Config,
	txOpts *bind.TransactOpts,
	sequencerClient *ethclient.Client,
) (*node.Node, error) {
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, err
	}

	arbFilteredTransactionsManager, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		sequencerClient,
	)
	if err != nil {
		return nil, err
	}

	api := &TransactionFiltererAPI{
		arbFilteredTransactionsManager: arbFilteredTransactionsManager,
		txOpts:                         txOpts,
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

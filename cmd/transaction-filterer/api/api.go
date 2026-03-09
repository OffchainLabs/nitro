// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const filterQueueSize = 100

type TransactionFiltererAPI struct {
	stopwaiter.StopWaiter

	queue chan func()

	arbFilteredTransactionsManager *precompilesgen.ArbFilteredTransactionsManager
	txOpts                         *bind.TransactOpts
}

func NewTransactionFiltererAPI(
	manager *precompilesgen.ArbFilteredTransactionsManager,
	txOpts *bind.TransactOpts,
) *TransactionFiltererAPI {
	return &TransactionFiltererAPI{
		queue:                          make(chan func(), filterQueueSize),
		arbFilteredTransactionsManager: manager,
		txOpts:                         txOpts,
	}
}

func (t *TransactionFiltererAPI) Start(ctx context.Context) error {
	t.StopWaiter.Start(ctx, t)
	return stopwaiter.CallWhenTriggeredWith(&t.StopWaiterSafe, func(_ context.Context, work func()) {
		work()
	}, t.queue)
}

// Filter adds the given transaction hash to the filtered transactions set,
// which is managed by the ArbFilteredTransactionsManager precompile.
// Requests are processed sequentially by a single consumer goroutine to avoid nonce collisions.
func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHashToFilter common.Hash) error {
	result := make(chan error, 1)
	select {
	case t.queue <- func() {
		if ctx.Err() != nil {
			result <- ctx.Err()
		} else {
			result <- t.filter(ctx, txHashToFilter)
		}
	}:
		return <-result
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *TransactionFiltererAPI) filter(ctx context.Context, txHashToFilter common.Hash) error {
	txOpts := *t.txOpts
	txOpts.Context = ctx

	log.Info("Received call to filter transaction", "txHashToFilter", txHashToFilter.Hex())
	if t.arbFilteredTransactionsManager == nil {
		return errors.New("sequencer client not set yet")
	}
	tx, err := t.arbFilteredTransactionsManager.AddFilteredTransaction(&txOpts, txHashToFilter)
	if err != nil {
		log.Warn("Failed to filter transaction", "txHashToFilter", txHashToFilter.Hex(), "err", err)
		return err
	}
	log.Info("Submitted filter transaction", "txHashToFilter", txHashToFilter.Hex(), "txHash", tx.Hash().Hex())
	return nil
}

// Only used for testing.
// Sequencer and TransactionFiltererAPI depend on each other, as a workaround for the egg/chicken problem,
// we set the sequencer client after both are created.
func (t *TransactionFiltererAPI) SetSequencerClient(_ *testing.T, sequencerClient *ethclient.Client) error {
	if sequencerClient == nil {
		return errors.New("cannot set nil sequencer client")
	}
	arbFilteredTransactionsManager, err := precompilesgen.NewArbFilteredTransactionsManager(
		types.ArbFilteredTransactionsManagerAddress,
		sequencerClient,
	)
	if err != nil {
		return err
	}
	t.arbFilteredTransactionsManager = arbFilteredTransactionsManager
	return nil
}

var DefaultStackConfig = node.Config{
	DataDir:             "", // ephemeral
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{gethexec.TransactionFiltererNamespace},
	HTTPHost:            node.DefaultHTTPHost,
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSHost:              node.DefaultWSHost,
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{gethexec.TransactionFiltererNamespace},
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
) (*node.Node, *TransactionFiltererAPI, error) {
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, nil, err
	}

	var arbFilteredTransactionsManager *precompilesgen.ArbFilteredTransactionsManager
	if sequencerClient != nil {
		arbFilteredTransactionsManager, err = precompilesgen.NewArbFilteredTransactionsManager(
			types.ArbFilteredTransactionsManagerAddress,
			sequencerClient,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	api := NewTransactionFiltererAPI(arbFilteredTransactionsManager, txOpts)
	apis := []rpc.API{{
		Namespace: gethexec.TransactionFiltererNamespace,
		Version:   "1.0",
		Service:   api,
		Public:    true,
	}}
	stack.RegisterAPIs(apis)

	return stack, api, nil
}

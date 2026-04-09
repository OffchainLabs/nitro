// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
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

	queue chan common.Hash

	arbFilteredTransactionsManager atomic.Pointer[precompilesgen.ArbFilteredTransactionsManager]
	txOpts                         *bind.TransactOpts
}

func NewTransactionFiltererAPI(
	manager *precompilesgen.ArbFilteredTransactionsManager,
	txOpts *bind.TransactOpts,
) *TransactionFiltererAPI {
	api := &TransactionFiltererAPI{
		queue:  make(chan common.Hash, filterQueueSize),
		txOpts: txOpts,
	}
	api.arbFilteredTransactionsManager.Store(manager)
	return api
}

func (t *TransactionFiltererAPI) Start(ctx context.Context) error {
	t.StopWaiter.Start(ctx, t)
	return stopwaiter.CallWhenTriggeredWith(&t.StopWaiterSafe, t.filter, t.queue)
}

// Filter adds the given transaction hash to the filtered transactions set,
// which is managed by the ArbFilteredTransactionsManager precompile.
// Requests are processed sequentially by a single consumer goroutine to avoid nonce collisions.
func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHashToFilter common.Hash) error {
	select {
	case t.queue <- txHashToFilter:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *TransactionFiltererAPI) filter(ctx context.Context, txHashToFilter common.Hash) {
	txOpts := *t.txOpts
	txOpts.Context = ctx

	log.Info("Received call to filter transaction", "txHashToFilter", txHashToFilter.Hex())
	manager := t.arbFilteredTransactionsManager.Load()
	if manager == nil {
		log.Warn("Sequencer client not set yet")
		return
	}
	tx, err := manager.AddFilteredTransaction(&txOpts, txHashToFilter)
	if err != nil {
		log.Warn("Failed to filter transaction", "txHashToFilter", txHashToFilter.Hex(), "err", err)
		return
	}
	log.Info("Submitted filter transaction", "txHashToFilter", txHashToFilter.Hex(), "txHash", tx.Hash().Hex())
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
	t.arbFilteredTransactionsManager.Store(arbFilteredTransactionsManager)
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

	api := &TransactionFiltererAPI{
		txOpts: txOpts,
	}
	if arbFilteredTransactionsManager != nil {
		api.arbFilteredTransactionsManager.Store(arbFilteredTransactionsManager)
	}
	apis := []rpc.API{{
		Namespace: gethexec.TransactionFiltererNamespace,
		Version:   "1.0",
		Service:   api,
		Public:    true,
	}}
	stack.RegisterAPIs(apis)

	stack.RegisterHandler("liveness", "/liveness", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	stack.RegisterHandler("readiness", "/readiness", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return stack, api, nil
}

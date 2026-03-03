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

type filterRequest struct {
	ctx  context.Context
	hash common.Hash
	// response fields, written by consumer before closing done
	txHash common.Hash
	err    error
	done   chan struct{}
}

type TransactionFiltererAPI struct {
	stopwaiter.StopWaiter

	queue chan *filterRequest

	arbFilteredTransactionsManager *precompilesgen.ArbFilteredTransactionsManager
	txOpts                         *bind.TransactOpts
}

func NewTransactionFiltererAPI(
	manager *precompilesgen.ArbFilteredTransactionsManager,
	txOpts *bind.TransactOpts,
) *TransactionFiltererAPI {
	return &TransactionFiltererAPI{
		arbFilteredTransactionsManager: manager,
		txOpts:                         txOpts,
		queue:                          make(chan *filterRequest, filterQueueSize),
	}
}

func (t *TransactionFiltererAPI) Start(ctx context.Context) error {
	t.StopWaiter.Start(ctx, t)
	return stopwaiter.CallWhenTriggeredWith(&t.StopWaiterSafe, func(_ context.Context, req *filterRequest) {
		if req.ctx.Err() != nil {
			req.err = req.ctx.Err()
		} else {
			req.txHash, req.err = t.filter(req.ctx, req.hash)
		}
		close(req.done)
	}, t.queue)
}

func (t *TransactionFiltererAPI) filter(ctx context.Context, txHashToFilter common.Hash) (common.Hash, error) {
	if t.arbFilteredTransactionsManager == nil {
		return common.Hash{}, errors.New("sequencer client not set yet")
	}
	txOpts := *t.txOpts
	txOpts.Context = ctx
	log.Info("Received call to filter transaction", "txHashToFilter", txHashToFilter.Hex())
	tx, err := t.arbFilteredTransactionsManager.AddFilteredTransaction(&txOpts, txHashToFilter)
	if err != nil {
		log.Warn("Failed to filter transaction", "txHashToFilter", txHashToFilter.Hex(), "err", err)
		return common.Hash{}, err
	}
	log.Info("Submitted filter transaction", "txHashToFilter", txHashToFilter.Hex(), "txHash", tx.Hash().Hex())
	return tx.Hash(), nil
}

// Filter adds the given transaction hash to the filtered transactions set, which is managed by the ArbFilteredTransactionsManager precompile.
func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHashToFilter common.Hash) (common.Hash, error) {
	req := &filterRequest{
		ctx:  ctx,
		hash: txHashToFilter,
		done: make(chan struct{}),
	}
	select {
	case t.queue <- req:
	case <-ctx.Done():
		return common.Hash{}, ctx.Err()
	}
	select {
	case <-req.done:
		return req.txHash, req.err
	case <-ctx.Done():
		return common.Hash{}, ctx.Err()
	}
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

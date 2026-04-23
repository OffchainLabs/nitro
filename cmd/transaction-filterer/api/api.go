// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

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

	filterQueue   chan common.Hash
	unfilterQueue chan common.Hash

	arbFilteredTransactionsManager atomic.Pointer[precompilesgen.ArbFilteredTransactionsManager]
	txOpts                         *bind.TransactOpts

	prune pruner
}

func NewTransactionFiltererAPI(
	manager *precompilesgen.ArbFilteredTransactionsManager,
	txOpts *bind.TransactOpts,
	pruneOpts *PruneOptions,
) *TransactionFiltererAPI {
	api := &TransactionFiltererAPI{
		filterQueue:   make(chan common.Hash, filterQueueSize),
		unfilterQueue: make(chan common.Hash, filterQueueSize),
		txOpts:        txOpts,
	}
	api.arbFilteredTransactionsManager.Store(manager)
	api.prune = newPruner(pruneOpts)
	return api
}

func (t *TransactionFiltererAPI) Start(ctx context.Context) error {
	t.StopWaiter.Start(ctx, t)
	t.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case h := <-t.filterQueue:
				t.filter(ctx, h)
			case h := <-t.unfilterQueue:
				t.unfilter(ctx, h)
			}
		}
	})
	if t.prune.config.Enable {
		t.CallIteratively(func(ctx context.Context) time.Duration {
			result, err := t.prune.step(ctx)
			if err != nil {
				log.Warn("pruner step failed", "err", err)
				return t.prune.config.PollInterval
			}
			t.checkAndUnfilter(ctx, result)
			return t.prune.config.PollInterval
		})
	}
	return nil
}

// checkAndUnfilter checks each candidate hash against the on-chain filter set at the
// finalized child-chain block and enqueues those that are filtered for removal via the shared
// consumer. IsTransactionFiltered is retried on transient RPC errors because the pruner cursor
// has already advanced past these hashes — a persistent check failure leaves the entry filtered
// until operator intervention.
func (t *TransactionFiltererAPI) checkAndUnfilter(ctx context.Context, result pruneResult) {
	if len(result.Hashes) == 0 {
		return
	}
	manager := t.arbFilteredTransactionsManager.Load()
	if manager == nil {
		log.Warn("Sequencer client not set yet")
		return
	}
	callOpts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: result.FinalizedChildNumber,
	}
	for _, h := range result.Hashes {
		filtered, err := isTransactionFilteredWithRetry(ctx, manager, callOpts, h)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			log.Error("IsTransactionFiltered failed after retries; hash stays filtered", "txHash", h.Hex(), "err", err)
			continue
		}
		if !filtered {
			continue
		}
		select {
		case t.unfilterQueue <- h:
			log.Info("enqueued unfilter", "txHash", h.Hex())
		case <-ctx.Done():
			return
		}
	}
}

const (
	isTransactionFilteredRetries    = 3
	isTransactionFilteredRetryDelay = 100 * time.Millisecond
)

func isTransactionFilteredWithRetry(
	ctx context.Context,
	manager *precompilesgen.ArbFilteredTransactionsManager,
	callOpts *bind.CallOpts,
	h common.Hash,
) (bool, error) {
	var err error
	for attempt := range isTransactionFilteredRetries {
		var filtered bool
		filtered, err = manager.IsTransactionFiltered(callOpts, h)
		if err == nil {
			return filtered, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, err
		}
		backoff := time.Duration(attempt+1) * isTransactionFilteredRetryDelay
		log.Warn("IsTransactionFiltered attempt failed; will retry", "txHash", h.Hex(), "attempt", attempt+1, "err", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return false, err
}

// Filter adds the given transaction hash to the filtered transactions set,
// which is managed by the ArbFilteredTransactionsManager precompile.
// Requests are processed sequentially by a single consumer goroutine to avoid nonce collisions.
func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHashToFilter common.Hash) error {
	return t.enqueue(ctx, t.filterQueue, txHashToFilter)
}

// Unfilter removes the given transaction hash from the filtered transactions set via the shared consumer.
func (t *TransactionFiltererAPI) Unfilter(ctx context.Context, txHashToUnfilter common.Hash) error {
	return t.enqueue(ctx, t.unfilterQueue, txHashToUnfilter)
}

func (t *TransactionFiltererAPI) enqueue(ctx context.Context, q chan<- common.Hash, h common.Hash) error {
	select {
	case q <- h:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *TransactionFiltererAPI) filter(ctx context.Context, h common.Hash) {
	t.submit(ctx, "filter", h, func(m *precompilesgen.ArbFilteredTransactionsManager, o *bind.TransactOpts) (*types.Transaction, error) {
		return m.AddFilteredTransaction(o, h)
	})
}

func (t *TransactionFiltererAPI) unfilter(ctx context.Context, h common.Hash) {
	t.submit(ctx, "unfilter", h, func(m *precompilesgen.ArbFilteredTransactionsManager, o *bind.TransactOpts) (*types.Transaction, error) {
		return m.DeleteFilteredTransaction(o, h)
	})
}

func (t *TransactionFiltererAPI) submit(
	ctx context.Context,
	op string,
	h common.Hash,
	call func(*precompilesgen.ArbFilteredTransactionsManager, *bind.TransactOpts) (*types.Transaction, error),
) {
	txOpts := *t.txOpts
	txOpts.Context = ctx

	log.Info("Received "+op+" request", "txHash", h.Hex())
	manager := t.arbFilteredTransactionsManager.Load()
	if manager == nil {
		log.Warn("Sequencer client not set yet")
		return
	}
	tx, err := call(manager, &txOpts)
	if err != nil {
		log.Warn("Failed to "+op+" transaction", "txHash", h.Hex(), "err", err)
		return
	}
	log.Info("Submitted "+op+" transaction", "txHash", h.Hex(), "onchainTxHash", tx.Hash().Hex())
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
	pruneOpts *PruneOptions,
) (*node.Node, *TransactionFiltererAPI, error) {
	if err := validatePruneOptions(pruneOpts); err != nil {
		return nil, nil, err
	}

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

	api := NewTransactionFiltererAPI(arbFilteredTransactionsManager, txOpts, pruneOpts)
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

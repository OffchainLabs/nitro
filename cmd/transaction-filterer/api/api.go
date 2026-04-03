// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
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
)

type TransactionFiltererAPI struct {
	apiMutex sync.Mutex // avoids concurrent transactions with the same nonce

	arbFilteredTransactionsManager *precompilesgen.ArbFilteredTransactionsManager
	txOpts                         *bind.TransactOpts
	filterSetReportingEndpoint     string
}

// Filter adds the given transaction hash to the filtered transactions set, which is managed by the ArbFilteredTransactionsManager precompile.
func (t *TransactionFiltererAPI) Filter(ctx context.Context, txHashToFilter common.Hash) (common.Hash, error) {
	t.apiMutex.Lock()
	defer t.apiMutex.Unlock()

	txOpts := *t.txOpts
	txOpts.Context = ctx

	log.Info("Received call to filter transaction", "txHashToFilter", txHashToFilter.Hex())
	if t.arbFilteredTransactionsManager == nil {
		return common.Hash{}, errors.New("sequencer client not set yet")
	}
	tx, err := t.arbFilteredTransactionsManager.AddFilteredTransaction(&txOpts, txHashToFilter)
	if err != nil {
		log.Warn("Failed to filter transaction", "txHashToFilter", txHashToFilter.Hex(), "err", err)
		return common.Hash{}, err
	} else {
		log.Info("Submitted filter transaction", "txHashToFilter", txHashToFilter.Hex(), "txHash", tx.Hash().Hex())
		return tx.Hash(), nil
	}
}

// ReportCurrentFilterSetId POSTs the given filter set ID to the configured external reporting endpoint.
func (t *TransactionFiltererAPI) ReportCurrentFilterSetId(ctx context.Context, filterSetId string) error {
	if t.filterSetReportingEndpoint == "" {
		return errors.New("filter set reporting endpoint not configured")
	}
	payload, err := json.Marshal(map[string]string{"filterSetId": filterSetId})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.filterSetReportingEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST filter set id: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	log.Info("Reported filter set id", "filterSetId", filterSetId)
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

	t.apiMutex.Lock()
	defer t.apiMutex.Unlock()
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
	filterSetReportingEndpoint string,
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
		arbFilteredTransactionsManager: arbFilteredTransactionsManager,
		txOpts:                         txOpts,
		filterSetReportingEndpoint:     filterSetReportingEndpoint,
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

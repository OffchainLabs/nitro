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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/execution/gethexec"
)

type FilteringReportAPI struct {
	filterSetReportingEndpoint string
}

var DefaultStackConfig = node.Config{
	DataDir:             "", // ephemeral
	HTTPPort:            node.DefaultHTTPPort,
	AuthAddr:            node.DefaultAuthHost,
	AuthPort:            node.DefaultAuthPort,
	AuthVirtualHosts:    node.DefaultAuthVhosts,
	HTTPModules:         []string{gethexec.FilteringReportNamespace},
	HTTPHost:            node.DefaultHTTPHost,
	HTTPVirtualHosts:    []string{"localhost"},
	HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
	WSHost:              node.DefaultWSHost,
	WSPort:              node.DefaultWSPort,
	WSModules:           []string{gethexec.FilteringReportNamespace},
	GraphQLVirtualHosts: []string{"localhost"},
	P2P: p2p.Config{
		ListenAddr:  "",
		NoDiscovery: true,
		NoDial:      true,
	},
}

// ReportCurrentFilterSetId POSTs the given filter set ID to the configured external reporting endpoint.
func (a *FilteringReportAPI) ReportCurrentFilterSetId(ctx context.Context, filterSetId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if a.filterSetReportingEndpoint == "" {
		return errors.New("filter set reporting endpoint not configured")
	}
	payload, err := json.Marshal(map[string]string{"filterSetId": filterSetId})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.filterSetReportingEndpoint, bytes.NewReader(payload))
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

func NewStack(stackConfig *node.Config, filterSetReportingEndpoint string) (*node.Node, error) {
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, err
	}

	reportAPI := &FilteringReportAPI{
		filterSetReportingEndpoint: filterSetReportingEndpoint,
	}
	apis := []rpc.API{{
		Namespace: gethexec.FilteringReportNamespace,
		Version:   "1.0",
		Service:   reportAPI,
		Public:    true,
	}}
	stack.RegisterAPIs(apis)

	stack.RegisterHandler("liveness", "/liveness", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	stack.RegisterHandler("readiness", "/readiness", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return stack, nil
}

// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"errors"
	"net/http"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

type filterSetReporter struct {
	url    string
	client *http.Client
	signer *signer.Signer
}

type FilteringReportAPI struct {
	queueClient       sqsclient.QueueClient
	filterSetReporter *filterSetReporter
}

// NewFilteringReportAPI builds the RPC service. When filter-set-id reporting is
// enabled (a non-empty URL), sgn must be non-nil: reports are forwarded to an
// external endpoint that verifies the service's signature, so sgn is shared
// with the forwarder to reuse a single signing identity.
func NewFilteringReportAPI(queueClient sqsclient.QueueClient, filterSetReporting *genericconf.HTTPClientConfig, sgn *signer.Signer) (*FilteringReportAPI, error) {
	if queueClient == nil {
		return nil, errors.New("queueClient must not be nil")
	}
	api := &FilteringReportAPI{queueClient: queueClient}
	if filterSetReporting != nil && filterSetReporting.URL != "" {
		if sgn == nil {
			return nil, errors.New("signer must not be nil when filter-set-id reporting is enabled")
		}
		api.filterSetReporter = &filterSetReporter{
			url:    filterSetReporting.URL,
			client: &http.Client{Timeout: filterSetReporting.Timeout},
			signer: sgn,
		}
	}
	return api, nil
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

func NewStack(
	stackConfig *node.Config,
	queueClient sqsclient.QueueClient,
	filterSetReporting *genericconf.HTTPClientConfig,
	sgn *signer.Signer,
) (*node.Node, error) {
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, err
	}

	api, err := NewFilteringReportAPI(queueClient, filterSetReporting, sgn)
	if err != nil {
		return nil, err
	}

	apis := []rpc.API{{
		Namespace: gethexec.FilteringReportNamespace,
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

	return stack, nil
}

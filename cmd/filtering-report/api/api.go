// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/execution/gethexec/addressfilter"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

type FilteringReportAPI struct {
	sqsClient *sqsclient.QueueClient
}

func NewFilteringReportAPI(sqsClient *sqsclient.QueueClient) *FilteringReportAPI {
	return &FilteringReportAPI{
		sqsClient: sqsClient,
	}
}

func (a *FilteringReportAPI) ReportFilteredTransactions(ctx context.Context, reports []addressfilter.FilteredTxReport) error {
	if a.sqsClient == nil {
		return errors.New("SQS client not configured")
	}
	log.Debug("Sending filtered transaction reports to SQS", "count", len(reports))
	for _, report := range reports {
		body, err := json.Marshal(report)
		if err != nil {
			return err
		}
		bodyStr := string(body)
		_, err = a.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    &a.sqsClient.QueueURL,
			MessageBody: &bodyStr,
		})
		if err != nil {
			log.Error("Failed to send filtered transaction report to SQS", "txHash", report.TxHash.Hex(), "err", err)
			return err
		}
		log.Debug("Successfully sent filtered transaction report to SQS", "txHash", report.TxHash.Hex())
	}
	return nil
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
	sqsClient *sqsclient.QueueClient,
) (*node.Node, *FilteringReportAPI, error) {
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, nil, err
	}

	api := NewFilteringReportAPI(sqsClient)

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

	return stack, api, nil
}

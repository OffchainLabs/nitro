// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package broadcastclients

import (
	"context"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/util/contracts"
)

type BroadcastClients struct {
	clients []*broadcastclient.BroadcastClient

	// Use atomic access
	connected int32
}

func NewBroadcastClients(
	configFetcher broadcastclient.ConfigFetcher,
	l2ChainId uint64,
	currentMessageCount arbutil.MessageIndex,
	txStreamer broadcastclient.TransactionStreamerInterface,
	confirmedSequenceNumberListener chan arbutil.MessageIndex,
	fatalErrChan chan error,
	bpVerifier contracts.BatchPosterVerifierInterface,
) (*BroadcastClients, error) {
	config := configFetcher()
	urlCount := len(config.URLs)
	if urlCount <= 0 {
		return nil, nil
	}

	clients := BroadcastClients{}
	clients.clients = make([]*broadcastclient.BroadcastClient, 0, urlCount)
	var lastClientErr error
	for _, address := range config.URLs {
		client, err := broadcastclient.NewBroadcastClient(
			configFetcher,
			address,
			l2ChainId,
			currentMessageCount,
			txStreamer,
			confirmedSequenceNumberListener,
			fatalErrChan,
			bpVerifier,
			func(delta int32) { clients.adjustCount(delta) },
		)
		if err != nil {
			lastClientErr = err
			log.Warn("init broadcast client failed", "address", address)
		}
		clients.clients = append(clients.clients, client)
	}
	if len(clients.clients) == 0 {
		log.Error("no connected feed on startup, last error: %w", lastClientErr)
	}

	return &clients, nil
}

func (bcs *BroadcastClients) adjustCount(delta int32) {
	connected := atomic.AddInt32(&bcs.connected, delta)
	if connected <= 0 {
		log.Error("no connected feed")
	}
}

func (bcs *BroadcastClients) Start(ctx context.Context) {
	for _, client := range bcs.clients {
		client.Start(ctx)
	}
}
func (bcs *BroadcastClients) StopAndWait() {
	for _, client := range bcs.clients {
		client.StopAndWait()
	}
}

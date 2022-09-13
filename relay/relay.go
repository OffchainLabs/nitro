// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package relay

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Relay struct {
	stopwaiter.StopWaiter
	broadcastClients            []*broadcastclient.BroadcastClient
	broadcaster                 *broadcaster.Broadcaster
	confirmedSequenceNumberChan chan arbutil.MessageIndex
	messageChan                 chan broadcaster.BroadcastFeedMessage
}

type RelayMessageQueue struct {
	queue chan broadcaster.BroadcastFeedMessage
}

func (q *RelayMessageQueue) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		q.queue <- *feedMessage
	}

	return nil
}

func NewRelay(feedConfig broadcastclient.FeedConfig, chainId uint64, feedErrChan chan error) (*Relay, error) {
	var broadcastClients []*broadcastclient.BroadcastClient

	q := RelayMessageQueue{make(chan broadcaster.BroadcastFeedMessage, 100)}

	confirmedSequenceNumberListener := make(chan arbutil.MessageIndex, 10)

	var lastClientError error
	for _, address := range feedConfig.Input.URLs {
		client, err := broadcastclient.NewBroadcastClient(feedConfig.Input, address, chainId, 0, &q, feedErrChan, nil)
		if err != nil {
			lastClientError = err
			log.Warn("init broadcast client failed", "address", address, "err", err)
			continue
		}
		client.ConfirmedSequenceNumberListener = confirmedSequenceNumberListener
		broadcastClients = append(broadcastClients, client)
	}
	if len(broadcastClients) == 0 && len(feedConfig.Input.URLs) > 0 {
		return nil, fmt.Errorf("no broadcast clients initialized. Last error: %w", lastClientError)
	}

	dataSignerErr := func([]byte) ([]byte, error) {
		return nil, errors.New("relay attempted to sign feed message")
	}
	return &Relay{
		broadcaster:                 broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &feedConfig.Output }, chainId, feedErrChan, dataSignerErr),
		broadcastClients:            broadcastClients,
		confirmedSequenceNumberChan: confirmedSequenceNumberListener,
		messageChan:                 q.queue,
	}, nil
}

const RECENT_FEED_ITEM_TTL time.Duration = time.Second * 10

func (r *Relay) Start(ctx context.Context) error {
	r.StopWaiter.Start(ctx)
	err := r.broadcaster.Initialize()
	if err != nil {
		return errors.New("broadcast unable to initialize")
	}
	err = r.broadcaster.Start(ctx)
	if err != nil {
		return errors.New("broadcast unable to start")
	}

	for _, client := range r.broadcastClients {
		client.Start(ctx)
	}

	recentFeedItems := make(map[arbutil.MessageIndex]time.Time)
	r.LaunchThread(func(ctx context.Context) {
		recentFeedItemsCleanup := time.NewTicker(RECENT_FEED_ITEM_TTL)
		defer recentFeedItemsCleanup.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-r.messageChan:
				if recentFeedItems[msg.SequenceNumber] != (time.Time{}) {
					continue
				}
				recentFeedItems[msg.SequenceNumber] = time.Now()
				r.broadcaster.BroadcastSingleFeedMessage(&msg)
			case cs := <-r.confirmedSequenceNumberChan:
				r.broadcaster.Confirm(cs)
			case <-recentFeedItemsCleanup.C:
				// Clear expired items from recentFeedItems
				recentFeedItemExpiry := time.Now().Add(-RECENT_FEED_ITEM_TTL)
				for acc, created := range recentFeedItems {
					if created.Before(recentFeedItemExpiry) {
						delete(recentFeedItems, acc)
					}
				}
			}
		}
	})

	return nil
}

func (r *Relay) GetListenerAddr() net.Addr {
	return r.broadcaster.ListenerAddr()
}

func (r *Relay) StopAndWait() {
	r.StopWaiter.StopAndWait()
	for _, client := range r.broadcastClients {
		client.StopAndWait()
	}
	r.broadcaster.StopAndWait()
}

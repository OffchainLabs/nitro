// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package relay

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
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
	messageChan                 chan broadcastFeedMessage
}

type broadcastFeedMessage struct {
	message        arbstate.MessageWithMetadata
	sequenceNumber arbutil.MessageIndex
}

type RelayMessageQueue struct {
	queue chan broadcastFeedMessage
}

func (q *RelayMessageQueue) AddBroadcastMessages(pos arbutil.MessageIndex, messages []arbstate.MessageWithMetadata) error {
	for i, message := range messages {
		q.queue <- broadcastFeedMessage{
			sequenceNumber: pos + arbutil.MessageIndex(i),
			message:        message,
		}
	}

	return nil
}

func NewRelay(serverConf wsbroadcastserver.BroadcasterConfig, clientConf broadcastclient.BroadcastClientConfig) *Relay {
	var broadcastClients []*broadcastclient.BroadcastClient

	q := RelayMessageQueue{make(chan broadcastFeedMessage, 100)}

	confirmedSequenceNumberListener := make(chan arbutil.MessageIndex, 10)

	for _, address := range clientConf.URLs {
		client := broadcastclient.NewBroadcastClient(address, nil, clientConf.Timeout, &q)
		client.ConfirmedSequenceNumberListener = confirmedSequenceNumberListener
		broadcastClients = append(broadcastClients, client)
	}

	return &Relay{
		broadcaster:                 broadcaster.NewBroadcaster(serverConf),
		broadcastClients:            broadcastClients,
		confirmedSequenceNumberChan: confirmedSequenceNumberListener,
		messageChan:                 q.queue,
	}
}

const RECENT_FEED_ITEM_TTL time.Duration = time.Second * 10

func (r *Relay) Start(ctx context.Context) error {
	r.StopWaiter.Start(ctx)
	err := r.broadcaster.Start(ctx)
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
				if recentFeedItems[msg.sequenceNumber] != (time.Time{}) {
					continue
				}
				recentFeedItems[msg.sequenceNumber] = time.Now()
				r.broadcaster.BroadcastSingle(msg.message, msg.sequenceNumber)
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

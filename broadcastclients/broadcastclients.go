// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcastclients

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const MAX_FEED_INACTIVE_TIME = time.Second * 6

type Router struct {
	stopwaiter.StopWaiter
	messageChan                 chan broadcaster.BroadcastFeedMessage
	confirmedSequenceNumberChan chan arbutil.MessageIndex

	forwardTxStreamer       broadcastclient.TransactionStreamerInterface
	forwardConfirmationChan chan arbutil.MessageIndex
}

func (r *Router) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		r.messageChan <- *feedMessage
	}
	return nil
}

type BroadcastClients struct {
	primaryClients        []*broadcastclient.BroadcastClient
	secondaryClients      []*broadcastclient.BroadcastClient
	numOfStartedSecondary int

	router *Router

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
	addrVerifier contracts.AddressVerifierInterface,
	queueCapcity int,
) (*BroadcastClients, error) {
	config := configFetcher()
	if len(config.URL) == 0 && len(config.SecondaryURL) == 0 {
		return nil, nil
	}

	clients := BroadcastClients{
		router: &Router{
			messageChan:                 make(chan broadcaster.BroadcastFeedMessage, queueCapcity),
			confirmedSequenceNumberChan: make(chan arbutil.MessageIndex, queueCapcity),
			forwardTxStreamer:           txStreamer,
			forwardConfirmationChan:     confirmedSequenceNumberListener,
		},
	}
	var lastClientErr error
	makeFeeds := func(url []string) []*broadcastclient.BroadcastClient {
		feeds := make([]*broadcastclient.BroadcastClient, 0, len(url))
		for _, address := range url {
			client, err := broadcastclient.NewBroadcastClient(
				configFetcher,
				address,
				l2ChainId,
				currentMessageCount,
				clients.router,
				clients.router.confirmedSequenceNumberChan,
				fatalErrChan,
				addrVerifier,
				func(delta int32) { clients.adjustCount(delta) },
			)
			if err != nil {
				lastClientErr = err
				log.Warn("init broadcast client failed", "address", address)
				continue
			}
			feeds = append(feeds, client)
		}
		return feeds
	}

	clients.primaryClients = makeFeeds(config.URL)
	clients.secondaryClients = makeFeeds(config.SecondaryURL)

	if len(clients.primaryClients) == 0 && len(clients.secondaryClients) == 0 {
		log.Error("no connected feed on startup, last error: %w", lastClientErr)
		return nil, nil
	}

	// have atleast one primary client
	if len(clients.primaryClients) == 0 {
		clients.primaryClients = append(clients.primaryClients, clients.secondaryClients[0])
		clients.secondaryClients = clients.secondaryClients[1:]
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
	bcs.router.StopWaiter.Start(ctx, bcs.router)

	for _, client := range bcs.primaryClients {
		client.Start(ctx)
	}

	bcs.router.LaunchThread(func(ctx context.Context) {
		startNewFeedTimer := time.NewTicker(MAX_FEED_INACTIVE_TIME)
		defer startNewFeedTimer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case cs := <-bcs.router.confirmedSequenceNumberChan:
				startNewFeedTimer.Stop()
				bcs.router.forwardConfirmationChan <- cs
				startNewFeedTimer.Reset(MAX_FEED_INACTIVE_TIME)
			case msg := <-bcs.router.messageChan:
				startNewFeedTimer.Stop()
				if err := bcs.router.forwardTxStreamer.AddBroadcastMessages([]*broadcaster.BroadcastFeedMessage{&msg}); err != nil {
					log.Error("Error routing message from Sequencer Feed", "err", err)
				}
				startNewFeedTimer.Reset(MAX_FEED_INACTIVE_TIME)
			case <-startNewFeedTimer.C:
				// failed to get messages from primary feed for ~5 seconds, start a new feed
				bcs.StartSecondaryFeed(ctx)
			}
		}
	})
}

func (bcs *BroadcastClients) StartSecondaryFeed(ctx context.Context) {
	if bcs.numOfStartedSecondary < len(bcs.secondaryClients) {
		client := bcs.secondaryClients[bcs.numOfStartedSecondary]
		bcs.numOfStartedSecondary += 1
		client.Start(ctx)
	} else {
		log.Warn("failed to start a new secondary feed all available secondary feeds were started")
	}
}

func (bcs *BroadcastClients) StopAndWait() {
	for _, client := range bcs.primaryClients {
		client.StopAndWait()
	}
	for i := 0; i < bcs.numOfStartedSecondary; i++ {
		bcs.secondaryClients[i].StopAndWait()
	}
}

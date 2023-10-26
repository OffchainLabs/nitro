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

const ROUTER_QUEUE_SIZE = 1024
const RECENT_FEED_INITIAL_MAP_SIZE = 1024
const RECENT_FEED_ITEM_TTL = time.Second * 10
const MAX_FEED_INACTIVE_TIME = time.Second * 5
const PRIMARY_FEED_UPTIME = time.Minute * 10

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

	primaryRouter   *Router
	secondaryRouter *Router

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
) (*BroadcastClients, error) {
	config := configFetcher()
	if len(config.URL) == 0 && len(config.SecondaryURL) == 0 {
		return nil, nil
	}
	newStandardRouter := func() *Router {
		return &Router{
			messageChan:                 make(chan broadcaster.BroadcastFeedMessage, ROUTER_QUEUE_SIZE),
			confirmedSequenceNumberChan: make(chan arbutil.MessageIndex, ROUTER_QUEUE_SIZE),
			forwardTxStreamer:           txStreamer,
			forwardConfirmationChan:     confirmedSequenceNumberListener,
		}
	}
	clients := BroadcastClients{
		primaryRouter:   newStandardRouter(),
		secondaryRouter: newStandardRouter(),
	}
	var lastClientErr error
	makeFeeds := func(url []string, router *Router) []*broadcastclient.BroadcastClient {
		feeds := make([]*broadcastclient.BroadcastClient, 0, len(url))
		for _, address := range url {
			client, err := broadcastclient.NewBroadcastClient(
				configFetcher,
				address,
				l2ChainId,
				currentMessageCount,
				router,
				router.confirmedSequenceNumberChan,
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

	clients.primaryClients = makeFeeds(config.URL, clients.primaryRouter)
	clients.secondaryClients = makeFeeds(config.SecondaryURL, clients.secondaryRouter)

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
	bcs.primaryRouter.StopWaiter.Start(ctx, bcs.primaryRouter)
	bcs.secondaryRouter.StopWaiter.Start(ctx, bcs.secondaryRouter)

	for _, client := range bcs.primaryClients {
		client.Start(ctx)
	}

	var lastConfirmed arbutil.MessageIndex
	recentFeedItemsNew := make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)
	recentFeedItemsOld := make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)
	bcs.primaryRouter.LaunchThread(func(ctx context.Context) {
		recentFeedItemsCleanup := time.NewTicker(RECENT_FEED_ITEM_TTL)
		startSecondaryFeedTimer := time.NewTicker(MAX_FEED_INACTIVE_TIME)
		stopSecondaryFeedTimer := time.NewTicker(PRIMARY_FEED_UPTIME)
		primaryFeedIsDownTimer := time.NewTicker(MAX_FEED_INACTIVE_TIME)
		defer recentFeedItemsCleanup.Stop()
		defer startSecondaryFeedTimer.Stop()
		defer stopSecondaryFeedTimer.Stop()
		defer primaryFeedIsDownTimer.Stop()
		for {
			select {
			case <-ctx.Done():
				return

			// Primary feeds
			case msg := <-bcs.primaryRouter.messageChan:
				startSecondaryFeedTimer.Reset(MAX_FEED_INACTIVE_TIME)
				primaryFeedIsDownTimer.Reset(MAX_FEED_INACTIVE_TIME)
				if _, ok := recentFeedItemsNew[msg.SequenceNumber]; ok {
					continue
				}
				if _, ok := recentFeedItemsOld[msg.SequenceNumber]; ok {
					continue
				}
				recentFeedItemsNew[msg.SequenceNumber] = time.Now()
				if err := bcs.primaryRouter.forwardTxStreamer.AddBroadcastMessages([]*broadcaster.BroadcastFeedMessage{&msg}); err != nil {
					log.Error("Error routing message from Primary Sequencer Feeds", "err", err)
				}
			case cs := <-bcs.primaryRouter.confirmedSequenceNumberChan:
				startSecondaryFeedTimer.Reset(MAX_FEED_INACTIVE_TIME)
				primaryFeedIsDownTimer.Reset(MAX_FEED_INACTIVE_TIME)
				if cs == lastConfirmed {
					continue
				}
				lastConfirmed = cs
				bcs.primaryRouter.forwardConfirmationChan <- cs

			// Secondary Feeds
			case msg := <-bcs.secondaryRouter.messageChan:
				startSecondaryFeedTimer.Reset(MAX_FEED_INACTIVE_TIME)
				if _, ok := recentFeedItemsNew[msg.SequenceNumber]; ok {
					continue
				}
				if _, ok := recentFeedItemsOld[msg.SequenceNumber]; ok {
					continue
				}
				recentFeedItemsNew[msg.SequenceNumber] = time.Now()
				if err := bcs.secondaryRouter.forwardTxStreamer.AddBroadcastMessages([]*broadcaster.BroadcastFeedMessage{&msg}); err != nil {
					log.Error("Error routing message from Secondary Sequencer Feeds", "err", err)
				}
			case cs := <-bcs.secondaryRouter.confirmedSequenceNumberChan:
				startSecondaryFeedTimer.Reset(MAX_FEED_INACTIVE_TIME)
				if cs == lastConfirmed {
					continue
				}
				lastConfirmed = cs
				bcs.secondaryRouter.forwardConfirmationChan <- cs

			// Cycle buckets to get rid of old entries
			case <-recentFeedItemsCleanup.C:
				recentFeedItemsOld = recentFeedItemsNew
				recentFeedItemsNew = make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)

			// failed to get messages from both primary and secondary feeds for ~5 seconds, start a new secondary feed
			case <-startSecondaryFeedTimer.C:
				bcs.startSecondaryFeed(ctx)

			// failed to get messages from primary feed for ~5 seconds, reset the timer responsible for stopping a secondary
			case <-primaryFeedIsDownTimer.C:
				stopSecondaryFeedTimer.Reset(PRIMARY_FEED_UPTIME)

			// primary feeds have been up and running for PRIMARY_FEED_UPTIME=10 mins without a failure, stop the recently started secondary feed
			case <-stopSecondaryFeedTimer.C:
				bcs.stopSecondaryFeed(ctx)
			}
		}
	})
}

func (bcs *BroadcastClients) startSecondaryFeed(ctx context.Context) {
	if bcs.numOfStartedSecondary < len(bcs.secondaryClients) {
		client := bcs.secondaryClients[bcs.numOfStartedSecondary]
		bcs.numOfStartedSecondary += 1
		client.Start(ctx)
	} else {
		log.Warn("failed to start a new secondary feed all available secondary feeds were started")
	}
}
func (bcs *BroadcastClients) stopSecondaryFeed(ctx context.Context) {
	if bcs.numOfStartedSecondary > 0 {
		bcs.numOfStartedSecondary -= 1
		client := bcs.secondaryClients[bcs.numOfStartedSecondary]
		client.StopAndWait()
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

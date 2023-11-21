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
	primaryClients   []*broadcastclient.BroadcastClient
	secondaryClients []*broadcastclient.BroadcastClient
	secondaryURL     []string

	primaryRouter   *Router
	secondaryRouter *Router

	// Use atomic access
	connected int32
}

var makeClient func(string, *Router) (*broadcastclient.BroadcastClient, error)

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
		primaryRouter:    newStandardRouter(),
		secondaryRouter:  newStandardRouter(),
		primaryClients:   make([]*broadcastclient.BroadcastClient, 0, len(config.URL)),
		secondaryClients: make([]*broadcastclient.BroadcastClient, 0, len(config.SecondaryURL)),
		secondaryURL:     config.SecondaryURL,
	}
	makeClient = func(url string, router *Router) (*broadcastclient.BroadcastClient, error) {
		return broadcastclient.NewBroadcastClient(
			configFetcher,
			url,
			l2ChainId,
			currentMessageCount,
			router,
			router.confirmedSequenceNumberChan,
			fatalErrChan,
			addrVerifier,
			func(delta int32) { clients.adjustCount(delta) },
		)
	}

	var lastClientErr error
	for _, address := range config.URL {
		client, err := makeClient(address, clients.primaryRouter)
		if err != nil {
			lastClientErr = err
			log.Warn("init broadcast client failed", "address", address)
			continue
		}
		clients.primaryClients = append(clients.primaryClients, client)
	}
	if len(clients.primaryClients) == 0 {
		log.Error("no connected feed on startup, last error: %w", lastClientErr)
		return nil, nil
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
				if bcs.primaryRouter.forwardConfirmationChan != nil {
					bcs.primaryRouter.forwardConfirmationChan <- cs
				}

			// Cycle buckets to get rid of old entries
			case <-recentFeedItemsCleanup.C:
				recentFeedItemsOld = recentFeedItemsNew
				recentFeedItemsNew = make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)

			// Failed to get messages from both primary and secondary feeds for ~5 seconds, start a new secondary feed
			case <-startSecondaryFeedTimer.C:
				bcs.startSecondaryFeed(ctx)

			// Failed to get messages from primary feed for ~5 seconds, reset the timer responsible for stopping a secondary
			case <-primaryFeedIsDownTimer.C:
				stopSecondaryFeedTimer.Reset(PRIMARY_FEED_UPTIME)

			// Primary feeds have been up and running for PRIMARY_FEED_UPTIME=10 mins without a failure, stop the recently started secondary feed
			case <-stopSecondaryFeedTimer.C:
				bcs.stopSecondaryFeed()

			default:
				select {
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
					if bcs.secondaryRouter.forwardConfirmationChan != nil {
						bcs.secondaryRouter.forwardConfirmationChan <- cs
					}
				default:
				}
			}
		}
	})
}

func (bcs *BroadcastClients) startSecondaryFeed(ctx context.Context) {
	pos := len(bcs.secondaryClients)
	if pos < len(bcs.secondaryURL) {
		url := bcs.secondaryURL[pos]
		client, err := makeClient(url, bcs.secondaryRouter)
		if err != nil {
			log.Warn("init broadcast secondary client failed", "address", url)
			bcs.secondaryURL = append(bcs.secondaryURL[:pos], bcs.secondaryURL[pos+1:]...)
			return
		}
		bcs.secondaryClients = append(bcs.secondaryClients, client)
		client.Start(ctx)
		log.Info("secondary feed started", "url", url)
	} else if len(bcs.secondaryURL) > 0 {
		log.Warn("failed to start a new secondary feed all available secondary feeds were started")
	}
}

func (bcs *BroadcastClients) stopSecondaryFeed() {
	pos := len(bcs.secondaryClients)
	if pos > 0 {
		pos -= 1
		bcs.secondaryClients[pos].StopAndWait()
		bcs.secondaryClients = bcs.secondaryClients[:pos]
		log.Info("disconnected secondary feed", "url", bcs.secondaryURL[pos])
	}
}

func (bcs *BroadcastClients) StopAndWait() {
	for _, client := range bcs.primaryClients {
		client.StopAndWait()
	}
	for _, client := range bcs.secondaryClients {
		client.StopAndWait()
	}
}

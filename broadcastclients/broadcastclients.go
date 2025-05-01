// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcastclients

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	m "github.com/offchainlabs/nitro/broadcaster/message"
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
	messageChan                 chan m.BroadcastFeedMessage
	confirmedSequenceNumberChan chan arbutil.MessageIndex

	forwardTxStreamer       broadcastclient.TransactionStreamerInterface
	forwardConfirmationChan chan arbutil.MessageIndex
}

func (r *Router) AddBroadcastMessages(feedMessages []*m.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		r.messageChan <- *feedMessage
	}
	return nil
}

type BroadcastClients struct {
	primaryClients   []*broadcastclient.BroadcastClient
	secondaryClients []*broadcastclient.BroadcastClient
	secondaryURL     []string
	makeClient       func(string, *Router) (*broadcastclient.BroadcastClient, error)

	primaryRouter   *Router
	secondaryRouter *Router

	// Use atomic access
	connected atomic.Int32
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
			messageChan:                 make(chan m.BroadcastFeedMessage, ROUTER_QUEUE_SIZE),
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
	clients.makeClient = func(url string, router *Router) (*broadcastclient.BroadcastClient, error) {
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
		client, err := clients.makeClient(address, clients.primaryRouter)
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
	connected := bcs.connected.Add(delta)
	if connected <= 0 {
		log.Error("no connected feed")
	}
}

// Clears out a ticker's channel and resets it to the interval
func clearAndResetTicker(timer *time.Ticker, interval time.Duration) {
	timer.Stop()
	// Clear out any previous ticks
	// A ticker's channel is only buffers one tick, so we don't need a loop here
	select {
	case <-timer.C:
	default:
	}
	timer.Reset(interval)
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

		msgHandler := func(msg m.BroadcastFeedMessage, router *Router) error {
			if _, ok := recentFeedItemsNew[msg.SequenceNumber]; ok {
				return nil
			}
			if _, ok := recentFeedItemsOld[msg.SequenceNumber]; ok {
				return nil
			}
			recentFeedItemsNew[msg.SequenceNumber] = time.Now()
			if err := router.forwardTxStreamer.AddBroadcastMessages([]*m.BroadcastFeedMessage{&msg}); err != nil {
				return err
			}
			return nil
		}
		confSeqHandler := func(cs arbutil.MessageIndex, router *Router) {
			if cs == lastConfirmed {
				return
			}
			lastConfirmed = cs
			if router.forwardConfirmationChan != nil {
				router.forwardConfirmationChan <- cs
			}
		}

		// Multiple select statements to prioritize reading messages from primary feeds' channels and avoid starving of timers
		for {
			select {
			// Cycle buckets to get rid of old entries
			case <-recentFeedItemsCleanup.C:
				recentFeedItemsOld = recentFeedItemsNew
				recentFeedItemsNew = make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)
			// Primary feeds have been up and running for PRIMARY_FEED_UPTIME=10 mins without a failure, stop the recently started secondary feed
			case <-stopSecondaryFeedTimer.C:
				bcs.stopSecondaryFeed()
			default:
			}

			select {
			case <-ctx.Done():
				return
			// Primary feeds
			case msg := <-bcs.primaryRouter.messageChan:
				if err := msgHandler(msg, bcs.primaryRouter); err != nil {
					if errors.Is(err, broadcastclient.TransactionStreamerBlockCreationStopped) {
						log.Info("stopping block creation in broadcast clients because transaction streamer has stopped")
						return
					}
					log.Error("Error routing message from Primary Sequencer Feeds", "err", err)
				}
				clearAndResetTicker(startSecondaryFeedTimer, MAX_FEED_INACTIVE_TIME)
				clearAndResetTicker(primaryFeedIsDownTimer, MAX_FEED_INACTIVE_TIME)
			case cs := <-bcs.primaryRouter.confirmedSequenceNumberChan:
				confSeqHandler(cs, bcs.primaryRouter)
				clearAndResetTicker(startSecondaryFeedTimer, MAX_FEED_INACTIVE_TIME)
				clearAndResetTicker(primaryFeedIsDownTimer, MAX_FEED_INACTIVE_TIME)
			// Failed to get messages from primary feed for ~5 seconds, reset the timer responsible for stopping a secondary
			case <-primaryFeedIsDownTimer.C:
				clearAndResetTicker(stopSecondaryFeedTimer, PRIMARY_FEED_UPTIME)
			default:
				select {
				case <-ctx.Done():
					return
				// Secondary Feeds
				case msg := <-bcs.secondaryRouter.messageChan:
					if err := msgHandler(msg, bcs.secondaryRouter); err != nil {
						log.Error("Error routing message from Secondary Sequencer Feeds", "err", err)
					}
					clearAndResetTicker(startSecondaryFeedTimer, MAX_FEED_INACTIVE_TIME)
				case cs := <-bcs.secondaryRouter.confirmedSequenceNumberChan:
					confSeqHandler(cs, bcs.secondaryRouter)
					clearAndResetTicker(startSecondaryFeedTimer, MAX_FEED_INACTIVE_TIME)
				case msg := <-bcs.primaryRouter.messageChan:
					if err := msgHandler(msg, bcs.primaryRouter); err != nil {
						log.Error("Error routing message from Primary Sequencer Feeds", "err", err)
					}
					clearAndResetTicker(startSecondaryFeedTimer, MAX_FEED_INACTIVE_TIME)
					clearAndResetTicker(primaryFeedIsDownTimer, MAX_FEED_INACTIVE_TIME)
				case cs := <-bcs.primaryRouter.confirmedSequenceNumberChan:
					confSeqHandler(cs, bcs.primaryRouter)
					clearAndResetTicker(startSecondaryFeedTimer, MAX_FEED_INACTIVE_TIME)
					clearAndResetTicker(primaryFeedIsDownTimer, MAX_FEED_INACTIVE_TIME)
				case <-startSecondaryFeedTimer.C:
					bcs.startSecondaryFeed(ctx)
				case <-primaryFeedIsDownTimer.C:
					clearAndResetTicker(stopSecondaryFeedTimer, PRIMARY_FEED_UPTIME)
				}
			}
		}
	})
}

func (bcs *BroadcastClients) startSecondaryFeed(ctx context.Context) {
	pos := len(bcs.secondaryClients)
	if pos < len(bcs.secondaryURL) {
		url := bcs.secondaryURL[pos]
		client, err := bcs.makeClient(url, bcs.secondaryRouter)
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

		// flush the secondary feeds' message and confirmedSequenceNumber channels
		for {
			select {
			case <-bcs.secondaryRouter.messageChan:
			case <-bcs.secondaryRouter.confirmedSequenceNumberChan:
			default:
				return
			}
		}
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

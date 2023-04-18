// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package relay

import (
	"context"
	"errors"
	"net"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcastclients"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/util/sharedmetrics"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type Relay struct {
	stopwaiter.StopWaiter
	broadcastClients            *broadcastclients.BroadcastClients
	broadcaster                 *broadcaster.Broadcaster
	confirmedSequenceNumberChan chan arbutil.MessageIndex
	messageChan                 chan broadcaster.BroadcastFeedMessage
}

type MessageQueue struct {
	queue chan broadcaster.BroadcastFeedMessage
}

func (q *MessageQueue) AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error {
	for _, feedMessage := range feedMessages {
		q.queue <- *feedMessage
	}

	return nil
}

func NewRelay(config *Config, feedErrChan chan error) (*Relay, error) {

	q := MessageQueue{make(chan broadcaster.BroadcastFeedMessage, config.Queue)}

	confirmedSequenceNumberListener := make(chan arbutil.MessageIndex, config.Queue)

	clients, err := broadcastclients.NewBroadcastClients(
		func() *broadcastclient.Config { return &config.Node.Feed.Input },
		config.L2.ChainId,
		0,
		&q,
		confirmedSequenceNumberListener,
		feedErrChan,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if clients == nil {
		return nil, errors.New("no feed servers found")
	}

	dataSignerErr := func([]byte) ([]byte, error) {
		return nil, errors.New("relay attempted to sign feed message")
	}
	return &Relay{
		broadcaster:                 broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config.Node.Feed.Output }, config.L2.ChainId, feedErrChan, dataSignerErr),
		broadcastClients:            clients,
		confirmedSequenceNumberChan: confirmedSequenceNumberListener,
		messageChan:                 q.queue,
	}, nil
}

const RECENT_FEED_ITEM_TTL = time.Second * 10
const RECENT_FEED_INITIAL_MAP_SIZE = 1024

func (r *Relay) Start(ctx context.Context) error {
	r.StopWaiter.Start(ctx, r)
	err := r.broadcaster.Initialize()
	if err != nil {
		return errors.New("broadcast unable to initialize")
	}
	err = r.broadcaster.Start(ctx)
	if err != nil {
		return errors.New("broadcast unable to start")
	}

	r.broadcastClients.Start(ctx)

	var lastConfirmed arbutil.MessageIndex
	recentFeedItemsNew := make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)
	recentFeedItemsOld := make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)
	r.LaunchThread(func(ctx context.Context) {
		recentFeedItemsCleanup := time.NewTicker(RECENT_FEED_ITEM_TTL)
		defer recentFeedItemsCleanup.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-r.messageChan:
				if _, ok := recentFeedItemsNew[msg.SequenceNumber]; ok {
					continue
				}
				if _, ok := recentFeedItemsOld[msg.SequenceNumber]; ok {
					continue
				}
				recentFeedItemsNew[msg.SequenceNumber] = time.Now()
				sharedmetrics.UpdateSequenceNumberGauge(msg.SequenceNumber)
				r.broadcaster.BroadcastSingleFeedMessage(&msg)
			case cs := <-r.confirmedSequenceNumberChan:
				if lastConfirmed == cs {
					continue
				}
				r.broadcaster.Confirm(cs)
			case <-recentFeedItemsCleanup.C:
				// Cycle buckets to get rid of old entries
				recentFeedItemsOld = recentFeedItemsNew
				recentFeedItemsNew = make(map[arbutil.MessageIndex]time.Time, RECENT_FEED_INITIAL_MAP_SIZE)
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
	r.broadcastClients.StopAndWait()
	r.broadcaster.StopAndWait()
}

type Config struct {
	Conf          genericconf.ConfConfig          `koanf:"conf"`
	L2            L2Config                        `koanf:"l2"`
	LogLevel      int                             `koanf:"log-level"`
	LogType       string                          `koanf:"log-type"`
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	Node          NodeConfig                      `koanf:"node"`
	Queue         int                             `koanf:"queue"`
}

var ConfigDefault = Config{
	Conf:          genericconf.ConfConfigDefault,
	L2:            L2ConfigDefault,
	LogLevel:      int(log.LvlInfo),
	LogType:       "plaintext",
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
	Node:          NodeConfigDefault,
	Queue:         1024,
}

func ConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	L2ConfigAddOptions("l2", f)
	f.Int("log-level", ConfigDefault.LogLevel, "log level")
	f.String("log-type", ConfigDefault.LogType, "log type")
	f.Bool("metrics", ConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	NodeConfigAddOptions("node", f)
	f.Int("queue", ConfigDefault.Queue, "size of relay queue")
}

type NodeConfig struct {
	Feed broadcastclient.FeedConfig `koanf:"feed"`
}

var NodeConfigDefault = NodeConfig{
	Feed: broadcastclient.FeedConfigDefault,
}

func NodeConfigAddOptions(prefix string, f *flag.FlagSet) {
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, true, true)
}

type L2Config struct {
	ChainId          uint64   `koanf:"chain-id"`
	ChainConfigFiles []string `koanf:"chain-config-files"`
}

var L2ConfigDefault = L2Config{
	ChainId:          0,
	ChainConfigFiles: []string{},
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L2ConfigDefault.ChainId, "L2 chain ID")
	f.StringSlice(prefix+".chain-config-files", L2ConfigDefault.ChainConfigFiles, "L2 chain config json files")
}

func ParseRelay(_ context.Context, args []string) (*Config, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	ConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var relayConfig Config
	if err := confighelpers.EndCommonParse(k, &relayConfig); err != nil {
		return nil, err
	}

	if relayConfig.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{})
		if err != nil {
			return nil, err
		}
	}

	return &relayConfig, nil
}

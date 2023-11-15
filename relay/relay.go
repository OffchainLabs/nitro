// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package relay

import (
	"context"
	"errors"
	"net"

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
		config.Chain.ID,
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
		broadcaster:                 broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config.Node.Feed.Output }, config.Chain.ID, feedErrChan, dataSignerErr),
		broadcastClients:            clients,
		confirmedSequenceNumberChan: confirmedSequenceNumberListener,
		messageChan:                 q.queue,
	}, nil
}

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

	r.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-r.messageChan:
				sharedmetrics.UpdateSequenceNumberGauge(msg.SequenceNumber)
				r.broadcaster.BroadcastSingleFeedMessage(&msg)
			case cs := <-r.confirmedSequenceNumberChan:
				r.broadcaster.Confirm(cs)
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
	Chain         L2Config                        `koanf:"chain"`
	LogLevel      int                             `koanf:"log-level"`
	LogType       string                          `koanf:"log-type"`
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
	Node          NodeConfig                      `koanf:"node"`
	Queue         int                             `koanf:"queue"`
}

var ConfigDefault = Config{
	Conf:          genericconf.ConfConfigDefault,
	Chain:         L2ConfigDefault,
	LogLevel:      int(log.LvlInfo),
	LogType:       "plaintext",
	Metrics:       false,
	MetricsServer: genericconf.MetricsServerConfigDefault,
	PProf:         false,
	PprofCfg:      genericconf.PProfDefault,
	Node:          NodeConfigDefault,
	Queue:         1024,
}

func ConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	L2ConfigAddOptions("chain", f)
	f.Int("log-level", ConfigDefault.LogLevel, "log level")
	f.String("log-type", ConfigDefault.LogType, "log type")
	f.Bool("metrics", ConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", ConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)
	NodeConfigAddOptions("node", f)
	f.Int("queue", ConfigDefault.Queue, "queue for incoming messages from sequencer")
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
	ID uint64 `koanf:"id"`
}

var L2ConfigDefault = L2Config{
	ID: 0,
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".id", L2ConfigDefault.ID, "L2 chain ID")
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

package relay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

type RelayBroadcastClient struct {
	stopwaiter.StopWaiter
	broadcaster *broadcaster.Broadcaster
}

func NewRelayBroadcastClient(config *Config, feedErrChan chan error) *RelayBroadcastClient {
	dataSignerErr := func([]byte) ([]byte, error) {
		return nil, errors.New("relay attempted to sign feed message")
	}
	return &RelayBroadcastClient{
		broadcaster: broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &config.Node.Feed.Output }, config.Chain.ID, feedErrChan, dataSignerErr),
	}
}

func (r *RelayBroadcastClient) Start(ctx context.Context) error {
	r.StopWaiter.Start(ctx, r)
	err := r.broadcaster.Initialize()
	if err != nil {
		return errors.New("broadcast unable to initialize")
	}
	err = r.broadcaster.Start(ctx)
	if err != nil {
		return errors.New("broadcast unable to start")
	}
	return nil
}

func (r *RelayBroadcastClient) PopulateFeedBacklogByNumber(ctx context.Context, backlogSize, l2MsgSize int) {
	was := r.broadcaster.GetCachedMessageCount()
	var seqNums []arbutil.MessageIndex
	for i := was; i < was+backlogSize; i++ {
		seqNums = append(seqNums, arbutil.MessageIndex(i))
	}

	messages := make([]*message.BroadcastFeedMessage, 0, len(seqNums))
	for _, seqNum := range seqNums {
		broadcastMessage := &message.BroadcastFeedMessage{
			SequenceNumber: seqNum,
			Message: arbostypes.MessageWithMetadata{
				Message: &arbostypes.L1IncomingMessage{
					L2msg: make([]byte, l2MsgSize),
				},
			},
		}
		messages = append(messages, broadcastMessage)
	}
	r.broadcaster.BroadcastFeedMessages(messages)
	waitForBacklog(r.broadcaster, was, was+backlogSize)
}

func waitForBacklog(b *broadcaster.Broadcaster, was, target int) {
	time.Sleep(time.Second)
	prevCount := was
	for count := b.GetCachedMessageCount(); count != target; count = b.GetCachedMessageCount() {
		if prevCount == count {
			log.Warn("unable to populate feed backlog. Cached message count did not increment")
			break
		} else {
			prevCount = count
		}
		log.Info("populating feed backlog to stress test relay", "current", count, "target", target)
		time.Sleep(5 * time.Second)
	}
}

type dummyTxStreamer struct {
	id            int
	logConnection bool
}

func (ts *dummyTxStreamer) AddBroadcastMessages(feedMessages []*message.BroadcastFeedMessage) error {
	// to mimic latency of txstreamer
	time.Sleep(50 * time.Millisecond)
	if !ts.logConnection {
		ts.logConnection = true
		log.Info("test client is succesfully receiving messages", "client_Id", ts.id, "msg_size", feedMessages[0].Size())
	}
	return nil
}

func largeBacklogRelayTestImpl(t *testing.T, numClients, backlogSize, l2MsgSize int, clientsRegDeadline time.Duration, relayBCPort, relayPort string) {
	// total size of the backlog = backlogSize * (l2MsgSize + 160)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	relayBroadcastClientConfig := &ConfigDefault
	relayBroadcastClientConfig.Node.Feed.Output.Addr = "127.0.0.1"
	relayBroadcastClientConfig.Node.Feed.Output.Port = relayBCPort
	relayBroadcastClientConfig.Node.Feed.Output.ClientTimeout = 5 * time.Minute
	relayBroadcastClient := NewRelayBroadcastClient(relayBroadcastClientConfig, nil)
	err := relayBroadcastClient.Start(ctx)
	if err != nil {
		t.Fatalf("error starting relay's broadcast client %v", err)
	}
	relayBroadcastClient.PopulateFeedBacklogByNumber(ctx, backlogSize, l2MsgSize)

	relayConfig := &ConfigDefault
	relayConfig.Node.Feed.Input.URL = []string{"ws://127.0.0.1:" + relayBCPort}
	relayConfig.Node.Feed.Output.Addr = "127.0.0.1"
	relayConfig.Node.Feed.Output.Port = relayPort
	relayConfig.Node.Feed.Output.ClientTimeout = 5 * time.Minute
	relay, err := NewRelay(relayConfig, nil)
	if err != nil {
		t.Fatalf("error initializing relay %v", err)
	}
	err = relay.Start(ctx)
	if err != nil {
		t.Fatalf("error starting relay %v", err)
	}
	waitForBacklog(relay.broadcaster, 0, backlogSize)

	relayURL := "ws://" + relay.GetListenerAddr().String()
	clientConfig := broadcastclient.DefaultTestConfig
	clientConfig.Timeout = 5 * time.Minute
	var streamers []*dummyTxStreamer
	for i := 0; i < numClients; i++ {
		ts := &dummyTxStreamer{id: i}
		streamers = append(streamers, ts)
		client, err := broadcastclient.NewBroadcastClient(func() *broadcastclient.Config { return &clientConfig }, relayURL, relayConfig.Chain.ID, 0, ts, nil, nil, nil, func(_ int32) {})
		if err != nil {
			t.FailNow()
		}
		client.Start(ctx)
	}

	// wait for all clients to atleast connect once
	time.Sleep(clientsRegDeadline)
	connected := 0
	for _, ts := range streamers {
		if ts.logConnection {
			connected++
		}
	}
	if int32(connected) != int32(numClients) {
		t.Fail()
	}
	log.Info("number of clients connected", "expected", numClients, "got", connected)
}

func TestRelayLargeBacklog16MB(t *testing.T) {
	t.Skip("This test is for manual inspection and would be unreliable in CI even if automated")
	largeBacklogRelayTestImpl(t, 150, 100000, 0, 40*time.Second, "9642", "9643")
}

func TestRelayLargeBacklog50MB(t *testing.T) {
	t.Skip("This test is for manual inspection and would be unreliable in CI even if automated")
	largeBacklogRelayTestImpl(t, 150, 100000, 340, 40*time.Second, "9644", "9645")
}

func TestRelayLargeBacklog100MB(t *testing.T) {
	t.Skip("This test is for manual inspection and would be unreliable in CI even if automated")
	largeBacklogRelayTestImpl(t, 150, 100000, 840, 40*time.Second, "9646", "9647")
}

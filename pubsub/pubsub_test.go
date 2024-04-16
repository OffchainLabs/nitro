package pubsub

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
)

var (
	consumersCount = 10
	messagesCount  = 100
)

type testRequest struct {
	Request string
}

type testResponse struct {
	Response string
}

func createGroup(ctx context.Context, t *testing.T, streamName, groupName string, client redis.UniversalClient) {
	t.Helper()
	if _, err := client.XGroupCreateMkStream(ctx, streamName, groupName, "$").Result(); err != nil {
		t.Fatalf("Error creating stream group: %v", err)
	}
}

func destroyGroup(ctx context.Context, t *testing.T, streamName, groupName string, client redis.UniversalClient) {
	t.Helper()
	if _, err := client.XGroupDestroy(ctx, streamName, groupName).Result(); err != nil {
		log.Debug("Error destroying a stream group", "error", err)
	}
}

type configOpt interface {
	apply(consCfg *ConsumerConfig, prodCfg *ProducerConfig)
}

type disableReproduce struct{}

func (e *disableReproduce) apply(_ *ConsumerConfig, prodCfg *ProducerConfig) {
	prodCfg.EnableReproduce = false
}

func producerCfg() *ProducerConfig {
	return &ProducerConfig{
		EnableReproduce:      DefaultTestProducerConfig.EnableReproduce,
		CheckPendingInterval: DefaultTestProducerConfig.CheckPendingInterval,
		KeepAliveTimeout:     DefaultTestProducerConfig.KeepAliveTimeout,
		CheckResultInterval:  DefaultTestProducerConfig.CheckResultInterval,
	}
}

func consumerCfg() *ConsumerConfig {
	return &ConsumerConfig{
		ResponseEntryTimeout: DefaultTestConsumerConfig.ResponseEntryTimeout,
		KeepAliveTimeout:     DefaultTestConsumerConfig.KeepAliveTimeout,
	}
}

func newProducerConsumers(ctx context.Context, t *testing.T, opts ...configOpt) (*Producer[testRequest, testResponse], []*Consumer[testRequest, testResponse]) {
	t.Helper()
	redisURL := redisutil.CreateTestRedis(ctx, t)
	prodCfg, consCfg := producerCfg(), consumerCfg()
	prodCfg.RedisURL, consCfg.RedisURL = redisURL, redisURL
	streamName := uuid.NewString()
	groupName := fmt.Sprintf("group_%s", streamName)
	prodCfg.RedisGroup, consCfg.RedisGroup = groupName, groupName
	prodCfg.RedisStream, consCfg.RedisStream = streamName, streamName
	for _, o := range opts {
		o.apply(consCfg, prodCfg)
	}
	producer, err := NewProducer[testRequest, testResponse](prodCfg)
	if err != nil {
		t.Fatalf("Error creating new producer: %v", err)
	}

	var consumers []*Consumer[testRequest, testResponse]
	for i := 0; i < consumersCount; i++ {
		c, err := NewConsumer[testRequest, testResponse](ctx, consCfg)
		if err != nil {
			t.Fatalf("Error creating new consumer: %v", err)
		}
		consumers = append(consumers, c)
	}
	createGroup(ctx, t, streamName, groupName, producer.client)
	t.Cleanup(func() {
		ctx := context.Background()
		destroyGroup(ctx, t, streamName, groupName, producer.client)
		var keys []string
		for _, c := range consumers {
			keys = append(keys, c.heartBeatKey())
		}
		if _, err := producer.client.Del(ctx, keys...).Result(); err != nil {
			log.Debug("Error deleting heartbeat keys", "error", err)
		}
	})
	return producer, consumers
}

func messagesMaps(n int) []map[string]string {
	ret := make([]map[string]string, n)
	for i := 0; i < n; i++ {
		ret[i] = make(map[string]string)
	}
	return ret
}

func wantMessages(n int) []string {
	var ret []string
	for i := 0; i < n; i++ {
		ret = append(ret, fmt.Sprintf("msg: %d", i))
	}
	sort.Strings(ret)
	return ret
}

func flatten(responses [][]string) []string {
	var ret []string
	for _, v := range responses {
		ret = append(ret, v...)
	}
	sort.Strings(ret)
	return ret
}

func produceMessages(ctx context.Context, msgs []string, producer *Producer[testRequest, testResponse]) ([]*containers.Promise[testResponse], error) {
	var promises []*containers.Promise[testResponse]
	for i := 0; i < messagesCount; i++ {
		promise, err := producer.Produce(ctx, testRequest{Request: msgs[i]})
		if err != nil {
			return nil, err
		}
		promises = append(promises, promise)
	}
	return promises, nil
}

func awaitResponses(ctx context.Context, promises []*containers.Promise[testResponse]) ([]string, error) {
	var (
		responses []string
		errs      []error
	)
	for _, p := range promises {
		res, err := p.Await(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		responses = append(responses, res.Response)
	}
	return responses, errors.Join(errs...)
}

// consume messages from every consumer except stopped ones.
func consume(ctx context.Context, t *testing.T, consumers []*Consumer[testRequest, testResponse]) ([]map[string]string, [][]string) {
	t.Helper()
	gotMessages := messagesMaps(consumersCount)
	wantResponses := make([][]string, consumersCount)
	for idx := 0; idx < consumersCount; idx++ {
		if consumers[idx].Stopped() {
			continue
		}
		idx, c := idx, consumers[idx]
		c.Start(ctx)
		c.StopWaiter.LaunchThread(
			func(ctx context.Context) {
				for {

					res, err := c.Consume(ctx)
					if err != nil {
						if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
							t.Errorf("Consume() unexpected error: %v", err)
							continue
						}
						return
					}
					if res == nil {
						continue
					}
					gotMessages[idx][res.ID] = res.Value.Request
					resp := fmt.Sprintf("result for: %v", res.ID)
					if err := c.SetResult(ctx, res.ID, testResponse{Response: resp}); err != nil {
						t.Errorf("Error setting a result: %v", err)
					}
					wantResponses[idx] = append(wantResponses[idx], resp)
				}
			})
	}
	return gotMessages, wantResponses
}

func TestRedisProduce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		killConsumers bool
	}{
		{
			name:          "all consumers are active",
			killConsumers: false,
		},
		{
			name:          "some consumers killed, others should take over their work",
			killConsumers: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			producer, consumers := newProducerConsumers(ctx, t)
			producer.Start(ctx)
			wantMsgs := wantMessages(messagesCount)
			promises, err := produceMessages(ctx, wantMsgs, producer)
			if err != nil {
				t.Fatalf("Error producing messages: %v", err)
			}
			if tc.killConsumers {
				// Consumer messages in every third consumer but don't ack them to check
				// that other consumers will claim ownership on those messages.
				for i := 0; i < len(consumers); i += 3 {
					if _, err := consumers[i].Consume(ctx); err != nil {
						t.Errorf("Error consuming message: %v", err)
					}
					consumers[i].StopAndWait()
				}

			}
			gotMessages, wantResponses := consume(ctx, t, consumers)
			gotResponses, err := awaitResponses(ctx, promises)
			if err != nil {
				t.Fatalf("Error awaiting responses: %v", err)
			}
			producer.StopAndWait()
			for _, c := range consumers {
				c.StopWaiter.StopAndWait()
			}
			got, err := mergeValues(gotMessages)
			if err != nil {
				t.Fatalf("mergeMaps() unexpected error: %v", err)
			}

			if diff := cmp.Diff(wantMsgs, got); diff != "" {
				t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
			}
			wantResp := flatten(wantResponses)
			sort.Strings(gotResponses)
			if diff := cmp.Diff(wantResp, gotResponses); diff != "" {
				t.Errorf("Unexpected diff in responses:\n%s\n", diff)
			}
			if cnt := producer.promisesLen(); cnt != 0 {
				t.Errorf("Producer still has %d unfullfilled promises", cnt)
			}
		})
	}
}

func TestRedisReproduceDisabled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	producer, consumers := newProducerConsumers(ctx, t, &disableReproduce{})
	producer.Start(ctx)
	wantMsgs := wantMessages(messagesCount)
	promises, err := produceMessages(ctx, wantMsgs, producer)
	if err != nil {
		t.Fatalf("Error producing messages: %v", err)
	}

	// Consumer messages in every third consumer but don't ack them to check
	// that other consumers will claim ownership on those messages.
	for i := 0; i < len(consumers); i += 3 {
		if _, err := consumers[i].Consume(ctx); err != nil {
			t.Errorf("Error consuming message: %v", err)
		}
		consumers[i].StopAndWait()
	}

	gotMessages, _ := consume(ctx, t, consumers)
	gotResponses, err := awaitResponses(ctx, promises)
	if err == nil {
		t.Fatalf("All promises were fullfilled with reproduce disabled and some consumers killed")
	}
	producer.StopAndWait()
	for _, c := range consumers {
		c.StopWaiter.StopAndWait()
	}
	got, err := mergeValues(gotMessages)
	if err != nil {
		t.Fatalf("mergeMaps() unexpected error: %v", err)
	}
	wantMsgCnt := messagesCount - ((consumersCount + 2) / 3)
	if len(got) != wantMsgCnt {
		t.Fatalf("Got: %d messages, want %d", len(got), wantMsgCnt)
	}
	if len(gotResponses) != wantMsgCnt {
		t.Errorf("Got %d responses want: %d\n", len(gotResponses), wantMsgCnt)
	}
	if cnt := producer.promisesLen(); cnt != 0 {
		t.Errorf("Producer still has %d unfullfilled promises", cnt)
	}
}

// mergeValues merges maps from the slice and returns their values.
// Returns and error if there exists duplicate key.
func mergeValues(messages []map[string]string) ([]string, error) {
	res := make(map[string]any)
	var ret []string
	for _, m := range messages {
		for k, v := range m {
			if _, found := res[k]; found {
				return nil, fmt.Errorf("duplicate key: %v", k)
			}
			res[k] = v
			ret = append(ret, v)
		}
	}
	sort.Strings(ret)
	return ret, nil
}

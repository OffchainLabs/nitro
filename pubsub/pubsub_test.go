package pubsub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

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

func createRedisGroup(ctx context.Context, t *testing.T, streamName string, client redis.UniversalClient) {
	t.Helper()
	// Stream name and group name are the same.
	if _, err := client.XGroupCreateMkStream(ctx, streamName, streamName, "$").Result(); err != nil {
		t.Fatalf("Error creating stream group: %v", err)
	}
}

func destroyRedisGroup(ctx context.Context, t *testing.T, streamName string, client redis.UniversalClient) {
	t.Helper()
	if _, err := client.XGroupDestroy(ctx, streamName, streamName).Result(); err != nil {
		log.Debug("Error destroying a stream group", "error", err)
	}
}

type configOpt interface {
	apply(consCfg *ConsumerConfig, prodCfg *ProducerConfig)
}

type withReproduce struct {
	reproduce bool
}

func (e *withReproduce) apply(_ *ConsumerConfig, prodCfg *ProducerConfig) {
	prodCfg.EnableReproduce = e.reproduce
}

func producerCfg() *ProducerConfig {
	return &ProducerConfig{
		EnableReproduce:      TestProducerConfig.EnableReproduce,
		CheckPendingInterval: TestProducerConfig.CheckPendingInterval,
		KeepAliveTimeout:     TestProducerConfig.KeepAliveTimeout,
		CheckResultInterval:  TestProducerConfig.CheckResultInterval,
		CheckPendingItems:    TestProducerConfig.CheckPendingItems,
	}
}

func consumerCfg() *ConsumerConfig {
	return &ConsumerConfig{
		ResponseEntryTimeout: TestConsumerConfig.ResponseEntryTimeout,
		KeepAliveTimeout:     TestConsumerConfig.KeepAliveTimeout,
	}
}

func newProducerConsumers(ctx context.Context, t *testing.T, opts ...configOpt) (redis.UniversalClient, string, *Producer[testRequest, testResponse], []*Consumer[testRequest, testResponse]) {
	t.Helper()
	redisClient, err := redisutil.RedisClientFromURL(redisutil.CreateTestRedis(ctx, t))
	if err != nil {
		t.Fatalf("RedisClientFromURL() unexpected error: %v", err)
	}
	prodCfg, consCfg := producerCfg(), consumerCfg()
	streamName := fmt.Sprintf("stream:%s", uuid.NewString())
	for _, o := range opts {
		o.apply(consCfg, prodCfg)
	}
	producer, err := NewProducer[testRequest, testResponse](redisClient, streamName, prodCfg)
	if err != nil {
		t.Fatalf("Error creating new producer: %v", err)
	}

	var consumers []*Consumer[testRequest, testResponse]
	for i := 0; i < consumersCount; i++ {
		c, err := NewConsumer[testRequest, testResponse](redisClient, streamName, consCfg)
		if err != nil {
			t.Fatalf("Error creating new consumer: %v", err)
		}
		consumers = append(consumers, c)
	}
	createRedisGroup(ctx, t, streamName, producer.client)
	t.Cleanup(func() {
		ctx := context.Background()
		destroyRedisGroup(ctx, t, streamName, producer.client)
		var keys []string
		for _, c := range consumers {
			keys = append(keys, c.heartBeatKey())
		}
		if _, err := producer.client.Del(ctx, keys...).Result(); err != nil {
			log.Debug("Error deleting heartbeat keys", "error", err)
		}
	})
	return redisClient, streamName, producer, consumers
}

func messagesMaps(n int) []map[string]string {
	ret := make([]map[string]string, n)
	for i := 0; i < n; i++ {
		ret[i] = make(map[string]string)
	}
	return ret
}

func msgForIndex(idx int) string {
	return fmt.Sprintf("msg: %d", idx)
}

func wantMessages(n int) []string {
	var ret []string
	for i := 0; i < n; i++ {
		ret = append(ret, msgForIndex(i))
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

func awaitResponses(ctx context.Context, promises []*containers.Promise[testResponse]) ([]string, []int) {
	var (
		responses []string
		errs      []int
	)
	for idx, p := range promises {
		res, err := p.Await(ctx)
		if err != nil {
			errs = append(errs, idx)
			continue
		}
		responses = append(responses, res.Response)
	}
	return responses, errs
}

// consume messages from every consumer except stopped ones.
func consume(ctx context.Context, t *testing.T, consumers []*Consumer[testRequest, testResponse], gotMessages []map[string]string) [][]string {
	t.Helper()
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
	return wantResponses
}

func TestRedisProduce(t *testing.T) {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelTrace, true)))
	t.Parallel()
	for _, tc := range []struct {
		name          string
		killConsumers bool
		autoRecover   bool
	}{
		{
			name:          "all consumers are active",
			killConsumers: false,
			autoRecover:   false,
		},
		{
			name:          "some consumers killed, others should take over their work",
			killConsumers: true,
			autoRecover:   true,
		},
		{
			name:          "some consumers killed, should return failure",
			killConsumers: true,
			autoRecover:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			redisClient, streamName, producer, consumers := newProducerConsumers(ctx, t, &withReproduce{tc.autoRecover})
			producer.Start(ctx)
			wantMsgs := wantMessages(messagesCount)
			promises, err := produceMessages(ctx, wantMsgs, producer)
			if err != nil {
				t.Fatalf("Error producing messages: %v", err)
			}
			gotMessages := messagesMaps(len(consumers))
			if tc.killConsumers {
				// Consumer messages in every third consumer but don't ack them to check
				// that other consumers will claim ownership on those messages.
				for i := 0; i < len(consumers); i += 3 {
					consumers[i].Start(ctx)
					req, err := consumers[i].Consume(ctx)
					if err != nil {
						t.Errorf("Error consuming message: %v", err)
					}
					if !tc.autoRecover {
						gotMessages[i][req.ID] = req.Value.Request
					}
					consumers[i].StopAndWait()
				}

			}
			time.Sleep(time.Second)
			wantResponses := consume(ctx, t, consumers, gotMessages)
			gotResponses, errIndexes := awaitResponses(ctx, promises)
			if len(errIndexes) != 0 && tc.autoRecover {
				t.Fatalf("Error awaiting responses: %v", errIndexes)
			}
			producer.StopAndWait()
			for _, c := range consumers {
				c.StopAndWait()
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
			// Trigger a trim
			producer.checkResponses(ctx)
			msgs, err := redisClient.XRange(ctx, streamName, "-", "+").Result()
			if err != nil {
				t.Errorf("XRange failed: %v", err)
			}
			if len(msgs) != 0 {
				t.Errorf("redis still has %v messages", len(msgs))
			}
		})
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

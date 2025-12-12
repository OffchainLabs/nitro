package pubsub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
)

var (
	consumersCount = 10
	messagesCount  = 100
)

type testRequest struct {
	Request   string
	IsInvalid bool
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

func producerConfigs(count int) []*ProducerConfig {
	var configs []*ProducerConfig
	for i := 0; i < count; i++ {
		config := TestProducerConfig
		config.RequestTimeout = 2 * time.Second
		configs = append(configs, &config)
	}
	return configs
}

func consumerConfigs(count int) []*ConsumerConfig {
	var configs []*ConsumerConfig
	for i := 0; i < count; i++ {
		config := TestConsumerConfig
		configs = append(configs, &config)
	}
	return configs
}

func newProducersConsumers(ctx context.Context, t *testing.T, producersCount, consumersCount, notRetryingConsumers int) (redis.UniversalClient, string, []*Producer[testRequest, testResponse], []*Consumer[testRequest, testResponse]) {
	t.Helper()
	if notRetryingConsumers > consumersCount {
		t.Fatal("internal test error - notRetryingConsumers > consumersCount")
	}
	producerConfigs, consumerConfigs := producerConfigs(producersCount), consumerConfigs(consumersCount)
	for i := 0; i < notRetryingConsumers; i++ {
		consumerConfigs[i].Retry = false
	}
	return newProducersConsumersForConfigs(ctx, t, producerConfigs, consumerConfigs)
}

func newProducersConsumersForConfigs(ctx context.Context, t *testing.T, producerConfigs []*ProducerConfig, consumerConfigs []*ConsumerConfig) (redis.UniversalClient, string, []*Producer[testRequest, testResponse], []*Consumer[testRequest, testResponse]) {
	t.Helper()
	if len(producerConfigs) == 0 {
		t.Fatalf("internal test error - helper got empty producer configs list")
	}
	redisClient, err := redisutil.RedisClientFromURL(redisutil.CreateTestRedis(ctx, t))
	if err != nil {
		t.Fatalf("RedisClientFromURL() unexpected error: %v", err)
	}
	streamName := fmt.Sprintf("stream:%s", uuid.NewString())
	var producers []*Producer[testRequest, testResponse]
	for _, producerConfig := range producerConfigs {
		producer, err := NewProducer[testRequest, testResponse](redisClient, streamName, producerConfig)
		if err != nil {
			t.Fatalf("Error creating new producer: %v", err)
		}
		producers = append(producers, producer)
	}
	var consumers []*Consumer[testRequest, testResponse]
	for _, consumerConfig := range consumerConfigs {
		c, err := NewConsumer[testRequest, testResponse](redisClient, streamName, consumerConfig)
		if err != nil {
			t.Fatalf("Error creating new consumer: %v", err)
		}
		consumers = append(consumers, c)
	}
	createRedisGroup(ctx, t, streamName, producers[0].client)
	t.Cleanup(func() {
		ctx := context.Background()
		destroyRedisGroup(ctx, t, streamName, producers[0].client)
	})
	return redisClient, streamName, producers, consumers
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

func wantMessages(entriesCounts []int) [][]string {
	ret := make([][]string, len(entriesCounts))
	for i, n := range entriesCounts {
		group := ""
		if len(entriesCounts) > 1 {
			group = fmt.Sprintf("%d.", i)
		}
		for j := 0; j < n; j++ {
			ret[i] = append(ret[i], group+msgForIndex(j))
		}
		sort.Strings(ret[i])
	}
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

func produceMessages(ctx context.Context, msgs []string, producer *Producer[testRequest, testResponse], withInvalidEntries bool) ([]*containers.Promise[testResponse], []testRequest, error) {
	var promises []*containers.Promise[testResponse]
	var requests []testRequest
	for i := 0; i < len(msgs); i++ {
		request := testRequest{Request: msgs[i]}
		if withInvalidEntries && i%50 == 0 {
			request.IsInvalid = true
		}
		promise, err := producer.Produce(ctx, request)
		if err != nil {
			return nil, nil, err
		}
		promises = append(promises, promise)
		requests = append(requests, request)
	}
	return promises, requests, nil
}

func awaitResponses(ctx context.Context, promises []*containers.Promise[testResponse]) ([]string, []error) {
	var (
		responses []string
		errs      []error
	)
	for _, p := range promises {
		res, err := p.Await(ctx)
		responses = append(responses, res.Response)
		errs = append(errs, err)
	}
	return responses, errs
}

// consume messages from every consumer except stopped ones.
func consume(ctx context.Context, t *testing.T, consumers []*Consumer[testRequest, testResponse], gotMessages []map[string]string) ([][]string, [][]string) {
	t.Helper()
	wantResponses := make([][]string, consumersCount)
	wantErrors := make([][]string, consumersCount)
	for idx := 0; idx < consumersCount; idx++ {
		if consumers[idx].Stopped() {
			continue
		}
		idx, c := idx, consumers[idx]
		c.Start(ctx)
		c.StopWaiter.LaunchThread(
			func(ctx context.Context) {
				for {
					msg, err := c.Consume(ctx)
					if err != nil {
						if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
							t.Errorf("Consume() unexpected error: %v", err)
							continue
						}
						return
					}
					if msg == nil {
						continue
					}
					gotMessages[idx][msg.ID] = msg.Value.Request
					if msg.Value.IsInvalid {
						errString := fmt.Sprintf("invalid request: %v", msg.ID)
						if err := c.SetError(ctx, msg.ID, errString); err != nil {
							t.Errorf("Error setting a error: %v", err)
						}
						wantErrors[idx] = append(wantErrors[idx], errString)
					} else {
						resp := fmt.Sprintf("result for: %v", msg.ID)
						if err := c.SetResult(ctx, msg.ID, testResponse{Response: resp}); err != nil {
							t.Errorf("Error setting a result: %v", err)
						}
						wantResponses[idx] = append(wantResponses[idx], resp)
					}
					msg.Ack()
				}
			})
	}
	return wantResponses, wantErrors
}

func TestRedisProduceComplex(t *testing.T) {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelTrace, true)))
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		entriesCount         []int
		numProducers         int
		killConsumers        bool
		withInvalidEntries   bool // If this is set, then every 50th entry is invalid (requests that can't be solved by any consumer)
		notRetryingConsumers int  // number of consumers that won't retry timed out messages
	}{
		{
			name:         "one producer, all consumers are active",
			entriesCount: []int{messagesCount},
			numProducers: 1,
		},
		{
			name:         "two producers, all consumers are active",
			entriesCount: []int{20, 20},
			numProducers: 2,
		},
		{
			name:          "one producer, some consumers killed, others should take over their work",
			entriesCount:  []int{messagesCount},
			numProducers:  1,
			killConsumers: true,
		},
		{
			name:                 "one producer, some consumers killed, others should NOT take over their work",
			entriesCount:         []int{messagesCount},
			numProducers:         1,
			killConsumers:        true,
			notRetryingConsumers: consumersCount,
		},
		{
			name:                 "one producer, some consumers killed, one retrying consumer should take over their work",
			entriesCount:         []int{messagesCount},
			numProducers:         1,
			killConsumers:        true,
			notRetryingConsumers: consumersCount - 1,
		},
		{
			name:          "two producers, some consumers killed, others should take over their work, unequal number of requests from producers",
			entriesCount:  []int{messagesCount, 2 * messagesCount},
			numProducers:  2,
			killConsumers: true,
		},
		{
			name:                 "two producers, some consumers killed, others should NOT take over their work, unequal number of requests from producers",
			entriesCount:         []int{messagesCount, 2 * messagesCount},
			numProducers:         2,
			killConsumers:        true,
			notRetryingConsumers: consumersCount,
		},
		{
			name:                 "two producers, some consumers killed, one retrying consumer take over their work, unequal number of requests from producers",
			entriesCount:         []int{messagesCount, 2 * messagesCount},
			numProducers:         2,
			killConsumers:        true,
			notRetryingConsumers: consumersCount - 1,
		},
		{
			name:               "two producers, some consumers killed, others should take over their work, some invalid entries, unequal number of requests from producers",
			entriesCount:       []int{messagesCount, 2 * messagesCount},
			numProducers:       2,
			killConsumers:      true,
			withInvalidEntries: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			redisClient, streamName, producers, consumers := newProducersConsumers(ctx, t, tc.numProducers, consumersCount, tc.notRetryingConsumers)

			for _, producer := range producers {
				producer.Start(ctx)
			}

			entries := wantMessages(tc.entriesCount)

			var promises [][]*containers.Promise[testResponse]
			var requests [][]testRequest
			for i := 0; i < tc.numProducers; i++ {
				prs, rqs, err := produceMessages(ctx, entries[i], producers[i], tc.withInvalidEntries)
				if err != nil {
					t.Fatalf("Error producing messages from producer%d: %v", i, err)
				}
				promises = append(promises, prs)
				requests = append(requests, rqs)
			}

			gotMessages := messagesMaps(len(consumers))
			killedEntries := make(map[string]struct{})
			if tc.killConsumers {
				// Consumer messages in every third consumer but don't ack them to check
				// that other consumers will claim ownership on those messages.

				// make sure not to remove the only retrying / not-retrying consumer
				var keepOneNotRetrying, keepOneRetrying bool
				if tc.notRetryingConsumers == 1 {
					keepOneNotRetrying = true
				}
				if tc.notRetryingConsumers == consumersCount-1 {
					keepOneRetrying = true
				}
				for i := 0; i < len(consumers); i += 3 {
					if keepOneRetrying && consumers[i].cfg.Retry {
						keepOneRetrying = false
						continue
					}
					if keepOneNotRetrying && !consumers[i].cfg.Retry {
						keepOneNotRetrying = false
						continue
					}
					consumers[i].Start(ctx)
					req, err := consumers[i].Consume(ctx)
					if err != nil {
						t.Errorf("Error consuming message: %v", err)
					}
					if req == nil {
						t.Error("Didn't consume any message")
					} else {
						killedEntries[req.Value.Request] = struct{}{}
					}
					// Kills the actnotifier hence allowing XAUTOCLAIM
					consumers[i].StopAndWait()
				}

			}
			killedConsumers := len(killedEntries)

			time.Sleep(time.Second)
			wantResponses, wantErrors := consume(ctx, t, consumers, gotMessages)

			var gotResponses []string
			var gotErrors []string
			waitingForTooLong := 0
			for i := 0; i < tc.numProducers; i++ {
				responses, errs := awaitResponses(ctx, promises[i])
				if len(errs) != len(responses) {
					t.Errorf("internal test error - got unexpected number of errors, should be equal number of responses, producer: %d, len(errs): %d, len(responses): %d", i, len(errs), len(responses))
				}
				if t.Failed() {
					continue
				}
				for j := 0; j < len(responses); j++ {
					request, response, err := requests[i][j], responses[j], errs[j]
					if err != nil && strings.Contains(err.Error(), "request has been waiting for too long") && tc.notRetryingConsumers == consumersCount && killedConsumers > 0 {
						// we expect this error in case all consumers are not retrying and some have been killed
						waitingForTooLong++
						continue
					}
					if !request.IsInvalid && err != nil {
						t.Errorf("Unexpected error while awaiting responses, producer: %d, response: %d, err: %v", i, j, err)
					} else if request.IsInvalid && err == nil {
						t.Errorf("Did not get expected error while awaiting responses, producer: %d, response: %d, err: %v", i, j, err)
					} else if err == nil {
						gotResponses = append(gotResponses, response)
					} else {
						gotErrors = append(gotErrors, err.Error())
					}
				}
			}
			if waitingForTooLong > killedConsumers {
				t.Errorf("Got to many \"request has been waiting for too long\" errors, got: %d, expected: %d", waitingForTooLong, killedConsumers)
			}
			if t.Failed() {
				t.FailNow()
			}

			for _, c := range consumers {
				c.StopAndWait()
			}

			got, err := mergeValues(gotMessages, tc.withInvalidEntries)
			if err != nil {
				t.Fatalf("mergeMaps() unexpected error: %v", err)
			}
			// Only when there are invalid entries got will have duplicates
			if tc.withInvalidEntries {
				got = removeDuplicates(got)
			}

			var combinedEntries []string
			for i := 0; i < tc.numProducers; i++ {
				producerEntries := entries[i]
				if len(killedEntries) > 0 && tc.notRetryingConsumers == consumersCount {
					producerEntries = filterEntries(producerEntries, killedEntries)
				}
				combinedEntries = append(combinedEntries, producerEntries...)
			}
			wantMsgs := combinedEntries
			if diff := cmp.Diff(wantMsgs, got); diff != "" {
				t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
			}

			sort.Strings(gotResponses)
			wantResp := flatten(wantResponses)
			if diff := cmp.Diff(wantResp, gotResponses); diff != "" {
				t.Errorf("Unexpected diff in responses:\n%s\n", diff)
			}

			sort.Strings(gotErrors)
			wantErr := flatten(wantErrors)
			if diff := cmp.Diff(wantErr, gotErrors); diff != "" {
				t.Errorf("Unexpected diff in errors:\n%s\n", diff)
			}

			// Check each producers all promises were responded to
			for i := 0; i < tc.numProducers; i++ {
				if cnt := producers[i].promisesLen(); cnt != 0 {
					t.Errorf("Producer%d still has %d unfullfilled promises", i, cnt)
				}
			}

			// Trigger a trim
			time.Sleep(time.Second)
			for i := 0; i < tc.numProducers; i++ {
				producers[i].checkResponses(ctx)
				producers[i].StopAndWait()
			}

			// Check that no messages remain in the stream
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

func removeDuplicates(list []string) []string {
	capture := map[string]bool{}
	var ret []string
	for _, elem := range list {
		if _, found := capture[elem]; !found {
			ret = append(ret, elem)
			capture[elem] = true
		}
	}
	sort.Strings(ret)
	return ret
}

// mergeValues merges maps from the slice and returns their values.
// Returns and error if there exists duplicate key.
func mergeValues(messages []map[string]string, withInvalidEntries bool) ([]string, error) {
	res := make(map[string]any)
	var ret []string
	for _, m := range messages {
		for k, v := range m {
			if _, found := res[k]; found && !withInvalidEntries {
				return nil, fmt.Errorf("duplicate key: %v", k)
			}
			res[k] = v
			ret = append(ret, v)
		}
	}
	sort.Strings(ret)
	return ret, nil
}

func filterEntries(entries []string, toSkip map[string]struct{}) []string {
	var res []string
	for _, e := range entries {
		if _, skip := toSkip[e]; !skip {
			res = append(res, e)
		}
	}
	return res
}

package pubsub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/redis/go-redis/v9"
)

var (
	consumersCount = 10
	messagesCount  = 100
)

type testRequest struct {
	Request   string
	IsInvalid bool
	SelfHash  string // Is a unique identifier which can be used to compare any two validationInputs
}

// SetSelfHash should be only called once. In the context of redis streams- by the producer
func (t *testRequest) SetSelfHash() {
	jsonData, err := json.Marshal(t)
	if err != nil {
		return
	}
	hash := sha256.Sum256(jsonData)
	t.SelfHash = hex.EncodeToString(hash[:])
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

func producerCfg() *ProducerConfig {
	return &ProducerConfig{
		CheckResultInterval:  TestProducerConfig.CheckResultInterval,
		ResponseEntryTimeout: TestProducerConfig.ResponseEntryTimeout,
		RequestTimeout:       2 * time.Second,
	}
}

func consumerCfg() *ConsumerConfig {
	return &ConsumerConfig{
		ResponseEntryTimeout: TestConsumerConfig.ResponseEntryTimeout,
		IdletimeToAutoclaim:  TestConsumerConfig.IdletimeToAutoclaim,
	}
}

func newProducerConsumers(ctx context.Context, t *testing.T) (redis.UniversalClient, string, *Producer[testRequest, testResponse], []*Consumer[testRequest, testResponse]) {
	t.Helper()
	redisClient, err := redisutil.RedisClientFromURL(redisutil.CreateTestRedis(ctx, t))
	if err != nil {
		t.Fatalf("RedisClientFromURL() unexpected error: %v", err)
	}
	prodCfg, consCfg := producerCfg(), consumerCfg()
	streamName := fmt.Sprintf("stream:%s", uuid.NewString())

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

func wantMessages(n int, group string, withDuplicates bool) []string {
	var ret []string
	for i := 0; i < n; i++ {
		ret = append(ret, group+msgForIndex(i))
		if withDuplicates && i%3 == 0 {
			ret = append(ret, msgForIndex(i))
		}
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

func produceMessages(ctx context.Context, msgs []string, producer *Producer[testRequest, testResponse], useUniqueIdentifier, withInvalidEntries bool) ([]*containers.Promise[testResponse], error) {
	var promises []*containers.Promise[testResponse]
	for i := 0; i < len(msgs); i++ {
		req := testRequest{Request: msgs[i]}
		if withInvalidEntries && i%50 == 0 {
			req.IsInvalid = true
		}
		if useUniqueIdentifier {
			req.SetSelfHash()
		}
		promise, err := producer.Produce(ctx, req.SelfHash, req)
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

					res, ackNotifier, err := c.Consume(ctx)
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
					if !res.Value.IsInvalid {
						resp := fmt.Sprintf("result for: %v", res.ID)
						if err := c.SetResult(ctx, res.Value.SelfHash, res.ID, testResponse{Response: resp}); err != nil {
							t.Errorf("Error setting a result: %v", err)
						}
						wantResponses[idx] = append(wantResponses[idx], resp)
					}
					close(ackNotifier)
				}
			})
	}
	return wantResponses
}

func TestRedisProduceComplex(t *testing.T) {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelTrace, true)))
	t.Parallel()
	for _, tc := range []struct {
		name               string
		entriesCount       []int
		numProducers       int
		withDuplicates     bool // If this is set, then every fourth entry (while generation) of each entries list is equal
		killConsumers      bool
		withInvalidEntries bool // If this is set, then every 50th entry is invalid (requests that can't be solved by any consumer)
	}{
		{
			name:         "one producer, all consumers are active",
			entriesCount: []int{messagesCount},
			numProducers: 1,
		},
		{
			name:          "one producer, some consumers killed, others should take over their work",
			entriesCount:  []int{messagesCount},
			numProducers:  1,
			killConsumers: true,
		},
		{
			name:         "two producers, all consumers are active, all unique entries",
			entriesCount: []int{20, 20},
			numProducers: 2,
		},
		{
			name:           "two producers, all consumers are active, some duplicate entries",
			entriesCount:   []int{20, 20},
			numProducers:   2,
			withDuplicates: true,
		},
		{
			name:           "two producers, some consumers killed, others should take over their work, some duplicate entries, unequal number of requests from producers",
			entriesCount:   []int{messagesCount, 2 * messagesCount},
			numProducers:   2,
			withDuplicates: true,
			killConsumers:  true,
		},
		{
			name:               "two producers, some consumers killed, others should take over their work, some duplicate entries, some invalid entries, unequal number of requests from producers",
			entriesCount:       []int{messagesCount, 2 * messagesCount},
			numProducers:       2,
			withDuplicates:     true,
			killConsumers:      true,
			withInvalidEntries: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var producers []*Producer[testRequest, testResponse]
			redisClient, streamName, producer, consumers := newProducerConsumers(ctx, t)
			producers = append(producers, producer)
			if tc.numProducers == 2 {
				producer, err := NewProducer[testRequest, testResponse](redisClient, streamName, producerCfg())
				if err != nil {
					t.Fatalf("Error creating second producer: %v", err)
				}
				producers = append(producers, producer)
			}

			for _, producer := range producers {
				producer.Start(ctx)
			}

			var entries [][]string
			if tc.numProducers == 2 {
				entries = append(entries, wantMessages(tc.entriesCount[0], "1.", tc.withDuplicates))
				entries = append(entries, wantMessages(tc.entriesCount[1], "2.", tc.withDuplicates))
			} else {
				entries = append(entries, wantMessages(tc.entriesCount[0], "", tc.withDuplicates))
			}

			var promises [][]*containers.Promise[testResponse]
			for i := 0; i < tc.numProducers; i++ {
				prs, err := produceMessages(ctx, entries[i], producers[i], tc.numProducers == 2, tc.withInvalidEntries)
				if err != nil {
					t.Fatalf("Error producing messages from producer%d: %v", i, err)
				}
				promises = append(promises, prs)
			}

			gotMessages := messagesMaps(len(consumers))
			if tc.killConsumers {
				// Consumer messages in every third consumer but don't ack them to check
				// that other consumers will claim ownership on those messages.
				for i := 0; i < len(consumers); i += 3 {
					consumers[i].Start(ctx)
					req, _, err := consumers[i].Consume(ctx)
					if err != nil {
						t.Errorf("Error consuming message: %v", err)
					}
					if req == nil {
						t.Error("Didn't consume any message")
					}
					// Kills the actnotifier hence allowing XAUTOCLAIM
					consumers[i].StopAndWait()
				}

			}

			time.Sleep(time.Second)
			wantResponses := consume(ctx, t, consumers, gotMessages)

			var gotResponses []string
			for i := 0; i < tc.numProducers; i++ {
				grs, errIndexes := awaitResponses(ctx, promises[i])
				if tc.withInvalidEntries {
					if errIndexes[len(errIndexes)-1]+50 <= len(entries[i]) {
						t.Fatalf("Unexpected number of invalid requests while awaiting responses")
					}
					for j, idx := range errIndexes {
						if idx != j*50 {
							t.Fatalf("Invalid request' index mismatch want: %d got %d", j*50, idx)
						}
					}
				} else if len(errIndexes) != 0 {
					t.Fatalf("Error awaiting responses from promises %d: %v", i, errIndexes)
				}
				gotResponses = append(gotResponses, grs...)
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
				combinedEntries = append(combinedEntries, entries[i]...)
			}
			wantMsgs := removeDuplicates(combinedEntries)
			if diff := cmp.Diff(wantMsgs, got); diff != "" {
				t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
			}

			// Consumers are not supposed to get duplicate requests
			gotResponses = removeDuplicates(gotResponses)
			wantResp := flatten(wantResponses)
			if diff := cmp.Diff(wantResp, gotResponses); diff != "" {
				t.Errorf("Unexpected diff in responses:\n%s\n", diff)
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

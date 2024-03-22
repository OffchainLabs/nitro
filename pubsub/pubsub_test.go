package pubsub

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/util/redisutil"
)

var (
	streamName     = "validator_stream"
	consumersCount = 10
	messagesCount  = 100
)

type testConsumer struct {
	consumer *Consumer
	cancel   context.CancelFunc
}

func createGroup(ctx context.Context, t *testing.T, client *redis.Client) {
	t.Helper()
	_, err := client.XGroupCreateMkStream(ctx, streamName, "default", "$").Result()
	if err != nil {
		t.Fatalf("Error creating stream group: %v", err)
	}
}

func newProducerConsumers(ctx context.Context, t *testing.T) (*Producer, []*testConsumer) {
	t.Helper()
	// tmpI, tmpT := KeepAliveInterval, KeepAliveTimeout
	// KeepAliveInterval, KeepAliveTimeout = 5*time.Millisecond, 30*time.Millisecond
	// t.Cleanup(func() { KeepAliveInterval, KeepAliveTimeout = tmpI, tmpT })

	redisURL := redisutil.CreateTestRedis(ctx, t)
	producer, err := NewProducer(&ProducerConfig{RedisURL: redisURL, RedisStream: streamName})
	if err != nil {
		t.Fatalf("Error creating new producer: %v", err)
	}
	var consumers []*testConsumer
	for i := 0; i < consumersCount; i++ {
		consumerCtx, cancel := context.WithCancel(ctx)
		c, err := NewConsumer(consumerCtx,
			&ConsumerConfig{
				RedisURL:          redisURL,
				RedisStream:       streamName,
				KeepAliveInterval: 5 * time.Millisecond,
				KeepAliveTimeout:  30 * time.Millisecond,
			},
		)
		if err != nil {
			t.Fatalf("Error creating new consumer: %v", err)
		}
		consumers = append(consumers, &testConsumer{
			consumer: c,
			cancel:   cancel,
		})
	}
	createGroup(ctx, t, producer.client)
	return producer, consumers
}

func messagesMap(n int) []map[string]any {
	ret := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		ret[i] = make(map[string]any)
	}
	return ret
}

func wantMessages(n int) []any {
	var ret []any
	for i := 0; i < n; i++ {
		ret = append(ret, fmt.Sprintf("msg: %d", i))
	}
	sort.Slice(ret, func(i, j int) bool {
		return fmt.Sprintf("%v", ret[i]) < fmt.Sprintf("%v", ret[j])
	})
	return ret
}

func TestProduce(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	producer, consumers := newProducerConsumers(ctx, t)
	consumerCtx, cancelConsumers := context.WithTimeout(ctx, time.Second)
	gotMessages := messagesMap(consumersCount)

	for idx, c := range consumers {
		idx, c := idx, c.consumer
		go func() {
			for {
				res, err := c.Consume(consumerCtx)
				if err != nil {
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Errorf("Consume() unexpected error: %v", err)
					}
					return
				}
				if res == nil {
					continue
				}
				gotMessages[idx][res.ID] = res.Value
				if err := c.ACK(consumerCtx, res.ID); err != nil {
					t.Errorf("Error ACKing message: %v, error: %v", res.ID, err)
				}
			}
		}()
	}

	for i := 0; i < messagesCount; i++ {
		value := fmt.Sprintf("msg: %d", i)
		if err := producer.Produce(ctx, value); err != nil {
			t.Errorf("Produce() unexpected error: %v", err)
		}
	}
	time.Sleep(time.Second)
	cancelConsumers()
	got, err := mergeValues(gotMessages)
	if err != nil {
		t.Fatalf("mergeMaps() unexpected error: %v", err)
	}
	want := wantMessages(messagesCount)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
	}
}

func TestClaimingOwnership(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	producer, consumers := newProducerConsumers(ctx, t)
	consumerCtx, cancelConsumers := context.WithCancel(ctx)
	gotMessages := messagesMap(consumersCount)

	// Consumer messages in every third consumer but don't ack them to check
	// that other consumers will claim ownership on those messages.
	for i := 0; i < len(consumers); i += 3 {
		i := i
		consumers[i].cancel()
		go func() {
			if _, err := consumers[i].consumer.Consume(context.Background()); err != nil {
				t.Errorf("Error consuming message: %v", err)
			}
		}()
	}
	var total atomic.Uint64

	for idx, c := range consumers {
		idx, c := idx, c.consumer
		go func() {
			for {
				if idx%3 == 0 {
					continue
				}
				res, err := c.Consume(consumerCtx)
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
				gotMessages[idx][res.ID] = res.Value
				if err := c.ACK(consumerCtx, res.ID); err != nil {
					t.Errorf("Error ACKing message: %v, error: %v", res.ID, err)
				}
				total.Add(1)
			}
		}()
	}

	for i := 0; i < messagesCount; i++ {
		value := fmt.Sprintf("msg: %d", i)
		if err := producer.Produce(ctx, value); err != nil {
			t.Errorf("Produce() unexpected error: %v", err)
		}
	}

	for {
		if total.Load() < uint64(messagesCount) {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	cancelConsumers()
	got, err := mergeValues(gotMessages)
	if err != nil {
		t.Fatalf("mergeMaps() unexpected error: %v", err)
	}
	want := wantMessages(messagesCount)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
	}
}

// mergeValues merges maps from the slice and returns their values.
// Returns and error if there exists duplicate key.
func mergeValues(messages []map[string]any) ([]any, error) {
	res := make(map[string]any)
	var ret []any
	for _, m := range messages {
		for k, v := range m {
			if _, found := res[k]; found {
				return nil, fmt.Errorf("duplicate key: %v", k)
			}
			res[k] = v
			ret = append(ret, v)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return fmt.Sprintf("%v", ret[i]) < fmt.Sprintf("%v", ret[j])
	})
	return ret, nil
}

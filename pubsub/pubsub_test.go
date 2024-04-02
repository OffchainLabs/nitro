package pubsub

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/redisutil"
)

var (
	streamName     = DefaultTestProducerConfig.RedisStream
	consumersCount = 10
	messagesCount  = 100
)

type testRequest struct {
	request string
}

func (r *testRequest) Marshal() []byte {
	return []byte(r.request)
}

func (r *testRequest) Unmarshal(val []byte) (*testRequest, error) {
	return &testRequest{
		request: string(val),
	}, nil
}

type testResponse struct {
	response string
}

func (r *testResponse) Marshal() []byte {
	return []byte(r.response)
}

func (r *testResponse) Unmarshal(val []byte) (*testResponse, error) {
	return &testResponse{
		response: string(val),
	}, nil
}

func createGroup(ctx context.Context, t *testing.T, client redis.UniversalClient) {
	t.Helper()
	_, err := client.XGroupCreateMkStream(ctx, streamName, defaultGroup, "$").Result()
	if err != nil {
		t.Fatalf("Error creating stream group: %v", err)
	}
}

func newProducerConsumers(ctx context.Context, t *testing.T) (*Producer[*testRequest, *testResponse], []*Consumer[*testRequest, *testResponse]) {
	t.Helper()
	redisURL := redisutil.CreateTestRedis(ctx, t)
	defaultProdCfg := DefaultTestProducerConfig
	defaultProdCfg.RedisURL = redisURL
	producer, err := NewProducer[*testRequest, *testResponse](defaultProdCfg)
	if err != nil {
		t.Fatalf("Error creating new producer: %v", err)
	}
	defaultCfg := DefaultTestConsumerConfig
	defaultCfg.RedisURL = redisURL
	var consumers []*Consumer[*testRequest, *testResponse]
	for i := 0; i < consumersCount; i++ {
		c, err := NewConsumer[*testRequest, *testResponse](ctx, defaultCfg)
		if err != nil {
			t.Fatalf("Error creating new consumer: %v", err)
		}
		consumers = append(consumers, c)
	}
	createGroup(ctx, t, producer.client)
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
	sort.Slice(ret, func(i, j int) bool {
		return fmt.Sprintf("%v", ret[i]) < fmt.Sprintf("%v", ret[j])
	})
	return ret
}

func TestProduce(t *testing.T) {
	ctx := context.Background()
	producer, consumers := newProducerConsumers(ctx, t)
	producer.Start(ctx)
	gotMessages := messagesMaps(consumersCount)
	wantResponses := make([][]string, len(consumers))
	for idx, c := range consumers {
		idx, c := idx, c
		c.Start(ctx)
		c.StopWaiter.LaunchThread(
			func(ctx context.Context) {
				for {
					res, err := c.Consume(ctx)
					if err != nil {
						if !errors.Is(err, context.Canceled) {
							t.Errorf("Consume() unexpected error: %v", err)
						}
						return
					}
					if res == nil {
						continue
					}
					gotMessages[idx][res.ID] = res.Value.request
					resp := &testResponse{response: fmt.Sprintf("result for: %v", res.ID)}
					if err := c.SetResult(ctx, res.ID, resp); err != nil {
						t.Errorf("Error setting a result: %v", err)
					}
					wantResponses[idx] = append(wantResponses[idx], resp.response)
				}
			})
	}

	var gotResponses []string

	for i := 0; i < messagesCount; i++ {
		value := &testRequest{request: fmt.Sprintf("msg: %d", i)}
		p, err := producer.Produce(ctx, value)
		if err != nil {
			t.Errorf("Produce() unexpected error: %v", err)
		}
		res, err := p.Await(ctx)
		if err != nil {
			t.Errorf("Await() unexpected error: %v", err)
		}
		gotResponses = append(gotResponses, res.response)
	}

	producer.StopWaiter.StopAndWait()
	for _, c := range consumers {
		c.StopAndWait()
	}

	got, err := mergeValues(gotMessages)
	if err != nil {
		t.Fatalf("mergeMaps() unexpected error: %v", err)
	}
	want := wantMessages(messagesCount)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
	}

	wantResp := flatten(wantResponses)
	sort.Slice(gotResponses, func(i, j int) bool {
		return gotResponses[i] < gotResponses[j]
	})
	if diff := cmp.Diff(wantResp, gotResponses); diff != "" {
		t.Errorf("Unexpected diff in responses:\n%s\n", diff)
	}
}

func flatten(responses [][]string) []string {
	var ret []string
	for _, v := range responses {
		ret = append(ret, v...)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

func TestClaimingOwnership(t *testing.T) {
	ctx := context.Background()
	producer, consumers := newProducerConsumers(ctx, t)
	producer.Start(ctx)
	gotMessages := messagesMaps(consumersCount)

	// Consumer messages in every third consumer but don't ack them to check
	// that other consumers will claim ownership on those messages.
	for i := 0; i < len(consumers); i += 3 {
		i := i
		if _, err := consumers[i].Consume(ctx); err != nil {
			t.Errorf("Error consuming message: %v", err)
		}
		consumers[i].StopAndWait()
	}
	var total atomic.Uint64

	wantResponses := make([][]string, len(consumers))
	for idx := 0; idx < len(consumers); idx++ {
		if idx%3 == 0 {
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
					gotMessages[idx][res.ID] = res.Value.request
					resp := &testResponse{response: fmt.Sprintf("result for: %v", res.ID)}
					if err := c.SetResult(ctx, res.ID, resp); err != nil {
						t.Errorf("Error setting a result: %v", err)
					}
					wantResponses[idx] = append(wantResponses[idx], resp.response)
					total.Add(1)
				}
			})
	}

	var promises []*containers.Promise[*testResponse]
	for i := 0; i < messagesCount; i++ {
		value := &testRequest{request: fmt.Sprintf("msg: %d", i)}
		promise, err := producer.Produce(ctx, value)
		if err != nil {
			t.Errorf("Produce() unexpected error: %v", err)
		}
		promises = append(promises, promise)
	}
	var gotResponses []string
	for _, p := range promises {
		res, err := p.Await(ctx)
		if err != nil {
			t.Errorf("Await() unexpected error: %v", err)
			continue
		}
		gotResponses = append(gotResponses, res.response)
	}

	for {
		if total.Load() < uint64(messagesCount) {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	for _, c := range consumers {
		c.StopWaiter.StopAndWait()
	}
	got, err := mergeValues(gotMessages)
	if err != nil {
		t.Fatalf("mergeMaps() unexpected error: %v", err)
	}
	want := wantMessages(messagesCount)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected diff (-want +got):\n%s\n", diff)
	}
	WantResp := flatten(wantResponses)
	sort.Slice(gotResponses, func(i, j int) bool {
		return gotResponses[i] < gotResponses[j]
	})
	if diff := cmp.Diff(WantResp, gotResponses); diff != "" {
		t.Errorf("Unexpected diff in responses:\n%s\n", diff)
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
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret, nil
}

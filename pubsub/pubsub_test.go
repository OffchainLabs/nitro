package pubsub

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

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

type testRequestMarshaller struct{}

func (t *testRequestMarshaller) Marshal(val string) []byte {
	return []byte(val)
}

func (t *testRequestMarshaller) Unmarshal(val []byte) (string, error) {
	return string(val), nil
}

type testResponseMarshaller struct{}

func (t *testResponseMarshaller) Marshal(val string) []byte {
	return []byte(val)
}

func (t *testResponseMarshaller) Unmarshal(val []byte) (string, error) {
	return string(val), nil
}

func createGroup(ctx context.Context, t *testing.T, streamName, groupName string, client redis.UniversalClient) {
	t.Helper()
	_, err := client.XGroupCreateMkStream(ctx, streamName, groupName, "$").Result()
	if err != nil {
		t.Fatalf("Error creating stream group: %v", err)
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

func newProducerConsumers(ctx context.Context, t *testing.T, opts ...configOpt) (*Producer[string, string], []*Consumer[string, string]) {
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
	producer, err := NewProducer[string, string](prodCfg, &testRequestMarshaller{}, &testResponseMarshaller{})
	if err != nil {
		t.Fatalf("Error creating new producer: %v", err)
	}

	var consumers []*Consumer[string, string]
	for i := 0; i < consumersCount; i++ {
		c, err := NewConsumer[string, string](ctx, consCfg, &testRequestMarshaller{}, &testResponseMarshaller{})
		if err != nil {
			t.Fatalf("Error creating new consumer: %v", err)
		}
		consumers = append(consumers, c)
	}
	createGroup(ctx, t, streamName, groupName, producer.client)
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

func TestRedisProduce(t *testing.T) {
	t.Parallel()
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
					gotMessages[idx][res.ID] = res.Value
					resp := fmt.Sprintf("result for: %v", res.ID)
					if err := c.SetResult(ctx, res.ID, resp); err != nil {
						t.Errorf("Error setting a result: %v", err)
					}
					wantResponses[idx] = append(wantResponses[idx], resp)
				}
			})
	}

	var gotResponses []string

	for i := 0; i < messagesCount; i++ {
		value := fmt.Sprintf("msg: %d", i)
		p, err := producer.Produce(ctx, value)
		if err != nil {
			t.Errorf("Produce() unexpected error: %v", err)
		}
		res, err := p.Await(ctx)
		if err != nil {
			t.Errorf("Await() unexpected error: %v", err)
		}
		gotResponses = append(gotResponses, res)
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

func produceMessages(ctx context.Context, producer *Producer[string, string]) ([]*containers.Promise[string], error) {
	var promises []*containers.Promise[string]
	for i := 0; i < messagesCount; i++ {
		value := fmt.Sprintf("msg: %d", i)
		promise, err := producer.Produce(ctx, value)
		if err != nil {
			return nil, err
		}
		promises = append(promises, promise)
	}
	return promises, nil
}

func awaitResponses(ctx context.Context, promises []*containers.Promise[string]) ([]string, error) {
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
		responses = append(responses, res)
	}
	return responses, errors.Join(errs...)
}

// consume messages from every consumer except every skipNth.
func consume(ctx context.Context, t *testing.T, consumers []*Consumer[string, string], skipN int) ([]map[string]string, [][]string) {
	t.Helper()
	gotMessages := messagesMaps(consumersCount)
	wantResponses := make([][]string, consumersCount)
	for idx := 0; idx < consumersCount; idx++ {
		if idx%skipN == 0 {
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
					gotMessages[idx][res.ID] = res.Value
					resp := fmt.Sprintf("result for: %v", res.ID)
					if err := c.SetResult(ctx, res.ID, resp); err != nil {
						t.Errorf("Error setting a result: %v", err)
					}
					wantResponses[idx] = append(wantResponses[idx], resp)
				}
			})
	}
	return gotMessages, wantResponses
}

func TestRedisClaimingOwnership(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	producer, consumers := newProducerConsumers(ctx, t)
	producer.Start(ctx)
	promises, err := produceMessages(ctx, producer)
	if err != nil {
		t.Fatalf("Error producing messages: %v", err)
	}

	// Consumer messages in every third consumer but don't ack them to check
	// that other consumers will claim ownership on those messages.
	for i := 0; i < len(consumers); i += 3 {
		i := i
		if _, err := consumers[i].Consume(ctx); err != nil {
			t.Errorf("Error consuming message: %v", err)
		}
		consumers[i].StopAndWait()
	}

	gotMessages, wantResponses := consume(ctx, t, consumers, 3)
	gotResponses, err := awaitResponses(ctx, promises)
	if err != nil {
		t.Fatalf("Error awaiting responses: %v", err)
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
	wantResp := flatten(wantResponses)
	sort.Strings(gotResponses)
	if diff := cmp.Diff(wantResp, gotResponses); diff != "" {
		t.Errorf("Unexpected diff in responses:\n%s\n", diff)
	}
	if cnt := producer.promisesLen(); cnt != 0 {
		t.Errorf("Producer still has %d unfullfilled promises", cnt)
	}
}

func TestRedisClaimingOwnershipReproduceDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	producer, consumers := newProducerConsumers(ctx, t, &disableReproduce{})
	producer.Start(ctx)
	promises, err := produceMessages(ctx, producer)
	if err != nil {
		t.Fatalf("Error producing messages: %v", err)
	}

	// Consumer messages in every third consumer but don't ack them to check
	// that other consumers will claim ownership on those messages.
	for i := 0; i < len(consumers); i += 3 {
		i := i
		if _, err := consumers[i].Consume(ctx); err != nil {
			t.Errorf("Error consuming message: %v", err)
		}
		consumers[i].StopAndWait()
	}

	gotMessages, _ := consume(ctx, t, consumers, 3)
	gotResponses, err := awaitResponses(ctx, promises)
	if err == nil {
		t.Fatalf("All promises were fullfilled with reproduce disabled and some consumers killed")
	}
	for _, c := range consumers {
		c.StopWaiter.StopAndWait()
	}
	got, err := mergeValues(gotMessages)
	if err != nil {
		t.Fatalf("mergeMaps() unexpected error: %v", err)
	}
	wantMsgCnt := messagesCount - (consumersCount / 3) - (consumersCount % 3)
	if len(got) != wantMsgCnt {
		t.Fatalf("Got: %d messages, want %d", len(got), wantMsgCnt)
	}
	if len(gotResponses) != wantMsgCnt {
		t.Errorf("Got %d responses want: %d\n", len(gotResponses), wantMsgCnt)
	}
	if cnt := len(producer.promises); cnt != 0 {
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

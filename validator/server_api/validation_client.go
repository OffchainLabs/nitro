package server_api

import (
	"context"
	"encoding/base64"
	"errors"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/pubsub"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api/validation"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

type ValidationClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
	name   string
	room   int32

	producer *pubsub.Producer[*validation.Request, validator.GoGlobalState]
}

type ValidationClientConfig struct {
	RPCClientConfig rpcclient.ClientConfig `koanf:"rpc-client-config"`
	ProducerCfg     pubsub.ProducerConfig  `koanf:"producer-cfg"`
}

var DefaultValidationClientConfig = ValidationClientConfig{
	// TODO: implement.
}

var TestValidationClientConfig = ValidationClientConfig{
	// TODO: implement.
}

func ValidationClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	// TODO: implement.
}

type ValidationClientConfigFetcher func() *ValidationClientConfig

func NewValidationClient(config ValidationClientConfigFetcher, stack *node.Node) *ValidationClient {
	rpcClientCfgFetcher := func() *rpcclient.ClientConfig {
		return &config().RPCClientConfig
	}
	ret := &ValidationClient{
		client: rpcclient.NewRpcClient(rpcClientCfgFetcher, stack),
	}
	producer, err := pubsub.NewProducer[*validation.Request, validator.GoGlobalState](
		&config().ProducerCfg,
	)
	if err == nil {
		ret.producer = producer
	} else if config().ProducerCfg.RedisURL != "" {
		log.Error("Error creating producer for validation client", "error", err)
	}
	return ret
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	atomic.AddInt32(&c.room, -1)
	input := ValidationInputToJson(entry)
	if c.producer == nil {
		promise := stopwaiter.LaunchPromiseThread[validator.GoGlobalState](c, func(ctx context.Context) (validator.GoGlobalState, error) {
			var res validator.GoGlobalState
			err := c.client.CallContext(ctx, &res, Namespace+"_validate", input, moduleRoot)
			atomic.AddInt32(&c.room, 1)
			return res, err
		})
		return server_common.NewValRun(promise, moduleRoot)
	}

	var (
		promise containers.PromiseInterface[validator.GoGlobalState]
		err     error
	)
	promise, err = c.producer.Produce(c.GetContext(), &validation.Request{
		Input:      input,
		ModuleRoot: moduleRoot,
	})
	if err != nil {
		promise = containers.NewReadyPromise(validator.GoGlobalState{}, err)
	}
	atomic.AddInt32(&c.room, 1)
	return server_common.NewValRun(promise, moduleRoot)
}

func (c *ValidationClient) Start(ctx_in context.Context) error {
	if c.producer != nil {
		c.producer.Start(ctx_in)
	}
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	err := c.client.Start(ctx)
	if err != nil {
		return err
	}
	var name string
	err = c.client.CallContext(ctx, &name, Namespace+"_name")
	if err != nil {
		return err
	}
	if len(name) == 0 {
		return errors.New("couldn't read name from server")
	}
	var room int
	err = c.client.CallContext(c.GetContext(), &room, Namespace+"_room")
	if err != nil {
		return err
	}
	if room < 2 {
		log.Warn("validation server not enough room, overriding to 2", "name", name, "room", room)
		room = 2
	} else {
		log.Info("connected to validation server", "name", name, "room", room)
	}
	atomic.StoreInt32(&c.room, int32(room))
	c.name = name
	return nil
}

func (c *ValidationClient) Stop() {
	if c.producer != nil {
		c.producer.StopAndWait()
	}
	c.StopWaiter.StopOnly()
	if c.client != nil {
		c.client.Close()
	}
}

func (c *ValidationClient) Name() string {
	if c.Started() {
		return c.name
	}
	return "(not started)"
}

func (c *ValidationClient) Room() int {
	room32 := atomic.LoadInt32(&c.room)
	if room32 < 0 {
		return 0
	}
	return int(room32)
}

type ExecutionClient struct {
	ValidationClient
}

func NewExecutionClient(config ValidationClientConfigFetcher, stack *node.Node) *ExecutionClient {
	return &ExecutionClient{
		ValidationClient: *NewValidationClient(config, stack),
	}
}

func (c *ExecutionClient) CreateExecutionRun(wasmModuleRoot common.Hash, input *validator.ValidationInput) containers.PromiseInterface[validator.ExecutionRun] {
	return stopwaiter.LaunchPromiseThread[validator.ExecutionRun](c, func(ctx context.Context) (validator.ExecutionRun, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, Namespace+"_createExecutionRun", wasmModuleRoot, ValidationInputToJson(input))
		if err != nil {
			return nil, err
		}
		run := &ExecutionClientRun{
			client: c,
			id:     res,
		}
		run.Start(c.GetContext()) // note: not this temporary thread's context!
		return run, nil
	})
}

type ExecutionClientRun struct {
	stopwaiter.StopWaiter
	client *ExecutionClient
	id     uint64
}

func (c *ExecutionClient) LatestWasmModuleRoot() containers.PromiseInterface[common.Hash] {
	return stopwaiter.LaunchPromiseThread[common.Hash](c, func(ctx context.Context) (common.Hash, error) {
		var res common.Hash
		err := c.client.CallContext(ctx, &res, Namespace+"_latestWasmModuleRoot")
		if err != nil {
			return common.Hash{}, err
		}
		return res, nil
	})
}

func (c *ExecutionClient) WriteToFile(input *validator.ValidationInput, expOut validator.GoGlobalState, moduleRoot common.Hash) containers.PromiseInterface[struct{}] {
	jsonInput := ValidationInputToJson(input)
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, Namespace+"_writeToFile", jsonInput, expOut, moduleRoot)
		return struct{}{}, err
	})
}

func (r *ExecutionClientRun) SendKeepAlive(ctx context.Context) time.Duration {
	err := r.client.client.CallContext(ctx, nil, Namespace+"_execKeepAlive", r.id)
	if err != nil {
		log.Error("execution run keepalive failed", "err", err)
	}
	return time.Minute // TODO: configurable
}

func (r *ExecutionClientRun) Start(ctx_in context.Context) {
	r.StopWaiter.Start(ctx_in, r)
	r.CallIteratively(r.SendKeepAlive)
}

func (r *ExecutionClientRun) GetStepAt(pos uint64) containers.PromiseInterface[*validator.MachineStepResult] {
	return stopwaiter.LaunchPromiseThread[*validator.MachineStepResult](r, func(ctx context.Context) (*validator.MachineStepResult, error) {
		var resJson MachineStepResultJson
		err := r.client.client.CallContext(ctx, &resJson, Namespace+"_getStepAt", r.id, pos)
		if err != nil {
			return nil, err
		}
		res, err := MachineStepResultFromJson(&resJson)
		if err != nil {
			return nil, err
		}
		return res, err
	})
}

func (r *ExecutionClientRun) GetProofAt(pos uint64) containers.PromiseInterface[[]byte] {
	return stopwaiter.LaunchPromiseThread[[]byte](r, func(ctx context.Context) ([]byte, error) {
		var resString string
		err := r.client.client.CallContext(ctx, &resString, Namespace+"_getProofAt", r.id, pos)
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(resString)
	})
}

func (r *ExecutionClientRun) GetLastStep() containers.PromiseInterface[*validator.MachineStepResult] {
	return r.GetStepAt(^uint64(0))
}

func (r *ExecutionClientRun) PrepareRange(start, end uint64) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](r, func(ctx context.Context) (struct{}, error) {
		err := r.client.client.CallContext(ctx, nil, Namespace+"_prepareRange", r.id, start, end)
		if err != nil && ctx.Err() == nil {
			log.Warn("prepare execution got error", "err", err)
		}
		return struct{}{}, err
	})
}

func (r *ExecutionClientRun) Close() {
	r.StopOnly()
	r.LaunchUntrackedThread(func() {
		err := r.client.client.CallContext(r.GetParentContext(), nil, Namespace+"_closeExec", r.id)
		if err != nil {
			log.Warn("closing execution client run got error", "err", err, "client", r.client.Name(), "id", r.id)
		}
	})
}

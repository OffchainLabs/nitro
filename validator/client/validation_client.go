package client

import (
	"context"
	"encoding/base64"
	"errors"
	"sync/atomic"
	"time"

	"github.com/offchainlabs/nitro/validator"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"

	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

type ValidationClient struct {
	stopwaiter.StopWaiter
	client *rpcclient.RpcClient
	name   string
	room   int32
}

func NewValidationClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ValidationClient {
	return &ValidationClient{
		client: rpcclient.NewRpcClient(config, stack),
	}
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	atomic.AddInt32(&c.room, -1)
	promise := stopwaiter.LaunchPromiseThread[validator.GoGlobalState](c, func(ctx context.Context) (validator.GoGlobalState, error) {
		input := server_api.ValidationInputToJson(entry)
		var res validator.GoGlobalState
		err := c.client.CallContext(ctx, &res, server_api.Namespace+"_validate", input, moduleRoot)
		atomic.AddInt32(&c.room, 1)
		return res, err
	})
	return server_common.NewValRun(promise, moduleRoot)
}

func (c *ValidationClient) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	if err := c.client.Start(ctx); err != nil {
		return err
	}
	var name string
	if err := c.client.CallContext(ctx, &name, server_api.Namespace+"_name"); err != nil {
		return err
	}
	if len(name) == 0 {
		return errors.New("couldn't read name from server")
	}
	var room int
	if err := c.client.CallContext(c.GetContext(), &room, server_api.Namespace+"_room"); err != nil {
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
	c.StopWaiter.StopOnly()
	if c.client != nil {
		c.client.Close()
	}
}

func (c *ValidationClient) Name() string {
	return c.name
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

func NewExecutionClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ExecutionClient {
	return &ExecutionClient{
		ValidationClient: *NewValidationClient(config, stack),
	}
}

func (c *ExecutionClient) CreateExecutionRun(wasmModuleRoot common.Hash, input *validator.ValidationInput) containers.PromiseInterface[validator.ExecutionRun] {
	return stopwaiter.LaunchPromiseThread[validator.ExecutionRun](c, func(ctx context.Context) (validator.ExecutionRun, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, server_api.Namespace+"_createExecutionRun", wasmModuleRoot, server_api.ValidationInputToJson(input))
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
		err := c.client.CallContext(ctx, &res, server_api.Namespace+"_latestWasmModuleRoot")
		if err != nil {
			return common.Hash{}, err
		}
		return res, nil
	})
}

func (c *ExecutionClient) WriteToFile(input *validator.ValidationInput, expOut validator.GoGlobalState, moduleRoot common.Hash) containers.PromiseInterface[struct{}] {
	jsonInput := server_api.ValidationInputToJson(input)
	return stopwaiter.LaunchPromiseThread[struct{}](c, func(ctx context.Context) (struct{}, error) {
		err := c.client.CallContext(ctx, nil, server_api.Namespace+"_writeToFile", jsonInput, expOut, moduleRoot)
		return struct{}{}, err
	})
}

func (r *ExecutionClientRun) SendKeepAlive(ctx context.Context) time.Duration {
	err := r.client.client.CallContext(ctx, nil, server_api.Namespace+"_execKeepAlive", r.id)
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
		var resJson server_api.MachineStepResultJson
		err := r.client.client.CallContext(ctx, &resJson, server_api.Namespace+"_getStepAt", r.id, pos)
		if err != nil {
			return nil, err
		}
		res, err := server_api.MachineStepResultFromJson(&resJson)
		if err != nil {
			return nil, err
		}
		return res, err
	})
}

func (r *ExecutionClientRun) GetProofAt(pos uint64) containers.PromiseInterface[[]byte] {
	return stopwaiter.LaunchPromiseThread[[]byte](r, func(ctx context.Context) ([]byte, error) {
		var resString string
		err := r.client.client.CallContext(ctx, &resString, server_api.Namespace+"_getProofAt", r.id, pos)
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
		err := r.client.client.CallContext(ctx, nil, server_api.Namespace+"_prepareRange", r.id, start, end)
		if err != nil && ctx.Err() == nil {
			log.Warn("prepare execution got error", "err", err)
		}
		return struct{}{}, err
	})
}

func (r *ExecutionClientRun) Close() {
	r.StopOnly()
	r.LaunchUntrackedThread(func() {
		err := r.client.client.CallContext(r.GetParentContext(), nil, server_api.Namespace+"_closeExec", r.id)
		if err != nil {
			log.Warn("closing execution client run got error", "err", err, "client", r.client.Name(), "id", r.id)
		}
	})
}

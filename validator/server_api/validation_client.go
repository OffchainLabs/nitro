package server_api

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/offchainlabs/nitro/validator"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"

	"github.com/offchainlabs/nitro/validator/server_common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type ValidationClient struct {
	stopwaiter.StopWaiter
	client    *rpc.Client
	url       string
	name      string
	jwtSecret []byte
}

func NewValidationClient(url string, jwtSecret []byte) *ValidationClient {
	return &ValidationClient{
		url:       url,
		jwtSecret: jwtSecret,
	}
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	valrun := server_common.NewValRun(moduleRoot)
	c.LaunchThread(func(ctx context.Context) {
		input := ValidationInputToJson(entry)
		var res validator.GoGlobalState
		err := c.client.CallContext(ctx, &res, Namespace+"_validate", input, moduleRoot)
		valrun.ConsumeResult(res, err)
	})
	return valrun
}

func (c *ValidationClient) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	var client *rpc.Client
	var err error
	if len(c.jwtSecret) == 0 {
		client, err = rpc.DialWebsocket(ctx, c.url, "")
	} else {
		client, err = rpc.DialWebsocketJWT(ctx, c.url, "", c.jwtSecret)
	}
	if err != nil {
		return err
	}
	var name string
	err = client.CallContext(ctx, &name, Namespace+"_name")
	if err != nil {
		return err
	}
	if len(name) == 0 {
		return errors.New("couldn't read name from server")
	}
	c.client = client
	c.name = name + " on " + c.url
	return nil
}

func (c *ValidationClient) Stop() {
	c.StopWaiter.StopOnly()
	if c.client != nil {
		c.client.Close()
	}
}

func (c *ValidationClient) Name() string {
	if c.Started() {
		return c.name
	}
	return "(not started) on " + c.url
}

func (c *ValidationClient) Room() int {
	var res int
	err := c.client.CallContext(c.GetContext(), &res, Namespace+"_room")
	if err != nil {
		log.Error("error contacting validation server", "name", c.name, "err", err)
		return 0
	}
	return res
}

type ExecutionClient struct {
	ValidationClient
}

func NewExecutionClient(url string, jwtSecret []byte) *ExecutionClient {
	return &ExecutionClient{
		ValidationClient: *NewValidationClient(url, jwtSecret),
	}
}

func (c *ExecutionClient) CreateExecutionRun(wasmModuleRoot common.Hash, input *validator.ValidationInput) containers.PromiseInterface[validator.ExecutionRun] {
	promise := containers.NewPromise[validator.ExecutionRun]()
	cancel := c.LaunchThreadWithCancel(func(ctx context.Context) {
		var res uint64
		err := c.client.CallContext(ctx, &res, Namespace+"_createExecutionRun", wasmModuleRoot, ValidationInputToJson(input))
		if err != nil {
			promise.ProduceError(err)
			return
		}
		run := &ExecutionClientRun{
			client: c,
			id:     res,
		}
		run.Start(c.GetContext()) // note: not this temporary thread's context!
		promise.Produce(run)
	})
	promise.SetCancel(cancel)
	return &promise
}

type ExecutionClientRun struct {
	stopwaiter.StopWaiter
	client *ExecutionClient
	id     uint64
}

func (c *ExecutionClient) LatestWasmModuleRoot() containers.PromiseInterface[common.Hash] {
	promise := containers.NewPromise[common.Hash]()
	cancel := c.LaunchThreadWithCancel(func(ctx context.Context) {
		var res common.Hash
		err := c.client.CallContext(c.GetContext(), &res, Namespace+"_latestWasmModuleRoot")
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(res)
	})
	promise.SetCancel(cancel)
	return &promise
}

func (c *ExecutionClient) WriteToFile(input *validator.ValidationInput, expOut validator.GoGlobalState, moduleRoot common.Hash) containers.PromiseInterface[struct{}] {
	jsonInput := ValidationInputToJson(input)
	promise := containers.NewPromise[struct{}]()
	cancel := c.LaunchThreadWithCancel(func(ctx context.Context) {
		err := c.client.CallContext(ctx, nil, Namespace+"_writeToFile", jsonInput, expOut, moduleRoot)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(struct{}{})
	})
	promise.SetCancel(cancel)
	return &promise

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

func (r *ExecutionClientRun) GetStepAt(pos uint64) containers.PromiseInterface[validator.MachineStepResult] {
	step := containers.NewPromise[validator.MachineStepResult]()
	cancel := r.LaunchThreadWithCancel(func(ctx context.Context) {
		var resJson MachineStepResultJson
		err := r.client.client.CallContext(ctx, &resJson, Namespace+"_getStepAt", r.id, pos)
		if err != nil {
			step.ProduceError(err)
			return
		}
		res, err := MachineStepResultFromJson(&resJson)
		if err != nil {
			step.ProduceError(err)
			return
		}
		step.Produce(*res)
	})
	step.SetCancel(cancel)
	return &step
}

func (r *ExecutionClientRun) GetProofAt(pos uint64) containers.PromiseInterface[[]byte] {
	proof := containers.NewPromise[[]byte]()
	cancel := r.LaunchThreadWithCancel(func(ctx context.Context) {
		var resString string
		err := r.client.client.CallContext(ctx, &resString, Namespace+"_getProofAt", r.id, pos)
		if err != nil {
			proof.ProduceError(err)
			return
		}
		res, err := base64.StdEncoding.DecodeString(resString)
		if err != nil {
			proof.ProduceError(err)
			return
		}
		proof.Produce(res)
	})
	proof.SetCancel(cancel)
	return &proof
}

func (r *ExecutionClientRun) GetLastStep() containers.PromiseInterface[validator.MachineStepResult] {
	return r.GetStepAt(^uint64(0))
}

func (r *ExecutionClientRun) PrepareRange(start, end uint64) {
	r.LaunchUntrackedThread(func() {
		err := r.client.client.CallContext(r.client.GetContext(), nil, Namespace+"_prepareRange", r.id, start, end)
		if err != nil {
			log.Warn("prepare execution got error", "err", err)
		}
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

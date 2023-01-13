package server_api

import (
	"context"
	"errors"

	"github.com/offchainlabs/nitro/validator"

	"github.com/offchainlabs/nitro/util/readymarker"
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
	c.client.Close()
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

func (c *ExecutionClient) CreateExecutionRun(wasmModuleRoot common.Hash, input *validator.ValidationInput) (validator.ExecutionRun, error) {
	var res uint64
	err := c.client.CallContext(c.GetContext(), &res, Namespace+"_createExecutionRun")
	if err != nil {
		return nil, err
	}
	return &ExecutionClientRun{c, res}, nil
}

type ExecutionClientRun struct {
	client *ExecutionClient
	id     uint64
}

func (c *ExecutionClient) LatestWasmModuleRoot() (common.Hash, error) {
	var res common.Hash
	err := c.client.CallContext(c.GetContext(), &res, Namespace+"_latestWasmModuleRoot")
	return res, err
}

func (c *ExecutionClient) WriteToFile(input *validator.ValidationInput, expOut validator.GoGlobalState, moduleRoot common.Hash) error {
	jsonInput := ValidationInputToJson(input)
	err := c.client.CallContext(c.GetContext(), nil, Namespace+"_writeToFile", jsonInput, expOut, moduleRoot)
	return err
}

type ExecutionClientStep struct {
	readymarker.ReadyMarker
	result validator.MachineStepResult
	cancel func()
}

func (r *ExecutionClientRun) GetStepAt(pos uint64) validator.MachineStep {
	ctx, cancel := context.WithCancel(r.client.GetContext())
	step := &ExecutionClientStep{
		ReadyMarker: readymarker.NewReadyMarker(),
		cancel:      cancel,
	}
	go func() {
		var resJson MachineStepResultJson
		err := r.client.client.CallContext(ctx, &resJson, Namespace+"_getStepAt", r.id, pos)
		if err != nil {
			step.SignalReady(err)
			return
		}
		res, err := MachineStepResultFromJson(&resJson)
		if err != nil {
			step.SignalReady(err)
			return
		}
		step.result = *res
		step.SignalReady(nil)
	}()
	return step
}

func (r *ExecutionClientRun) GetLastStep() validator.MachineStep {
	return r.GetStepAt(^uint64(0))
}

func (r *ExecutionClientRun) PrepareRange(start, end uint64) {
	go func() {
		err := r.client.client.CallContext(r.client.GetContext(), nil, Namespace+"_prepareRange", r.id, start, end)
		if err != nil {
			log.Warn("prepare execution got error", "err", err)
		}
	}()
}

func (r *ExecutionClientRun) Close() {
	go func() {
		err := r.client.client.CallContext(r.client.GetContext(), nil, Namespace+"_closeExec", r.id)
		if err != nil {
			log.Warn("closing execution client run got error", "err", err, "client", r.client.Name(), "id", r.id)
		}
	}()
}

func (f *ExecutionClientStep) Close() {
	f.cancel()
}

func (f *ExecutionClientStep) Get() (*validator.MachineStepResult, error) {
	err := f.TestReady()
	if err != nil {
		return nil, err
	}
	return &f.result, nil
}

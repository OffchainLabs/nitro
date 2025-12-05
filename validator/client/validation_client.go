// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_common"
)

var executionNodeOfflineCounter = metrics.NewRegisteredCounter("arb/state_provider/execution_node_offline", nil)

type ValidationClient struct {
	stopwaiter.StopWaiter
	client          *rpcclient.RpcClient
	name            string
	stylusArchs     []rawdb.WasmTarget
	capacity        int
	wasmModuleRoots []common.Hash
}

func NewValidationClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ValidationClient {
	return &ValidationClient{
		client:      rpcclient.NewRpcClient(config, stack),
		name:        "not started",
		stylusArchs: []rawdb.WasmTarget{"not started"},
	}
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	promise := stopwaiter.LaunchPromiseThread[validator.GoGlobalState](c, func(ctx context.Context) (validator.GoGlobalState, error) {
		input := server_api.ValidationInputToJson(entry)
		var res validator.GoGlobalState
		err := c.client.CallContext(ctx, &res, server_api.Namespace+"_validate", input, moduleRoot)
		return res, err
	})
	return server_common.NewValRun(promise, moduleRoot)
}

func (c *ValidationClient) Start(ctx context.Context) error {
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
	var stylusArchs []rawdb.WasmTarget
	if err := c.client.CallContext(ctx, &stylusArchs, server_api.Namespace+"_stylusArchs"); err != nil {
		return fmt.Errorf("could not read stylus arch from server: %w", err)
	} else {
		if len(stylusArchs) == 0 {
			return fmt.Errorf("could not read stylus archs from validation server")
		}
		for _, stylusArch := range stylusArchs {
			if !rawdb.IsSupportedWasmTarget(rawdb.WasmTarget(stylusArch)) && stylusArch != "mock" {
				return fmt.Errorf("unsupported stylus architecture: %v", stylusArch)
			}
		}
	}
	var moduleRoots []common.Hash
	if err := c.client.CallContext(ctx, &moduleRoots, server_api.Namespace+"_wasmModuleRoots"); err != nil {
		return err
	}
	if len(moduleRoots) == 0 {
		return fmt.Errorf("server reported no wasmModuleRoots")
	}
	var spawnerCapacity int
	if err := c.client.CallContext(ctx, &spawnerCapacity, server_api.Namespace+"_capacity"); err != nil {
		// handle the forward compatibility case where the server doesn't have the capacity method
		log.Warn("could not get capacity from validation server, use room method instead", "err", err)
		if err := c.client.CallContext(ctx, &spawnerCapacity, server_api.Namespace+"_room"); err != nil {
			return err
		}
	}
	if spawnerCapacity < 2 {
		log.Warn("validation server not enough workers, overriding to 2", "name", name, "maxWorkers", spawnerCapacity)
		spawnerCapacity = 2
	} else {
		log.Info("connected to validation server", "name", name, "maxWorkers", spawnerCapacity)
	}
	c.capacity = spawnerCapacity
	c.wasmModuleRoots = moduleRoots
	c.name = name
	c.stylusArchs = stylusArchs
	c.StopWaiter.Start(ctx, c)
	return nil
}

func (c *ValidationClient) WasmModuleRoots() ([]common.Hash, error) {
	if c.Started() {
		return c.wasmModuleRoots, nil
	}
	return nil, errors.New("not started")
}

func (c *ValidationClient) StylusArchs() []rawdb.WasmTarget {
	if c.Started() {
		return c.stylusArchs
	}
	return []rawdb.WasmTarget{"not started"}
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

func (c *ValidationClient) Capacity() int {
	return c.capacity
}

type ExecutionClient struct {
	ValidationClient
}

func NewExecutionClient(config rpcclient.ClientConfigFetcher, stack *node.Node) *ExecutionClient {
	return &ExecutionClient{
		ValidationClient: *NewValidationClient(config, stack),
	}
}

func (c *ExecutionClient) CreateExecutionRun(
	wasmModuleRoot common.Hash,
	input *validator.ValidationInput,
	useBoldMachine bool,
) containers.PromiseInterface[validator.ExecutionRun] {
	return stopwaiter.LaunchPromiseThread(c, func(ctx context.Context) (validator.ExecutionRun, error) {
		var res uint64
		err := c.client.CallContext(ctx, &res, server_api.Namespace+"_createExecutionRun", wasmModuleRoot, server_api.ValidationInputToJson(input), useBoldMachine)
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

var _ validator.BOLDExecutionSpawner = (*BOLDExecutionClient)(nil)

type BOLDExecutionClient struct {
	executionSpawner validator.ExecutionSpawner
}

func NewBOLDExecutionClient(executionSpawner validator.ExecutionSpawner) *BOLDExecutionClient {
	return &BOLDExecutionClient{
		executionSpawner: executionSpawner,
	}
}

func (b *BOLDExecutionClient) WasmModuleRoots() ([]common.Hash, error) {
	return b.executionSpawner.WasmModuleRoots()
}

func (b *BOLDExecutionClient) GetMachineHashesWithStepSize(ctx context.Context, wasmModuleRoot common.Hash, input *validator.ValidationInput, machineStartIndex, stepSize, maxIterations uint64) ([]common.Hash, error) {
	execRun, err := b.executionSpawner.CreateExecutionRun(wasmModuleRoot, input, true).Await(ctx)
	if err != nil {
		return nil, err
	}
	defer execRun.Close()
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	stepLeaves := execRun.GetMachineHashesWithStepSize(machineStartIndex, stepSize, maxIterations)
	return stepLeaves.Await(ctxCheckAlive)
}

func (b *BOLDExecutionClient) GetProofAt(ctx context.Context, wasmModuleRoot common.Hash, input *validator.ValidationInput, position uint64) ([]byte, error) {
	execRun, err := b.executionSpawner.CreateExecutionRun(wasmModuleRoot, input, true).Await(ctx)
	if err != nil {
		return nil, err
	}
	defer execRun.Close()
	ctxCheckAlive, cancelCheckAlive := ctxWithCheckAlive(ctx, execRun)
	defer cancelCheckAlive()
	oneStepProofPromise := execRun.GetProofAt(position)
	return oneStepProofPromise.Await(ctxCheckAlive)
}

// CtxWithCheckAlive Creates a context with a check alive routine that will
// cancel the context if the check alive routine fails.
func ctxWithCheckAlive(ctxIn context.Context, execRun validator.ExecutionRun) (context.Context, context.CancelFunc) {
	// Create a context that will cancel if the check alive routine fails.
	// This is to ensure that we do not have the validator froze indefinitely if
	// the execution run is no longer alive.
	ctx, cancel := context.WithCancel(ctxIn)
	go func() {
		// Call cancel so that the calling function is canceled if the check alive
		// routine fails/returns.
		defer cancel()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Create a context with a timeout, so that the check alive routine does
				// not run indefinitely.
				ctxCheckAliveWithTimeout, cancelCheckAliveWithTimeout := context.WithTimeout(ctx, 5*time.Second)
				err := execRun.CheckAlive(ctxCheckAliveWithTimeout)
				if err != nil {
					executionNodeOfflineCounter.Inc(1)
					cancelCheckAliveWithTimeout()
					return
				}
				cancelCheckAliveWithTimeout()
			}
		}
	}()
	return ctx, cancel
}

type ExecutionClientRun struct {
	stopwaiter.StopWaiter
	client *ExecutionClient
	id     uint64
}

func (r *ExecutionClientRun) SendKeepAlive(ctx context.Context) time.Duration {
	err := r.client.client.CallContext(ctx, nil, server_api.Namespace+"_execKeepAlive", r.id)
	if err != nil {
		log.Error("execution run keepalive failed", "err", err)
	}
	return time.Minute // TODO: configurable
}

func (r *ExecutionClientRun) CheckAlive(ctx context.Context) error {
	return r.client.client.CallContext(ctx, nil, server_api.Namespace+"_checkAlive", r.id)
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

func (r *ExecutionClientRun) GetMachineHashesWithStepSize(machineStartIndex, stepSize, maxIterations uint64) containers.PromiseInterface[[]common.Hash] {
	return stopwaiter.LaunchPromiseThread[[]common.Hash](r, func(ctx context.Context) ([]common.Hash, error) {
		var resJson []common.Hash
		err := r.client.client.CallContext(ctx, &resJson, server_api.Namespace+"_getMachineHashesWithStepSize", r.id, machineStartIndex, stepSize, maxIterations)
		if err != nil {
			return nil, err
		}
		return resJson, err
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

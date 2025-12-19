package validator

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/util/containers"
)

type ValidationSpawner interface {
	Launch(entry *ValidationInput, moduleRoot common.Hash) ValidationRun
	WasmModuleRoots() ([]common.Hash, error)
	Start(context.Context) error
	Stop()
	Name() string
	StylusArchs() []rawdb.WasmTarget
	// This is a static number representing the maximum number of workers, should not change over time.
	// block_validator uses this to size its worker pool.
	Capacity() int
}

type ValidationRun interface {
	containers.PromiseInterface[GoGlobalState]
	WasmModuleRoot() common.Hash
}

type ExecutionSpawner interface {
	ValidationSpawner
	CreateExecutionRun(wasmModuleRoot common.Hash, input *ValidationInput, useBoldMachine bool) containers.PromiseInterface[ExecutionRun]
}

type BOLDExecutionSpawner interface {
	WasmModuleRoots() ([]common.Hash, error)
	GetMachineHashesWithStepSize(ctx context.Context, wasmModuleRoot common.Hash, input *ValidationInput, machineStartIndex, stepSize, maxIterations uint64) ([]common.Hash, error)
	GetProofAt(ctx context.Context, wasmModuleRoot common.Hash, input *ValidationInput, position uint64) ([]byte, error)
}

type ExecutionRun interface {
	GetStepAt(uint64) containers.PromiseInterface[*MachineStepResult]
	GetMachineHashesWithStepSize(machineStartIndex, stepSize, maxIterations uint64) containers.PromiseInterface[[]common.Hash]
	GetLastStep() containers.PromiseInterface[*MachineStepResult]
	GetProofAt(uint64) containers.PromiseInterface[[]byte]
	PrepareRange(uint64, uint64) containers.PromiseInterface[struct{}]
	Close()
	CheckAlive(ctx context.Context) error
}

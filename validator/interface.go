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
	StylusArchs() []rawdb.Target
	Room() int
}

type ValidationRun interface {
	containers.PromiseInterface[GoGlobalState]
	WasmModuleRoot() common.Hash
}

type ExecutionSpawner interface {
	ValidationSpawner
	CreateExecutionRun(wasmModuleRoot common.Hash, input *ValidationInput) containers.PromiseInterface[ExecutionRun]
	LatestWasmModuleRoot() containers.PromiseInterface[common.Hash]
	WriteToFile(input *ValidationInput, expOut GoGlobalState, moduleRoot common.Hash) containers.PromiseInterface[struct{}]
}

type ExecutionRun interface {
	GetStepAt(uint64) containers.PromiseInterface[*MachineStepResult]
	GetMachineHashesWithStepSize(machineStartIndex, stepSize, maxIterations uint64) containers.PromiseInterface[[]common.Hash]
	GetLastStep() containers.PromiseInterface[*MachineStepResult]
	GetProofAt(uint64) containers.PromiseInterface[[]byte]
	PrepareRange(uint64, uint64) containers.PromiseInterface[struct{}]
	Close()
}

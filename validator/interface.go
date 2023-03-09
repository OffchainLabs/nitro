package validator

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/containers"
)

type ValidationSpawner interface {
	Launch(entry *ValidationInput, moduleRoot common.Hash) ValidationRun
	Start(context.Context) error
	Stop()
	Name() string
	Room() int
}

type ValidationRun interface {
	containers.PromiseInterface[GoGlobalState]
	WasmModuleRoot() common.Hash
	Close()
}

type ExecutionSpawner interface {
	ValidationSpawner
	CreateExecutionRun(wasmModuleRoot common.Hash, input *ValidationInput) (ExecutionRun, error)
	LatestWasmModuleRoot() (common.Hash, error)
	WriteToFile(input *ValidationInput, expOut GoGlobalState, moduleRoot common.Hash) error
}

type ExecutionRun interface {
	GetStepAt(uint64) MachineStep
	GetLastStep() MachineStep
	GetProofAt(uint64) ProofPromise
	PrepareRange(uint64, uint64)
	Close()
}

type ProofPromise interface {
	containers.PromiseInterface[[]byte]
	Close()
}

type MachineStep interface {
	containers.PromiseInterface[MachineStepResult]
	Close()
}

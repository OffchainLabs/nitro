package validator

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	state_hashes "github.com/OffchainLabs/bold/state-commitments/state-hashes"

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
}

type ExecutionSpawner interface {
	ValidationSpawner
	CreateExecutionRun(wasmModuleRoot common.Hash, input *ValidationInput) containers.PromiseInterface[ExecutionRun]
	CreateBoldExecutionRun(wasmModuleRoot common.Hash, stepSize uint64, input *ValidationInput) containers.PromiseInterface[ExecutionRun]
	LatestWasmModuleRoot() containers.PromiseInterface[common.Hash]
	WriteToFile(input *ValidationInput, expOut GoGlobalState, moduleRoot common.Hash) containers.PromiseInterface[struct{}]
}

type ExecutionRun interface {
	GetStepAt(uint64) containers.PromiseInterface[*MachineStepResult]
	GetLeavesWithStepSize(machineStartIndex, stepSize, numDesiredLeaves uint64) containers.PromiseInterface[*state_hashes.StateHashes]
	GetLastStep() containers.PromiseInterface[*MachineStepResult]
	GetProofAt(uint64) containers.PromiseInterface[[]byte]
	PrepareRange(uint64, uint64) containers.PromiseInterface[struct{}]
	Close()
	CheckAlive(ctx context.Context) error
}

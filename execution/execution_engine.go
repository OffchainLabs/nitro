package execution

import (
	"encoding/binary"
	"errors"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Capsule API description:
//
// BlockGenerator generates the blocks that make up a chain.
// NewBlockGenerator(maxInstructionsPerBlock uint64) creates a block generator
//      each block will use up to maxInstructionsPerBlock instructions of execution (randomly varying)
//
// blockGenerator.BlockHash(blockNum) gets the block hash of blockNum
// blockGenerator.NewExecutionEngine(blockNum) creates an execution engine for the state transition function
//      execution that creates blockNum (starting with the state at blockNum-1
//
// executionEngine.NumSteps() returns the number of steps of execution to create that block
// executionEngine.StateAfter(num) returns the ExecutionState after executing num instructions (or error)
//
// executionState.Hash() gets the state root of executionState
// executionState.NextState() gets the execution state after executing one instruction from executionState
// executionState.OneStepProof() generates a one-step proof for executing one instruction from executionState
//
// VerifyOneStepProof(beforeStateRoot, claimedAfterStateRoot, proof) verifies a one-step proof

var OutOfBoundsError = errors.New("instruction number out of bounds")

type BlockGenerator struct {
	stateRoots              []common.Hash
	maxInstructionsPerBlock uint64
}

func NewBlockGenerator(maxInstructionsPerBlock uint64) *BlockGenerator {
	return &BlockGenerator{
		stateRoots:              []common.Hash{util.HashForUint(0)},
		maxInstructionsPerBlock: maxInstructionsPerBlock,
	}
}

func (gen *BlockGenerator) BlockHash(blockNum uint64) common.Hash {
	for uint64(len(gen.stateRoots)) <= blockNum {
		gen.stateRoots = append(
			gen.stateRoots,
			crypto.Keccak256Hash(gen.stateRoots[len(gen.stateRoots)-1].Bytes()),
		)
	}
	return gen.stateRoots[blockNum]
}

type ExecEngineConfig struct {
	NumSteps          uint64
	RandomizeNumSteps bool
}

func DefaultEngineConfig() *ExecEngineConfig {
	return &ExecEngineConfig{
		RandomizeNumSteps: true,
	}
}

func (gen *BlockGenerator) NewExecutionEngine(blockNum uint64, cfg *ExecEngineConfig) (*ExecutionEngine, error) {
	if blockNum == 0 {
		return nil, errors.New("tried to make execution engine for genesis block")
	}
	startStateRoot := gen.BlockHash(blockNum - 1)
	endStateRoot := gen.BlockHash(blockNum)
	var numSteps uint64
	if cfg == nil || cfg.RandomizeNumSteps {
		numSteps = binary.BigEndian.Uint64(crypto.Keccak256(startStateRoot.Bytes())[:8]) % (1 + gen.maxInstructionsPerBlock)
	} else {
		numSteps = cfg.NumSteps
	}
	if numSteps == 0 {
		return nil, errors.New("must have at least one step of execution")
	}
	return &ExecutionEngine{
		startStateRoot: startStateRoot,
		endStateRoot:   endStateRoot,
		numSteps:       numSteps,
	}, nil
}

type ExecutionEngine struct {
	startStateRoot common.Hash
	endStateRoot   common.Hash
	numSteps       uint64
}

func (engine *ExecutionEngine) serialize() []byte {
	ret := []byte{}
	ret = append(ret, engine.startStateRoot.Bytes()...)
	ret = append(ret, engine.endStateRoot.Bytes()...)
	ret = append(ret, binary.BigEndian.AppendUint64([]byte{}, engine.numSteps)...)
	return ret
}

func deserializeExecutionEngine(buf []byte) (*ExecutionEngine, error) {
	if len(buf) != 32+32+8 {
		return nil, errors.New("deserialization error")
	}
	return &ExecutionEngine{
		startStateRoot: common.BytesToHash(buf[:32]),
		endStateRoot:   common.BytesToHash(buf[32:64]),
		numSteps:       binary.BigEndian.Uint64(buf[64:]),
	}, nil
}

func (engine *ExecutionEngine) internalHash() common.Hash {
	return crypto.Keccak256Hash(engine.serialize())
}

type ExecutionState struct {
	engine  *ExecutionEngine
	stepNum uint64
}

func (engine *ExecutionEngine) NumSteps() uint64 {
	return engine.numSteps
}

func (engine *ExecutionEngine) StateAfter(num uint64) (*ExecutionState, error) {
	if num > engine.numSteps {
		return nil, OutOfBoundsError
	}
	return &ExecutionState{
		engine:  engine,
		stepNum: num,
	}, nil
}

func (execState *ExecutionState) IsStopped() bool {
	return execState.stepNum == execState.engine.numSteps
}

func (execState *ExecutionState) Hash() common.Hash {
	if execState.IsStopped() {
		return execState.engine.endStateRoot
	}
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64(execState.engine.internalHash().Bytes(), execState.stepNum))
}

func (execState *ExecutionState) NextState() (*ExecutionState, error) {
	if execState.IsStopped() {
		return nil, OutOfBoundsError
	}
	return &ExecutionState{
		engine:  execState.engine,
		stepNum: execState.stepNum + 1,
	}, nil
}

func (execState *ExecutionState) OneStepProof() ([]byte, error) {
	if execState.IsStopped() {
		return nil, OutOfBoundsError
	}
	ret := execState.engine.serialize()
	ret = append(ret, binary.BigEndian.AppendUint64([]byte{}, execState.stepNum)...)
	return ret, nil
}

func VerifyOneStepProof(beforeStateRoot common.Hash, claimedAfterStateRoot common.Hash, proof []byte) bool {
	if len(proof) < 8 {
		return false
	}
	engine, err := deserializeExecutionEngine(proof[:len(proof)-8])
	if err != nil {
		return false
	}
	beforeState := ExecutionState{
		engine:  engine,
		stepNum: binary.BigEndian.Uint64(proof[len(proof)-8:]),
	}
	if beforeState.Hash() != beforeStateRoot {
		return false
	}
	afterState, err := beforeState.NextState()
	if err != nil {
		return false
	}
	return afterState.Hash() == claimedAfterStateRoot
}

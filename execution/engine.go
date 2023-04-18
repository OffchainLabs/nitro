package execution

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrOutOfBounds = errors.New("instruction number out of bounds")
)

// EngineAtBlock defines a struct that can provide the number of opcodes, big steps,
// and execution states after N opcodes at a specific L2 block height.
type EngineAtBlock interface {
	NumOpcodes() uint64
	NumBigSteps() uint64
	StateAfterSmallSteps(n uint64) (IntermediateStateIterator, error)
	StateAfterBigSteps(n uint64) (IntermediateStateIterator, error)
	StartingAssertionStateHash() common.Hash
	EndingAssertionStateHash() common.Hash
	FirstMachineState() IntermediateStateIterator
	Serialize() []byte
}

// IntermediateStateIterator defines a struct which can be used for iterating over intermediate
// states using an execution engine at a specific L2 block height.
type IntermediateStateIterator interface {
	Engine() EngineAtBlock
	NextMachineState() (IntermediateStateIterator, error)
	CurrentStepNum() uint64
	Hash() common.Hash
	IsStopped() bool
}

// MachineConfig for the machines in the execution engine.
type MachineConfig struct {
	MaxInstructionsPerBlock uint64
	BigStepSize             uint64
}

// DefaultMachineConfig for the engine's machines.
func DefaultMachineConfig() *MachineConfig {
	return &MachineConfig{
		// MaxInstructions per block is defined as 2^43 WAVM opcodes in Arbitrum.
		MaxInstructionsPerBlock: 1 << 43,
		// BigStepSize defines a "BigStep" in the challenge protocol
		// as 2^20 WAVM opcodes.
		BigStepSize: 1 << 20,
	}
}

// Engine can provide an execution engine for a specific pre-state of an L2 block,
// giving access to a state iterator to advance opcode-by-opcode and fetch one-step-proofs.
type Engine struct {
	machineCfg                 *MachineConfig
	numSteps                   uint64
	startingAssertionStateHash common.Hash
	endingAssertionStateHash   common.Hash
}

// NewExecutionEngine constructs an engine at a specific block number when given
// a pre and post-state for L2.
func NewExecutionEngine(
	machineCfg *MachineConfig,
	startAssertionStateHash,
	endAssertionStateHash common.Hash,
) (*Engine, error) {
	return &Engine{
		machineCfg:                 machineCfg,
		numSteps:                   machineCfg.MaxInstructionsPerBlock,
		startingAssertionStateHash: startAssertionStateHash,
		endingAssertionStateHash:   endAssertionStateHash,
	}, nil
}

func (engine *Engine) StartingAssertionStateHash() common.Hash {
	return engine.startingAssertionStateHash
}

func (engine *Engine) EndingAssertionStateHash() common.Hash {
	return engine.endingAssertionStateHash
}

// Serialize an execution engine.
func (engine *Engine) Serialize() []byte {
	var ret []byte
	ret = append(ret, engine.startingAssertionStateHash.Bytes()...)
	ret = append(ret, engine.endingAssertionStateHash.Bytes()...)
	ret = append(ret, binary.BigEndian.AppendUint64([]byte{}, engine.numSteps)...)
	return ret
}

// SerializeForHash prepares a machine serialization that helps with determining
// intermediate hashes is comprised of keccak(serializeForHash(machine), stepNum).
// We want validators to agree up to a certain height in subchallenges, and by encoding only
// the start state root and num steps in the machine serialization we achieve that.
func (engine *Engine) SerializeForHash() []byte {
	var ret []byte
	ret = append(ret, engine.startingAssertionStateHash.Bytes()...)
	ret = append(ret, binary.BigEndian.AppendUint64([]byte{}, engine.numSteps)...)
	return ret
}

func deserializeExecutionEngine(buf []byte) (*Engine, error) {
	if len(buf) != 32+32+8 {
		return nil, errors.New("deserialization error")
	}
	return &Engine{
		startingAssertionStateHash: common.BytesToHash(buf[:32]),
		endingAssertionStateHash:   common.BytesToHash(buf[32:64]),
		numSteps:                   binary.BigEndian.Uint64(buf[64:]),
	}, nil
}

func (engine *Engine) internalHash() common.Hash {
	return crypto.Keccak256Hash(engine.SerializeForHash())
}

// NumOpcodes in the engine at the block height.
func (engine *Engine) NumOpcodes() uint64 {
	return engine.numSteps
}

// NumBigSteps in the engine at the block height.
func (engine *Engine) NumBigSteps() uint64 {
	if engine.numSteps <= engine.machineCfg.BigStepSize {
		return 1
	}
	return engine.numSteps / engine.machineCfg.BigStepSize
}

// StateAfterBigSteps gets the intermediate state after executing N big step(s).
// If the number of total steps is less than the total number of opcodes in the N big steps,
// we simply advance by the number of opcodes.
func (engine *Engine) StateAfterBigSteps(num uint64) (IntermediateStateIterator, error) {
	numOpcodes := num * engine.machineCfg.BigStepSize
	if numOpcodes > engine.numSteps {
		numOpcodes = engine.numSteps
	}
	return &ExecutionState{
		engine:  engine,
		stepNum: numOpcodes,
	}, nil
}

// StateAfterSmallSteps gets the intermediate state after executing N WAVM opcode(s).
func (engine *Engine) StateAfterSmallSteps(num uint64) (IntermediateStateIterator, error) {
	if num > engine.numSteps {
		return nil, ErrOutOfBounds
	}
	return &ExecutionState{
		engine:  engine,
		stepNum: num,
	}, nil
}

func (engine *Engine) FirstMachineState() IntermediateStateIterator {
	return &ExecutionState{
		engine:  engine,
		stepNum: 0,
	}
}

// ExecutionState represents execution of opcodes within an L2 block, which is able
// to provide the hash the intermediate machine state as well as retrieve the next state.
type ExecutionState struct {
	engine  *Engine
	stepNum uint64
}

func (execState *ExecutionState) Engine() EngineAtBlock {
	return execState.engine
}

// IsStopped checks if the execution state's machine has reached the last step of computation.
func (execState *ExecutionState) IsStopped() bool {
	return execState.stepNum == execState.engine.numSteps
}

// CurrentStepNum of execution.
func (execState *ExecutionState) CurrentStepNum() uint64 {
	return execState.stepNum
}

// Hash of the execution state is defined as the end state root if the machine
// has finished, or otherwise the intermediary state root defined by hashing the
// internal hash with the step number.
func (execState *ExecutionState) Hash() common.Hash {
	// This is the intermediary state root after executing N steps with the engine.
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64(execState.engine.internalHash().Bytes(), execState.stepNum))
}

// NextState fetches the state at the next step of execution. If the machine is stopped,
// it will return an error.
func (execState *ExecutionState) NextMachineState() (IntermediateStateIterator, error) {
	if execState.IsStopped() {
		return nil, ErrOutOfBounds
	}
	return &ExecutionState{
		engine:  execState.engine,
		stepNum: execState.stepNum + 1,
	}, nil
}

// OneStepProof provides a proof of execution of a single WAVM step for an execution state.
func OneStepProof(execState IntermediateStateIterator) ([]byte, error) {
	if execState.IsStopped() {
		return nil, ErrOutOfBounds
	}
	ret := execState.Engine().Serialize()
	ret = append(ret, binary.BigEndian.AppendUint64([]byte{}, execState.CurrentStepNum())...)
	return ret, nil
}

// VerifyOneStepProof checks the claimed post-state root results from executing
// a specified pre-state hash.
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
	afterState, err := beforeState.NextMachineState()
	if err != nil {
		return false
	}
	return afterState.Hash() == claimedAfterStateRoot
}

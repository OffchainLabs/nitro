package execution

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// MaxInstructions per block is defined as 2^43 WAVM opcodes in Arbitrum.
	MaxInstructionsPerBlock = 1 << 43
	// BigStepSize defines a "BigStep" in the challenge protocol
	// as 2^20 WAVM opcodes.
	BigStepSize = 1 << 20
)

var (
	ErrOutOfBounds = errors.New("instruction number out of bounds")
)

// EngineAtBlock defines a struct that can provide the number of opcodes, big steps,
// and execution states after N opcodes at a specific L2 block height.
type EngineAtBlock interface {
	BlockNum() uint64
	NumOpcodes() uint64
	NumBigSteps() uint64
	StateAfterSmallSteps(n uint64) (IntermediateStateIterator, error)
	StateAfterBigSteps(n uint64) (IntermediateStateIterator, error)
	Serialize() []byte
}

// IntermediateStateIterator defines a struct which can be used for iterating over intermediate
// states using an execution engine at a specific L2 block height.
type IntermediateStateIterator interface {
	Engine() EngineAtBlock
	NextState() (IntermediateStateIterator, error)
	CurrentStepNum() uint64
	Hash() common.Hash
	IsStopped() bool
}

// Config for the engine.
type Config struct {
	FixedNumSteps uint64
}

// DefaultConfig for the engine.
func DefaultConfig() *Config {
	return &Config{
		FixedNumSteps: 0,
	}
}

// BigStepHeight computes the big step an opcode index is in, 1-indexed.
func BigStepHeight(opcodeIndex uint64) uint64 {
	if opcodeIndex < BigStepSize {
		return 1
	}
	return opcodeIndex / BigStepSize
}

// Engine can provide an execution engine for a specific pre-state of an L2 block,
// giving access to a state iterator to advance opcode-by-opcode and fetch one-step-proofs.
type Engine struct {
	startStateRoot common.Hash
	endStateRoot   common.Hash
	numSteps       uint64
	blockNum       uint64
}

// NewExecutionEngine constructs an engine at a specific block number when given
// a pre and post-state for L2.
func NewExecutionEngine(
	blockNum uint64,
	preStateRoot common.Hash,
	postStateRoot common.Hash,
	cfg *Config,
) (*Engine, error) {
	if blockNum == 0 {
		return nil, errors.New("tried to make execution engine for genesis block")
	}
	var numSteps uint64
	if cfg == nil || cfg.FixedNumSteps == 0 {
		numSteps = binary.BigEndian.Uint64(crypto.Keccak256(preStateRoot.Bytes())[:8]) % (1 + MaxInstructionsPerBlock)
	} else {
		numSteps = cfg.FixedNumSteps
	}
	if numSteps == 0 {
		return nil, errors.New("must have at least one step of execution")
	}
	return &Engine{
		startStateRoot: preStateRoot,
		endStateRoot:   postStateRoot,
		numSteps:       numSteps,
		blockNum:       blockNum,
	}, nil
}

// Serialize an execution engine.
func (engine *Engine) Serialize() []byte {
	ret := []byte{}
	ret = append(ret, engine.startStateRoot.Bytes()...)
	ret = append(ret, engine.endStateRoot.Bytes()...)
	ret = append(ret, binary.BigEndian.AppendUint64([]byte{}, engine.numSteps)...)
	return ret
}

func deserializeExecutionEngine(buf []byte) (*Engine, error) {
	if len(buf) != 32+32+8 {
		return nil, errors.New("deserialization error")
	}
	return &Engine{
		startStateRoot: common.BytesToHash(buf[:32]),
		endStateRoot:   common.BytesToHash(buf[32:64]),
		numSteps:       binary.BigEndian.Uint64(buf[64:]),
	}, nil
}

func (engine *Engine) internalHash() common.Hash {
	return crypto.Keccak256Hash(engine.Serialize())
}

// NumOpcodes in the engine at the block height.
func (engine *Engine) NumOpcodes() uint64 {
	return engine.numSteps
}

// NumBigSteps in the engine at the block height.
func (engine *Engine) NumBigSteps() uint64 {
	if engine.numSteps <= BigStepSize {
		return 1
	}
	return engine.numSteps / BigStepSize
}

// BlockNum of the L2 state the engine corresponds to.
func (engine *Engine) BlockNum() uint64 {
	return engine.blockNum
}

// StateAfterBigSteps gets the intermediate state after executing N big step(s).
// If the number of total steps is less than the total number of opcodes in the N big steps,
// we simply advance by the number of opcodes.
func (engine *Engine) StateAfterBigSteps(num uint64) (IntermediateStateIterator, error) {
	numOpcodes := num * BigStepSize
	if numOpcodes > engine.numSteps {
		numOpcodes = engine.numSteps
	}
	return &ExecutionState{
		engine:  engine,
		stepNum: num,
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
	if execState.IsStopped() {
		return execState.engine.endStateRoot
	}
	// This is the intermediary state root after executing N steps with the engine.
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64(execState.engine.internalHash().Bytes(), execState.stepNum))
}

// NextState fetches the state at the next step of execution. If the machine is stopped,
// it will return an error.
func (execState *ExecutionState) NextState() (IntermediateStateIterator, error) {
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
	afterState, err := beforeState.NextState()
	if err != nil {
		return false
	}
	return afterState.Hash() == claimedAfterStateRoot
}

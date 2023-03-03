package execution

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	MaxInstructionsPerBlock = 1 << 43
	BigStepSize             = 1 << 20
)

var (
	OutOfBoundsError = errors.New("instruction number out of bounds")
)

type StateReader interface {
	BlockNum() uint64
	NumOpcodes() uint64
	NumBigSteps() uint64
	StateAfter(n uint64) (*ExecutionState, error)
}

type StateIterator interface {
	NextState() (*ExecutionState, error)
	Hash() common.Hash
	IsStopped() bool
}

type Config struct {
	FixedNumSteps uint64
}

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

type Engine struct {
	startStateRoot common.Hash
	endStateRoot   common.Hash
	numSteps       uint64
	blockNum       uint64
}

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

func (engine *Engine) serialize() []byte {
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
	return crypto.Keccak256Hash(engine.serialize())
}

type ExecutionState struct {
	engine  *Engine
	stepNum uint64
}

func (engine *Engine) NumOpcodes() uint64 {
	return engine.numSteps
}

func (engine *Engine) NumBigSteps() uint64 {
	if engine.numSteps <= BigStepSize {
		return 1
	}
	return engine.numSteps / BigStepSize
}

func (engine *Engine) BlockNum() uint64 {
	return engine.blockNum
}

func (engine *Engine) StateAfter(num uint64) (*ExecutionState, error) {
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
	// This is the intermediary state root after executing N steps with the engine.
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

func OneStepProof(execState *ExecutionState) ([]byte, error) {
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

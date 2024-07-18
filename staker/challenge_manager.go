// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/offchainlabs/nitro/validator"
)

const maxBisectionDegree uint64 = 40

const challengeModeExecution = 2

var initiatedChallengeID common.Hash
var challengeBisectedID common.Hash
var executionChallengeBegunID common.Hash

func init() {
	parsedChallengeManagerABI, err := challengegen.ChallengeManagerMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	initiatedChallengeID = parsedChallengeManagerABI.Events["InitiatedChallenge"].ID
	challengeBisectedID = parsedChallengeManagerABI.Events["Bisected"].ID
	executionChallengeBegunID = parsedChallengeManagerABI.Events["ExecutionChallengeBegun"].ID
}

type ChallengeBackend interface {
	SetRange(ctx context.Context, start uint64, end uint64) error
	GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error)
}

// Assert that ExecutionChallengeBackend implements ChallengeBackend
var _ ChallengeBackend = (*ExecutionChallengeBackend)(nil)

type challengeCore struct {
	con                  *challengegen.ChallengeManager
	challengeManagerAddr common.Address
	challengeIndex       uint64
	client               bind.ContractBackend
	auth                 *bind.TransactOpts
	actingAs             common.Address
	startL1Block         *big.Int
	confirmationBlocks   int64
}

type ChallengeManager struct {
	// fields used in both block and execution challenge
	*challengeCore

	// fields below are used while working on block challenge
	blockChallengeBackend *BlockChallengeBackend

	// fields below are only used to create execution challenge from block challenge
	validator      *StatelessBlockValidator
	maxBatchesRead uint64
	wasmModuleRoot common.Hash

	// these fields are empty until working on execution challenge
	initialMachineMessageCount arbutil.MessageIndex
	executionChallengeBackend  *ExecutionChallengeBackend
	machineFinalStepCount      uint64
}

// NewChallengeManager constructs a new challenge manager.
// Note: latestMachineLoader may be nil if the block validator is disabled
func NewChallengeManager(
	ctx context.Context,
	l1client bind.ContractBackend,
	auth *bind.TransactOpts,
	fromAddr common.Address,
	challengeManagerAddr common.Address,
	challengeIndex uint64,
	val *StatelessBlockValidator,
	startL1Block uint64,
	confirmationBlocks int64,
) (*ChallengeManager, error) {
	con, err := challengegen.NewChallengeManager(challengeManagerAddr, l1client)
	if err != nil {
		return nil, fmt.Errorf("error creating bindgen ChallengeManager: %w", err)
	}

	logs, err := l1client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(startL1Block),
		Addresses: []common.Address{challengeManagerAddr},
		Topics:    [][]common.Hash{{initiatedChallengeID}, {uint64ToIndex(challengeIndex)}},
	})
	if err != nil {
		return nil, fmt.Errorf("error searching logs for InitiatedChallenge event from block %v: %w", startL1Block, err)
	}
	if len(logs) == 0 {
		return nil, fmt.Errorf("didn't find InitiatedChallenge event for challenge %v starting at block %v", challengeIndex, startL1Block)
	}
	if len(logs) > 1 {
		log.Warn("found multiple InitiatedChallenge logs", "challenge", challengeIndex, "count", len(logs), "fromBlock", startL1Block)
	}
	// Multiple logs are in theory fine, as they should all reveal the same preimage.
	// We'll use the most recent log to be safe.
	evmLog := logs[len(logs)-1]
	parsedLog, err := con.ParseInitiatedChallenge(evmLog)
	if err != nil {
		return nil, fmt.Errorf("error parsing InitiatedChallenge event for challenge %v: %w", challengeIndex, err)
	}

	callOpts := &bind.CallOpts{Context: ctx}
	challengeInfo, err := con.Challenges(callOpts, new(big.Int).SetUint64(challengeIndex))
	if err != nil {
		return nil, fmt.Errorf("error getting challenge %v info: %w", challengeIndex, err)
	}

	backend, err := NewBlockChallengeBackend(
		parsedLog,
		challengeInfo.MaxInboxMessages,
		val.streamer,
		val.inboxTracker,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating block challenge backend for challenge %v: %w", challengeIndex, err)
	}
	return &ChallengeManager{
		challengeCore: &challengeCore{
			con:                  con,
			challengeManagerAddr: challengeManagerAddr,
			challengeIndex:       challengeIndex,
			client:               l1client,
			auth:                 auth,
			actingAs:             fromAddr,
			startL1Block:         new(big.Int).SetUint64(startL1Block),
			confirmationBlocks:   confirmationBlocks,
		},
		blockChallengeBackend: backend,
		validator:             val,
		wasmModuleRoot:        challengeInfo.WasmModuleRoot,
		maxBatchesRead:        challengeInfo.MaxInboxMessages,
	}, nil
}

// NewExecutionChallengeManager is for testing only - skips block challenges
func NewExecutionChallengeManager(
	l1client bind.ContractBackend,
	auth *bind.TransactOpts,
	challengeManagerAddr common.Address,
	challengeIndex uint64,
	exec validator.ExecutionRun,
	startL1Block uint64,
	confirmationBlocks int64,
) (*ChallengeManager, error) {
	con, err := challengegen.NewChallengeManager(challengeManagerAddr, l1client)
	if err != nil {
		return nil, err
	}
	backend, err := NewExecutionChallengeBackend(exec)
	if err != nil {
		return nil, err
	}
	return &ChallengeManager{
		challengeCore: &challengeCore{
			con:                  con,
			challengeManagerAddr: challengeManagerAddr,
			challengeIndex:       challengeIndex,
			client:               l1client,
			auth:                 auth,
			actingAs:             auth.From,
			startL1Block:         new(big.Int).SetUint64(startL1Block),
			confirmationBlocks:   confirmationBlocks,
		},
		executionChallengeBackend: backend,
	}, nil
}

type ChallengeSegment struct {
	Hash     common.Hash
	Position uint64
}

type ChallengeState struct {
	Start       *big.Int
	End         *big.Int
	Segments    []ChallengeSegment
	RawSegments [][32]byte
}

// Returns nil if client is a SimulatedBackend
func (m *ChallengeManager) latestConfirmedBlock(ctx context.Context) (*big.Int, error) {
	_, isSimulated := m.client.(*backends.SimulatedBackend)
	if isSimulated {
		return nil, nil
	}
	latestBlock, err := m.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting latest header from client: %w", err)
	}
	if latestBlock.Difficulty.Sign() == 0 {
		latestConfirmed, err := m.client.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
		if err != nil {
			return nil, fmt.Errorf("error getting finalized block from client: %w", err)
		}
		return latestConfirmed.Number, nil
	}
	block := new(big.Int).Sub(latestBlock.Number, big.NewInt(m.confirmationBlocks))
	if block.Sign() < 0 {
		block.SetInt64(0)
	}
	return block, nil
}

func (m *ChallengeManager) ChallengeIndex() uint64 {
	return m.challengeIndex
}

func uint64ToIndex(val uint64) common.Hash {
	var challengeIndex common.Hash
	binary.BigEndian.PutUint64(challengeIndex[(32-8):], val)
	return challengeIndex
}

// Given the challenge's state hash, resolve the full challenge state via the Bisected event.
func (m *ChallengeManager) resolveStateHash(ctx context.Context, stateHash common.Hash) (ChallengeState, error) {
	logs, err := m.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: m.startL1Block,
		Addresses: []common.Address{m.challengeManagerAddr},
		Topics:    [][]common.Hash{{challengeBisectedID}, {uint64ToIndex(m.challengeIndex)}, {stateHash}},
	})
	if err != nil {
		return ChallengeState{}, fmt.Errorf("error searching logs for Bisected event from block %v: %w", m.startL1Block, err)
	}
	if len(logs) == 0 {
		return ChallengeState{}, fmt.Errorf("didn't find Bisected event for challenge %v state hash %v starting at block %v", m.challengeIndex, stateHash, m.startL1Block)
	}
	if len(logs) > 1 {
		log.Warn("found multiple Bisected logs", "challenge", m.challengeIndex, "count", len(logs), "fromBlock", m.startL1Block)
	}
	// Multiple logs are in theory fine, as they should all reveal the same preimage.
	// We'll use the most recent log to be safe.
	evmLog := logs[len(logs)-1]
	parsedLog, err := m.con.ParseBisected(evmLog)
	if err != nil {
		return ChallengeState{}, fmt.Errorf("error parsing Bisected event log for challenge %v state hash %v: %w", m.challengeIndex, stateHash, err)
	}
	state := ChallengeState{
		Start:       parsedLog.ChallengedSegmentStart,
		End:         new(big.Int).Add(parsedLog.ChallengedSegmentStart, parsedLog.ChallengedSegmentLength),
		Segments:    make([]ChallengeSegment, len(parsedLog.ChainHashes)),
		RawSegments: parsedLog.ChainHashes,
	}
	degree := len(parsedLog.ChainHashes) - 1
	currentPosition := new(big.Int).Set(parsedLog.ChallengedSegmentStart)
	normalSegmentLength := new(big.Int).Div(parsedLog.ChallengedSegmentLength, big.NewInt(int64(degree)))
	for i, h := range parsedLog.ChainHashes {
		hash := common.Hash(h)
		if i == len(parsedLog.ChainHashes)-1 {
			if currentPosition.Cmp(state.End) > 0 {
				return ChallengeState{}, errors.New("computed last segment position past end")
			}
			currentPosition.Set(state.End)
		}
		if !currentPosition.IsUint64() {
			return ChallengeState{}, errors.New("challenge segment position doesn't fit in a uint64")
		}
		state.Segments[i] = ChallengeSegment{
			Hash:     hash,
			Position: currentPosition.Uint64(),
		}
		currentPosition.Add(currentPosition, normalSegmentLength)
	}
	return state, nil
}

func (m *ChallengeManager) bisect(ctx context.Context, backend ChallengeBackend, oldState *ChallengeState, startSegment int) (*types.Transaction, error) {
	startSegmentPosition := oldState.Segments[startSegment].Position
	endSegmentPosition := oldState.Segments[startSegment+1].Position
	newChallengeLength := endSegmentPosition - startSegmentPosition
	err := backend.SetRange(ctx, startSegmentPosition, endSegmentPosition)
	if err != nil {
		return nil, fmt.Errorf("error setting challenge %v range of %v to %v on backend: %w", m.challengeIndex, startSegmentPosition, endSegmentPosition, err)
	}
	bisectionDegree := maxBisectionDegree
	if newChallengeLength < bisectionDegree {
		bisectionDegree = newChallengeLength
	}
	newSegments := make([][32]byte, int(bisectionDegree+1))
	position := startSegmentPosition
	normalSegmentLength := newChallengeLength / bisectionDegree
	for i := range newSegments {
		if i == len(newSegments)-1 {
			if position > endSegmentPosition {
				return nil, errors.New("computed last segment position past end when bisecting")
			}
			position = endSegmentPosition
		}
		newSegments[i], err = backend.GetHashAtStep(ctx, position)
		if err != nil {
			return nil, fmt.Errorf("error getting challenge %v hash at step %v: %w", m.challengeIndex, position, err)
		}
		position += normalSegmentLength
	}
	return m.con.BisectExecution(
		m.auth,
		m.challengeIndex,
		challengegen.ChallengeLibSegmentSelection{
			OldSegmentsStart:  oldState.Start,
			OldSegmentsLength: new(big.Int).Sub(oldState.End, oldState.Start),
			OldSegments:       oldState.RawSegments,
			ChallengePosition: big.NewInt(int64(startSegment)),
		},
		newSegments,
	)
}

func (m *ChallengeManager) IsMyTurn(ctx context.Context) (bool, error) {
	callOpts := &bind.CallOpts{Context: ctx}
	responder, err := m.con.CurrentResponder(callOpts, m.challengeIndex)
	if err != nil {
		return false, fmt.Errorf("error getting current responder of challenge %v: %w", m.challengeIndex, err)
	}
	if responder != m.actingAs {
		return false, nil
	}
	// Perform future checks against the latest confirmed block
	callOpts.BlockNumber, err = m.latestConfirmedBlock(ctx)
	if err != nil {
		return false, err
	}
	responder, err = m.con.CurrentResponder(callOpts, m.challengeIndex)
	if err != nil {
		return false, fmt.Errorf("error getting confirmed current responder of challenge %v: %w", m.challengeIndex, err)
	}
	if responder != m.actingAs {
		return false, nil
	}
	return true, nil
}

func (m *ChallengeManager) GetChallengeState(ctx context.Context) (*ChallengeState, error) {
	callOpts := &bind.CallOpts{Context: ctx}
	var err error
	callOpts.BlockNumber, err = m.latestConfirmedBlock(ctx)
	if err != nil {
		return nil, err
	}
	challengeState, err := m.con.ChallengeInfo(callOpts, m.challengeIndex)
	if err != nil {
		return nil, fmt.Errorf("error getting challenge %v info: %w", m.challengeIndex, err)
	}
	if challengeState.ChallengeStateHash == (common.Hash{}) {
		return nil, errors.New("lost challenge (state hash 0)")
	}
	state, err := m.resolveStateHash(ctx, challengeState.ChallengeStateHash)
	if err != nil {
		return nil, fmt.Errorf("error resolving challenge %v state hash %v: %w", m.challengeIndex, challengeState.ChallengeStateHash, err)
	}
	return &state, nil
}

func (m *ChallengeManager) ScanChallengeState(ctx context.Context, backend ChallengeBackend, state *ChallengeState) (int, error) {
	for i, segment := range state.Segments {
		ourHash, err := backend.GetHashAtStep(ctx, segment.Position)
		if err != nil {
			return 0, fmt.Errorf("error getting hash from challenge %v backend at step %v: %w", m.challengeIndex, segment.Position, err)
		}
		log.Debug("checking challenge segment", "challenge", m.challengeIndex, "position", segment.Position, "ourHash", ourHash, "segmentHash", segment.Hash)
		if segment.Hash != ourHash {
			if i == 0 {
				return 0, fmt.Errorf(
					"first segment of challenge %v doesn't match: at step count %v challenge has %v but resolved %v",
					m.challengeIndex, segment.Position, segment.Hash, ourHash,
				)
			}
			return i - 1, nil
		}
	}
	return 0, fmt.Errorf("agreed with entire challenge %v (start step count %v and end step count %v)", m.challengeIndex, state.Start.String(), state.End.String())
}

// Checks if an execution challenge exists on-chain.
// If it exists on-chain but we don't have a backend for it, it creates the execution challenge backend.
// If we have a backend for it but it doesn't exist on-chain, it removes the execution challenge backend.
func (m *ChallengeManager) LoadExecChallengeIfExists(ctx context.Context) error {
	latestConfirmedBlock, err := m.latestConfirmedBlock(ctx)
	if err != nil {
		return err
	}
	callOpts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: latestConfirmedBlock,
	}
	challengeState, err := m.con.ChallengeInfo(callOpts, m.challengeIndex)
	if err != nil {
		return fmt.Errorf("error getting challenge %v info: %w", m.challengeIndex, err)
	}
	if challengeState.Mode != challengeModeExecution {
		m.executionChallengeBackend = nil
		return nil
	}
	if m.executionChallengeBackend != nil {
		return nil
	}
	logs, err := m.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: m.startL1Block,
		Addresses: []common.Address{m.challengeManagerAddr},
		Topics:    [][]common.Hash{{executionChallengeBegunID}, {uint64ToIndex(m.challengeIndex)}},
	})
	if err != nil {
		return fmt.Errorf("error searching challenge %v logs for ExecutionChallengeBegun event from block %v: %w", m.challengeIndex, m.startL1Block, err)
	}
	if len(logs) == 0 {
		return fmt.Errorf("didn't find ExecutionChallengeBegun event for challenge %v starting at block %v", m.challengeIndex, m.startL1Block)
	}
	if len(logs) > 1 {
		return errors.New("expected only one ExecutionChallengeBegun event")
	}
	ev, err := m.con.ParseExecutionChallengeBegun(logs[0])
	if err != nil {
		return fmt.Errorf("error parsing ExecutionChallengeBegun event of challenge %v: %w", m.challengeIndex, err)
	}
	if !ev.BlockSteps.IsUint64() {
		return fmt.Errorf("ExecutionChallengeBegun event has non-uint64 blockSteps of %v", ev.BlockSteps)
	}
	return m.createExecutionBackend(ctx, ev.BlockSteps.Uint64())
}

func (m *ChallengeManager) IssueOneStepProof(
	ctx context.Context,
	oldState *ChallengeState,
	startSegment int,
) (*types.Transaction, error) {
	position := oldState.Segments[startSegment].Position
	proof, err := m.executionChallengeBackend.GetProofAt(ctx, position)
	if err != nil {
		return nil, fmt.Errorf("error getting OSP from challenge %v backend at step %v: %w", m.challengeIndex, position, err)
	}
	return m.challengeCore.con.OneStepProveExecution(
		m.challengeCore.auth,
		m.challengeCore.challengeIndex,
		challengegen.ChallengeLibSegmentSelection{
			OldSegmentsStart:  oldState.Start,
			OldSegmentsLength: new(big.Int).Sub(oldState.End, oldState.Start),
			OldSegments:       oldState.RawSegments,
			ChallengePosition: big.NewInt(int64(startSegment)),
		},
		proof,
	)
}

func (m *ChallengeManager) createExecutionBackend(ctx context.Context, step uint64) error {
	initialCount := m.blockChallengeBackend.GetMessageCountAtStep(step)
	if m.initialMachineMessageCount == initialCount && m.executionChallengeBackend != nil {
		return nil
	}
	m.executionChallengeBackend = nil
	entry, err := m.validator.CreateReadyValidationEntry(ctx, initialCount)
	if err != nil {
		return fmt.Errorf("error creating validation entry for challenge %v msg %v for execution challenge: %w", m.challengeIndex, initialCount, err)
	}
	input, err := entry.ToInput("wavm")
	if err != nil {
		return fmt.Errorf("error getting validation entry input of challenge %v msg %v: %w", m.challengeIndex, initialCount, err)
	}
	var prunedBatches []validator.BatchInfo
	for _, batch := range input.BatchInfo {
		if batch.Number < m.maxBatchesRead {
			prunedBatches = append(prunedBatches, batch)
		}
	}
	input.BatchInfo = prunedBatches
	var execRun validator.ExecutionRun
	for _, spawner := range m.validator.execSpawners {
		if validator.SpawnerSupportsModule(spawner, m.wasmModuleRoot) {
			execRun, err = spawner.CreateExecutionRun(m.wasmModuleRoot, input).Await(ctx)
			if err != nil {
				return fmt.Errorf("error creating execution backend for msg %v: %w", initialCount, err)
			}
			break
		}
	}
	if execRun == nil {
		return fmt.Errorf("did not find valid execution backend")
	}
	backend, err := NewExecutionChallengeBackend(execRun)
	if err != nil {
		return err
	}
	expectedState, expectedStatus, err := m.blockChallengeBackend.GetInfoAtStep(step + 1)
	if err != nil {
		return fmt.Errorf("error getting info from block challenge backend: %w", err)
	}
	machineStepCount, computedState, computedStatus, err := backend.GetFinalState(ctx)
	if err != nil {
		return fmt.Errorf("error getting execution challenge final state: %w", err)
	}
	if expectedStatus != computedStatus {
		return fmt.Errorf("after msg %v expected status %v but got %v", initialCount, expectedStatus, computedStatus)
	}
	if computedStatus == StatusFinished {
		if computedState != expectedState {
			return fmt.Errorf("after msg %v expected global state %v but got %v", initialCount, expectedState, computedState)
		}
	}
	m.executionChallengeBackend = backend
	m.machineFinalStepCount = machineStepCount
	m.initialMachineMessageCount = initialCount
	return nil
}

func (m *ChallengeManager) Act(ctx context.Context) (*types.Transaction, error) {
	err := m.LoadExecChallengeIfExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading execution challenge: %w", err)
	}
	myTurn, err := m.IsMyTurn(ctx)
	if err != nil {
		return nil, fmt.Errorf("error checking if it's our turn: %w", err)
	}
	if !myTurn {
		return nil, nil
	}
	state, err := m.GetChallengeState(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting challenge state: %w", err)
	}

	var backend ChallengeBackend
	if m.executionChallengeBackend != nil {
		backend = m.executionChallengeBackend
	} else {
		backend = m.blockChallengeBackend
	}

	err = backend.SetRange(ctx, state.Start.Uint64(), state.End.Uint64())
	if err != nil {
		return nil, fmt.Errorf("error setting challenge range on backend: %w", err)
	}

	nextMovePos, err := m.ScanChallengeState(ctx, backend, state)
	if err != nil {
		return nil, fmt.Errorf("error scanning challenge state: %w", err)
	}
	startPosition := state.Segments[nextMovePos].Position
	endPosition := state.Segments[nextMovePos+1].Position
	if startPosition+1 != endPosition {
		log.Info("bisecting execution", "challenge", m.challengeIndex, "startPosition", startPosition, "endPosition", endPosition)
		return m.bisect(ctx, backend, state, nextMovePos)
	}
	if m.executionChallengeBackend != nil {
		log.Info("sending onestepproof", "challenge", m.challengeIndex, "startPosition", startPosition, "endPosition", endPosition)
		return m.IssueOneStepProof(
			ctx,
			state,
			nextMovePos,
		)
	}
	err = m.createExecutionBackend(ctx, uint64(nextMovePos))
	if err != nil {
		return nil, fmt.Errorf("error creating execution backend: %w", err)
	}
	machineStepCount := m.machineFinalStepCount
	log.Info("issuing one step proof", "challenge", m.challengeIndex, "machineStepCount", machineStepCount, "initialCount", m.initialMachineMessageCount)
	return m.blockChallengeBackend.IssueExecChallenge(
		m.challengeCore,
		state,
		nextMovePos,
		machineStepCount,
	)
}

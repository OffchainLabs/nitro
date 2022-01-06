//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/pkg/errors"
)

const MAX_BISECTION_DEGREE uint64 = 40
const CONFIRMATION_BLOCKS int64 = 12

var challengeBisectedID common.Hash

func init() {
	parsedChallengeCoreABI, err := abi.JSON(strings.NewReader(challengegen.ChallengeCoreABI))
	if err != nil {
		panic(err)
	}
	challengeBisectedID = parsedChallengeCoreABI.Events["Bisected"].ID
}

// Returns nil if client is a SimulatedBackend
func LatestConfirmedBlock(ctx context.Context, client bind.ContractBackend) (*big.Int, error) {
	_, isSimulated := client.(*backends.SimulatedBackend)
	if isSimulated {
		return nil, nil
	}
	latestBlock, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	block := new(big.Int).Sub(latestBlock.Number, big.NewInt(CONFIRMATION_BLOCKS))
	if block.Sign() < 0 {
		block.SetInt64(0)
	}
	return block, nil
}

type ChallengeBackend interface {
	SetRange(ctx context.Context, start uint64, end uint64) error
	GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error)
}

type ChallengeManager struct {
	// fields used in both block and execution challenge
	con           *challengegen.ChallengeCore
	challengeAddr common.Address
	client        bind.ContractBackend
	auth          *bind.TransactOpts
	actingAs      common.Address
	startL1Block  *big.Int

	// fields below are used while working on block challenge
	blockChallengeBackend *BlockChallengeBackend

	// fields below are only used to create execution challenge from block challenge
	inboxReader       InboxReaderInterface
	inboxTracker      InboxTrackerInterface
	txStreamer        TransactionStreamerInterface
	blockchain        *core.BlockChain
	targetNumMachines int

	initialMachine        *ArbitratorMachine
	initialMachineBlockNr uint64

	// nil untill working on execution challenge
	executionChallengeBackend *ExecutionChallengeBackend
}

func NewChallengeManager(ctx context.Context, l1client bind.ContractBackend, auth *bind.TransactOpts, blockChallengeAddr common.Address, l2blockChain *core.BlockChain, inboxReader InboxReaderInterface, inboxTracker InboxTrackerInterface, txStreamer TransactionStreamerInterface, startL1Block uint64, targetNumMachines int) (*ChallengeManager, error) {
	challengeCoreCon, err := challengegen.NewChallengeCore(blockChallengeAddr, l1client)
	if err != nil {
		return nil, err
	}
	backend, err := NewBlockChallengeBackend(ctx, l2blockChain, inboxTracker, l1client, blockChallengeAddr)
	if err != nil {
		return nil, err
	}
	return &ChallengeManager{
		con:                   challengeCoreCon,
		challengeAddr:         blockChallengeAddr,
		client:                l1client,
		auth:                  auth,
		actingAs:              auth.From,
		startL1Block:          new(big.Int).SetUint64(startL1Block),
		blockChallengeBackend: backend,
		inboxReader:           inboxReader,
		inboxTracker:          inboxTracker,
		txStreamer:            txStreamer,
		blockchain:            l2blockChain,
		targetNumMachines:     targetNumMachines,
	}, nil
}

// for testing only - skips block challenges
func NewExecutionChallengeManager(ctx context.Context, l1client bind.ContractBackend, auth *bind.TransactOpts, execChallengeAddr common.Address, initialMachine MachineInterface, startL1Block uint64, targetNumMachines int) (*ChallengeManager, error) {
	challengeCoreCon, err := challengegen.NewChallengeCore(execChallengeAddr, l1client)
	if err != nil {
		return nil, err
	}
	backend, err := NewExecutionChallengeBackend(initialMachine, targetNumMachines, nil)
	if err != nil {
		return nil, err
	}
	return &ChallengeManager{
		con:                       challengeCoreCon,
		challengeAddr:             execChallengeAddr,
		client:                    l1client,
		auth:                      auth,
		actingAs:                  auth.From,
		startL1Block:              new(big.Int).SetUint64(startL1Block),
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

// Given the challenge's state hash, resolve the full challenge state via the Bisected event.
func (m *ChallengeManager) resolveStateHash(ctx context.Context, stateHash common.Hash) (ChallengeState, error) {
	logs, err := m.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: m.startL1Block,
		Addresses: []common.Address{m.challengeAddr},
		Topics:    [][]common.Hash{{challengeBisectedID}, {stateHash}},
	})
	if err != nil {
		return ChallengeState{}, err
	}
	if len(logs) == 0 {
		return ChallengeState{}, errors.New("didn't find Bisected event")
	}
	// Multiple logs are in theory fine, as they should all reveal the same preimage.
	// We'll use the most recent log to be safe.
	log := logs[len(logs)-1]
	parsedLog, err := m.con.ParseBisected(log)
	if err != nil {
		return ChallengeState{}, err
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
		return nil, err
	}
	bisectionDegree := MAX_BISECTION_DEGREE
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
			return nil, err
		}
		position += normalSegmentLength
	}
	return m.con.BisectExecution(
		m.auth,
		oldState.Start,
		new(big.Int).Sub(oldState.End, oldState.Start),
		oldState.RawSegments,
		big.NewInt(int64(startSegment)),
		newSegments,
	)
}

func (m *ChallengeManager) IsMyTurn(ctx context.Context) (bool, error) {
	callOpts := &bind.CallOpts{Context: ctx}
	responder, err := m.con.CurrentResponder(callOpts)
	if err != nil {
		return false, err
	}
	if responder != m.actingAs {
		return false, nil
	}
	// Perform future checks against the latest confirmed block
	callOpts.BlockNumber, err = LatestConfirmedBlock(ctx, m.client)
	if err != nil {
		return false, err
	}
	responder, err = m.con.CurrentResponder(callOpts)
	if err != nil {
		return false, err
	}
	if responder != m.actingAs {
		return false, nil
	}
	return true, nil
}

func (m *ChallengeManager) GetChallengeState(ctx context.Context) (*ChallengeState, error) {
	callOpts := &bind.CallOpts{Context: ctx}
	var err error
	callOpts.BlockNumber, err = LatestConfirmedBlock(ctx, m.client)
	if err != nil {
		return nil, err
	}
	stateHash, err := m.con.ChallengeStateHash(callOpts)
	if err != nil {
		return nil, err
	}
	if stateHash == (common.Hash{}) {
		return nil, errors.New("lost challenge (state hash 0)")
	}
	state, err := m.resolveStateHash(ctx, stateHash)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *ChallengeManager) ScanChallengeState(ctx context.Context, backend ChallengeBackend, state *ChallengeState) (int, error) {
	for i, segment := range state.Segments {
		ourHash, err := backend.GetHashAtStep(ctx, segment.Position)
		if err != nil {
			return 0, err
		}
		log.Debug("checking challenge segment", "challenge", m.challengeAddr, "position", segment.Position, "ourHash", ourHash, "segmentHash", segment.Hash)
		if segment.Hash != ourHash {
			if i == 0 {
				return 0, errors.Errorf(
					"first challenge segment doesn't match: at step count %v challenge has %v but resolved %v",
					segment.Position, segment.Hash, ourHash,
				)
			}
			return i - 1, nil
		}
	}
	return 0, errors.Errorf("agreed with entire challenge (start step count %v and end step count %v)", state.Start.String(), state.End.String())
}

func (m *ChallengeManager) createInitialMachine(ctx context.Context, blockNum uint64) error {
	if m.initialMachine != nil && m.initialMachineBlockNr == blockNum {
		return nil
	}
	blockHeader := m.blockchain.GetHeaderByNumber(blockNum)
	initialFrozenMachine, err := GetZeroStepMachine(ctx)
	if err != nil {
		return err
	}
	machine := initialFrozenMachine.Clone()
	globalState, err := m.blockChallengeBackend.FindGlobalStateFromHeader(ctx, blockHeader)
	if err != nil {
		return err
	}
	err = machine.SetGlobalState(globalState)
	if err != nil {
		return err
	}
	message, err := m.txStreamer.GetMessage(blockNum)
	if err != nil {
		return err
	}
	nextHeader := m.blockchain.GetHeaderByNumber(blockNum + 1)
	preimages, hasDelayedMsg, delayedMsgNr, err := BlockDataForValidation(m.blockchain, nextHeader, blockHeader, message)
	if err != nil {
		return err
	}
	err = machine.AddPreimages(preimages)
	if err != nil {
		return err
	}
	if hasDelayedMsg {
		delayedBytes, err := m.inboxTracker.GetDelayedMessageBytes(delayedMsgNr)
		if err != nil {
			return err
		}
		err = machine.AddDelayedInboxMessage(delayedMsgNr, delayedBytes)
		if err != nil {
			return err
		}
	}
	batchBytes, err := m.inboxReader.GetSequencerMessageBytes(ctx, globalState.Batch)
	if err != nil {
		return err
	}
	err = machine.AddSequencerInboxMessage(globalState.Batch, batchBytes)
	if err != nil {
		return err
	}
	m.initialMachine = machine
	m.initialMachine.Freeze()
	m.initialMachineBlockNr = blockNum
	return nil
}

func (m *ChallengeManager) TestExecChallenge(ctx context.Context) error {
	if m.executionChallengeBackend != nil {
		return nil
	}

	inExec, addr, blockNum, err := m.blockChallengeBackend.IsInExecutionChallenge(ctx)
	if err != nil || !inExec {
		return err
	}
	con, err := challengegen.NewChallengeCore(addr, m.client)
	if err != nil {
		return err
	}
	err = m.createInitialMachine(ctx, blockNum)
	if err != nil {
		return err
	}
	execBackend, err := NewExecutionChallengeBackend(m.initialMachine, m.targetNumMachines, nil)
	if err != nil {
		return err
	}
	m.con = con
	m.executionChallengeBackend = execBackend
	m.challengeAddr = addr
	return nil
}

func (m *ChallengeManager) Act(ctx context.Context) (*types.Transaction, error) {
	err := m.TestExecChallenge(ctx)
	if err != nil {
		return nil, err
	}
	myTurn, err := m.IsMyTurn(ctx)
	if !myTurn || (err != nil) {
		return nil, err
	}
	state, err := m.GetChallengeState(ctx)
	if err != nil {
		return nil, err
	}

	var backend ChallengeBackend
	if m.executionChallengeBackend != nil {
		backend = m.executionChallengeBackend
	} else {
		backend = m.blockChallengeBackend
	}

	err = backend.SetRange(ctx, state.Start.Uint64(), state.End.Uint64())
	if err != nil {
		return nil, err
	}

	nextMovePos, err := m.ScanChallengeState(ctx, backend, state)
	if err != nil {
		return nil, err
	}
	startPosition := state.Segments[nextMovePos].Position
	endPosition := state.Segments[nextMovePos+1].Position
	if startPosition+1 != endPosition {
		log.Info("bisecting execution", "challenge", m.challengeAddr, "startPosition", startPosition, "endPosition", endPosition)
		return m.bisect(ctx, backend, state, nextMovePos)
	}
	if m.executionChallengeBackend != nil {
		log.Info("sending onestepproof", "challenge", m.challengeAddr, "startPosition", startPosition, "endPosition", endPosition)
		return m.executionChallengeBackend.IssueOneStepProof(ctx, m.client, m.auth, m.challengeAddr, state, nextMovePos)
	}
	blockNum := m.blockChallengeBackend.GetBlockNrAtStep(uint64(nextMovePos))
	err = m.createInitialMachine(ctx, blockNum)
	if err != nil {
		return nil, err
	}
	// TODO: we might also use HostIoMachineTo Speed things up
	stepCountMachine := m.initialMachine.Clone()
	var stepCount uint64
	for stepCountMachine.IsRunning() {
		stepsPerLoop := uint64(1_000_000_000)
		if stepCount > 0 {
			log.Debug("step count machine", "block", blockNum, "steps", stepCount)
		}
		err = stepCountMachine.Step(ctx, stepsPerLoop)
		if err != nil {
			return nil, err
		}
		stepCount += stepsPerLoop
	}
	stepCount = stepCountMachine.GetStepCount()
	log.Info("issuing one step proof", "challenge", m.challengeAddr, "stepCount", stepCount, "blockNum", blockNum)
	return m.blockChallengeBackend.IssueExecChallenge(ctx, m.client, m.auth, m.challengeAddr, state, nextMovePos, stepCount)
}

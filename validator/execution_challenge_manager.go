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
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/pkg/errors"
)

const MAX_BISECTION_DEGREE uint64 = 40

var executionChallengeBisectedID common.Hash

func init() {
	parsedExecutionChallengeABI, err := abi.JSON(strings.NewReader(challengegen.ExecutionChallengeABI))
	if err != nil {
		panic(err)
	}
	executionChallengeBisectedID = parsedExecutionChallengeABI.Events["Bisected"].ID
}

type ExecutionChallengeManager struct {
	con               *challengegen.ExecutionChallenge
	client            bind.ContractBackend
	challengeAddr     common.Address
	startL1Block      *big.Int
	auth              *bind.TransactOpts
	actingAs          common.Address
	initialMachine    MachineInterface
	targetNumMachines int
	cache             *MachineCache
}

func NewExecutionChallengeManager(ctx context.Context, client bind.ContractBackend, auth *bind.TransactOpts, addr common.Address, startL1Block uint64, initialMachine MachineInterface, targetNumMachines int) (*ExecutionChallengeManager, error) {
	if initialMachine.GetStepCount() != 0 {
		return nil, errors.New("initial machine not at 0 steps")
	}
	con, err := challengegen.NewExecutionChallenge(addr, client)
	if err != nil {
		return nil, err
	}
	return &ExecutionChallengeManager{
		con:               con,
		client:            client,
		challengeAddr:     addr,
		startL1Block:      new(big.Int).SetUint64(startL1Block),
		auth:              auth,
		actingAs:          auth.From,
		initialMachine:    initialMachine,
		targetNumMachines: targetNumMachines,
	}, nil
}

type challengeSegment struct {
	hash     common.Hash
	position uint64
}

type challengeState struct {
	start       *big.Int
	end         *big.Int
	segments    []challengeSegment
	rawSegments [][32]byte
}

// Given the challenge's state hash, resolve the full challenge state via the Bisected event.
func (m *ExecutionChallengeManager) resolveStateHash(ctx context.Context, stateHash common.Hash) (challengeState, error) {
	logs, err := m.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: m.startL1Block,
		Addresses: []common.Address{m.challengeAddr},
		Topics:    [][]common.Hash{{executionChallengeBisectedID}, {stateHash}},
	})
	if err != nil {
		return challengeState{}, err
	}
	if len(logs) == 0 {
		return challengeState{}, errors.New("didn't find Bisected event")
	}
	// Multiple logs are in theory fine, as they should all reveal the same preimage.
	// We'll use the most recent log to be safe.
	log := logs[len(logs)-1]
	parsedLog, err := m.con.ParseBisected(log)
	if err != nil {
		return challengeState{}, err
	}
	state := challengeState{
		start:       parsedLog.ChallengedSegmentStart,
		end:         parsedLog.ChallengedSegmentLength,
		segments:    make([]challengeSegment, len(parsedLog.ChainHashes)),
		rawSegments: parsedLog.ChainHashes,
	}
	currentPosition := new(big.Int).Set(parsedLog.ChallengedSegmentStart)
	normalSegmentLength := new(big.Int).Sub(state.end, state.start)
	normalSegmentLength.Div(normalSegmentLength, big.NewInt(int64(len(parsedLog.ChainHashes))))
	for i, h := range parsedLog.ChainHashes {
		hash := common.Hash(h)
		if i == len(parsedLog.ChainHashes)-1 {
			currentPosition.Set(state.end)
		}
		if !currentPosition.IsUint64() {
			return challengeState{}, errors.New("challenge segment position doesn't fit in a uint64")
		}
		state.segments[i] = challengeSegment{
			hash:     hash,
			position: currentPosition.Uint64(),
		}
		currentPosition.Add(currentPosition, normalSegmentLength)
	}
	return state, nil
}

func (m *ExecutionChallengeManager) issueOneStepProof(ctx context.Context, oldState challengeState, position int, proof []byte) error {
	_, err := m.con.OneStepProveExecution(
		m.auth,
		oldState.start,
		new(big.Int).Sub(oldState.end, oldState.start),
		oldState.rawSegments,
		big.NewInt(int64(position)),
		proof,
	)
	return err
}

func (m *ExecutionChallengeManager) bisect(ctx context.Context, oldState challengeState, startSegment int, initialMachine MachineInterface) error {
	startSegmentPosition := oldState.segments[startSegment].position
	endSegmentPosition := oldState.segments[startSegment+1].position
	newChallengeLength := endSegmentPosition - startSegmentPosition
	if !initialMachine.ValidForStep(startSegmentPosition) {
		return errors.New("initial machine not at start segment position")
	}
	m.cache = nil // allow Go to free the old machines
	var err error
	m.cache, err = NewMachineCacheWithEndSteps(ctx, initialMachine, m.targetNumMachines, endSegmentPosition)
	if err != nil {
		return err
	}
	bisectionDegree := MAX_BISECTION_DEGREE
	if newChallengeLength < bisectionDegree {
		bisectionDegree = newChallengeLength
	}
	newSegments := make([][32]byte, int(bisectionDegree+1))
	var mach MachineInterface
	position := startSegmentPosition
	normalSegmentLength := newChallengeLength / uint64(len(newSegments))
	for i := range newSegments {
		if i == len(newSegments)-1 {
			position = endSegmentPosition
		}
		mach, err = m.cache.GetMachineAt(ctx, mach, position)
		if err != nil {
			return err
		}
		newSegments[i] = mach.Hash()
		position += normalSegmentLength
	}
	_, err = m.con.BisectExecution(
		m.auth,
		oldState.start,
		new(big.Int).Sub(oldState.end, oldState.start),
		oldState.rawSegments,
		big.NewInt(int64(startSegment)),
		newSegments,
	)
	return err
}

func (m *ExecutionChallengeManager) Act(ctx context.Context) error {
	callOpts := &bind.CallOpts{Context: ctx}
	responder, err := m.con.CurrentResponder(callOpts)
	if err != nil {
		return err
	}
	if responder != m.actingAs {
		return nil
	}
	stateHash, err := m.con.ChallengeStateHash(callOpts)
	if err != nil {
		return err
	}
	state, err := m.resolveStateHash(ctx, stateHash)
	if err != nil {
		return err
	}

	if m.cache == nil {
		initialMachine := m.initialMachine.CloneMachineInterface()
		if initialMachine.GetStepCount() != 0 {
			return errors.New("initial machine not at 0 steps")
		}
		err = initialMachine.Step(ctx, state.start.Uint64())
		if err != nil {
			return err
		}
		m.cache, err = NewMachineCacheWithEndSteps(ctx, initialMachine, m.targetNumMachines, state.end.Uint64())
		if err != nil {
			return err
		}
	}

	var mach MachineInterface
	for i, segment := range state.segments {
		var prevMach MachineInterface
		if mach != nil {
			prevMach = mach.CloneMachineInterface()
		}
		mach, err = m.cache.GetMachineAt(ctx, mach, segment.position)
		if err != nil {
			return err
		}
		machineHash := mach.Hash()
		if segment.hash != machineHash {
			if i == 0 {
				return errors.Errorf(
					"first challenge segment doesn't match: at step count %v challenge has %v but resolved %v",
					segment.position, segment.hash, machineHash,
				)
			}
			lastSegment := state.segments[i-1]
			if prevMach == nil || !prevMach.ValidForStep(lastSegment.position) {
				return errors.New("internal error: prevMach nil or at wrong step count")
			}
			if lastSegment.position+1 == segment.position {
				err = m.issueOneStepProof(ctx, state, i-1, prevMach.ProveNextStep())
				if err != nil {
					return err
				}
			} else {
				err = m.bisect(ctx, state, i-1, prevMach)
				if err != nil {
					return err
				}
			}
		}
	}

	return errors.Errorf("agreed with entire challenge (start step count %v and end step count %v)", state.start.String(), state.end.String())
}

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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/pkg/errors"
)

const MAX_BISECTION_DEGREE uint64 = 40

var challengeBisectedID common.Hash

func init() {
	parsedChallengeCoreABI, err := abi.JSON(strings.NewReader(challengegen.ChallengeCoreABI))
	if err != nil {
		panic(err)
	}
	challengeBisectedID = parsedChallengeCoreABI.Events["Bisected"].ID
}

type ChallengeBackend interface {
	SetRange(ctx context.Context, start uint64, end uint64) error
	GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error)
	IssueOneStepProof(ctx context.Context, client bind.ContractBackend, auth *bind.TransactOpts, challenge common.Address, oldState challengeState, startSegment int) (*types.Transaction, error)
}

type ChallengeManager struct {
	con           *challengegen.ChallengeCore
	client        bind.ContractBackend
	challengeAddr common.Address
	startL1Block  *big.Int
	auth          *bind.TransactOpts
	actingAs      common.Address
	backend       ChallengeBackend
}

func NewChallengeManager(ctx context.Context, client bind.ContractBackend, auth *bind.TransactOpts, addr common.Address, startL1Block uint64, backend ChallengeBackend) (*ChallengeManager, error) {
	con, err := challengegen.NewChallengeCore(addr, client)
	if err != nil {
		return nil, err
	}
	return &ChallengeManager{
		con:           con,
		client:        client,
		challengeAddr: addr,
		startL1Block:  new(big.Int).SetUint64(startL1Block),
		auth:          auth,
		actingAs:      auth.From,
		backend:       backend,
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
func (m *ChallengeManager) resolveStateHash(ctx context.Context, stateHash common.Hash) (challengeState, error) {
	logs, err := m.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: m.startL1Block,
		Addresses: []common.Address{m.challengeAddr},
		Topics:    [][]common.Hash{{challengeBisectedID}, {stateHash}},
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
		end:         new(big.Int).Add(parsedLog.ChallengedSegmentStart, parsedLog.ChallengedSegmentLength),
		segments:    make([]challengeSegment, len(parsedLog.ChainHashes)),
		rawSegments: parsedLog.ChainHashes,
	}
	degree := len(parsedLog.ChainHashes) - 1
	currentPosition := new(big.Int).Set(parsedLog.ChallengedSegmentStart)
	normalSegmentLength := new(big.Int).Div(parsedLog.ChallengedSegmentLength, big.NewInt(int64(degree)))
	for i, h := range parsedLog.ChainHashes {
		hash := common.Hash(h)
		if i == len(parsedLog.ChainHashes)-1 {
			if currentPosition.Cmp(state.end) > 0 {
				return challengeState{}, errors.New("computed last segment position past end")
			}
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

func (m *ChallengeManager) bisect(ctx context.Context, oldState challengeState, startSegment int) (*types.Transaction, error) {
	startSegmentPosition := oldState.segments[startSegment].position
	endSegmentPosition := oldState.segments[startSegment+1].position
	newChallengeLength := endSegmentPosition - startSegmentPosition
	err := m.backend.SetRange(ctx, startSegmentPosition, endSegmentPosition)
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
		newSegments[i], err = m.backend.GetHashAtStep(ctx, position)
		if err != nil {
			return nil, err
		}
		position += normalSegmentLength
	}
	return m.con.BisectExecution(
		m.auth,
		oldState.start,
		new(big.Int).Sub(oldState.end, oldState.start),
		oldState.rawSegments,
		big.NewInt(int64(startSegment)),
		newSegments,
	)
}

func (m *ChallengeManager) Act(ctx context.Context) (*types.Transaction, error) {
	callOpts := &bind.CallOpts{Context: ctx}
	responder, err := m.con.CurrentResponder(callOpts)
	if err != nil {
		return nil, err
	}
	if responder != m.actingAs {
		return nil, nil
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

	err = m.backend.SetRange(ctx, state.start.Uint64(), state.end.Uint64())
	if err != nil {
		return nil, err
	}

	for i, segment := range state.segments {
		ourHash, err := m.backend.GetHashAtStep(ctx, segment.position)
		if err != nil {
			return nil, err
		}
		log.Debug("checking challenge segment", "challenge", m.challengeAddr, "position", segment.position, "ourHash", ourHash, "segmentHash", segment.hash)
		if segment.hash != ourHash {
			if i == 0 {
				return nil, errors.Errorf(
					"first challenge segment doesn't match: at step count %v challenge has %v but resolved %v",
					segment.position, segment.hash, ourHash,
				)
			}
			lastSegment := state.segments[i-1]
			if lastSegment.position+1 == segment.position {
				log.Debug("issuing one step proof", "challenge", m.challengeAddr, "startPosition", lastSegment.position)
				return m.backend.IssueOneStepProof(ctx, m.client, m.auth, m.challengeAddr, state, i-1)
			} else {
				log.Debug("bisecting execution", "challenge", m.challengeAddr, "startPosition", lastSegment.position, "endPosition", segment.position)
				return m.bisect(ctx, state, i-1)
			}
		}
	}

	return nil, errors.Errorf("agreed with entire challenge (start step count %v and end step count %v)", state.start.String(), state.end.String())
}

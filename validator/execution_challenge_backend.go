// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/challengegen"
	"github.com/pkg/errors"
)

type ExecutionChallengeBackend struct {
	initialMachine    MachineInterface
	lastMachine       MachineInterface
	machineCache      *MachineCache
	machineCacheStart uint64
	machineCacheEnd   uint64
	targetNumMachines int
}

// Assert that ExecutionChallengeBackend implements ChallengeBackend
var _ ChallengeBackend = (*ExecutionChallengeBackend)(nil)

// machineCache may be nil, but if present, it must not have a restricted range
func NewExecutionChallengeBackend(
	initialMachine MachineInterface,
	targetNumMachines int,
	machineCache *MachineCache,
) (*ExecutionChallengeBackend, error) {
	if initialMachine.GetStepCount() != 0 {
		return nil, errors.New("initialMachine not at step count 0")
	}
	return &ExecutionChallengeBackend{
		initialMachine:    initialMachine,
		targetNumMachines: targetNumMachines,
		machineCache:      machineCache,
	}, nil
}

func (b *ExecutionChallengeBackend) getMachineAt(ctx context.Context, stepCount uint64) (MachineInterface, error) {
	if b.machineCache == nil {
		mach := b.initialMachine
		if b.lastMachine != nil && b.lastMachine.GetStepCount() <= stepCount {
			mach = b.lastMachine
		}
		mach = mach.CloneMachineInterface()
		err := mach.Step(ctx, stepCount-mach.GetStepCount())
		if err != nil {
			return nil, err
		}
		b.lastMachine = mach
		return mach, nil
	} else {
		mach, err := b.machineCache.GetMachineAt(ctx, b.lastMachine, stepCount)
		if err != nil {
			return nil, err
		}
		b.lastMachine = mach
		return mach, nil
	}
}

func (b *ExecutionChallengeBackend) SetRange(ctx context.Context, start uint64, end uint64) error {
	if b.machineCache != nil && b.machineCacheStart == start && b.machineCacheEnd == end {
		return nil
	}
	startMach, err := b.getMachineAt(ctx, start)
	if err != nil {
		return err
	}
	b.machineCache = nil
	b.machineCache, err = NewMachineCacheWithEndSteps(ctx, startMach, b.targetNumMachines, end)
	return err
}

func (b *ExecutionChallengeBackend) GetHashAtStep(ctx context.Context, position uint64) (common.Hash, error) {
	mach, err := b.getMachineAt(ctx, position)
	if err != nil {
		return common.Hash{}, err
	}
	return mach.Hash(), nil
}

func (b *ExecutionChallengeBackend) IssueOneStepProof(
	ctx context.Context,
	core *challengeCore,
	oldState *ChallengeState,
	startSegment int,
) (*types.Transaction, error) {
	mach, err := b.getMachineAt(ctx, oldState.Segments[startSegment].Position)
	if err != nil {
		return nil, err
	}
	proof := mach.ProveNextStep()
	return core.con.OneStepProveExecution(
		core.auth,
		core.challengeIndex,
		challengegen.ChallengeLibSegmentSelection{
			OldSegmentsStart:  oldState.Start,
			OldSegmentsLength: new(big.Int).Sub(oldState.End, oldState.Start),
			OldSegments:       oldState.RawSegments,
			ChallengePosition: big.NewInt(int64(startSegment)),
		},
		proof,
	)
}

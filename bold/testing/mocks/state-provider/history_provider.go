// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package stateprovider

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/containers/option"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
)

// Collects a list of machine hashes at a message number based on some configuration parameters.
func (s *L2StateBackend) CollectMachineHashes(
	ctx context.Context, cfg *l2stateprovider.HashCollectorConfig,
) ([]common.Hash, error) {
	// We step through the machine in our desired increments, and gather the
	// machine hashes along the way for the history commitment.
	machine, err := s.machineAtBlock(ctx, uint64(cfg.BlockChallengeHeight))
	if err != nil {
		return nil, err
	}
	// Advance the machine to the start index.
	if machErr := machine.Step(uint64(cfg.MachineStartIndex)); machErr != nil {
		return nil, machErr
	}
	hashes := make([]common.Hash, 0, cfg.NumDesiredHashes)
	hashes = append(hashes, s.getMachineHash(machine, uint64(cfg.BlockChallengeHeight)))
	for i := uint64(1); i < cfg.NumDesiredHashes; i++ {
		if stepErr := machine.Step(uint64(cfg.StepSize)); stepErr != nil {
			return nil, stepErr
		}
		hashes = append(hashes, s.getMachineHash(machine, uint64(cfg.BlockChallengeHeight)))
	}
	return hashes, nil
}

// CollectProof Collects osp of at a message number and OpcodeIndex .
func (s *L2StateBackend) CollectProof(
	ctx context.Context,
	assertionMetadata *l2stateprovider.AssociatedAssertionMetadata,
	blockChallengeHeight l2stateprovider.Height,
	machineIndex l2stateprovider.OpcodeIndex,
) ([]byte, error) {
	machine, err := s.machineAtBlock(ctx, uint64(blockChallengeHeight))
	if err != nil {
		return nil, err
	}
	err = machine.Step(uint64(machineIndex))
	if err != nil {
		return nil, err
	}
	return machine.OneStepProof()
}

// Computes a block history commitment from a start L2 message to an end L2 message index
// and up to a required batch index. The hashes used for this commitment are the machine hashes
// at each message number.
func (s *L2StateBackend) L2MessageStatesUpTo(
	ctx context.Context,
	fromState protocol.GoGlobalState,
	batchLimit l2stateprovider.Batch,
	toHeight option.Option[l2stateprovider.Height],
) ([]common.Hash, error) {
	var to l2stateprovider.Height
	if !toHeight.IsNone() {
		to = toHeight.Unwrap()
	} else {
		blockChallengeLeafHeight := s.challengeLeafHeights[0]
		to = l2stateprovider.Height(blockChallengeLeafHeight)
	}
	return s.statesUpTo(uint64(fromState.PosInBatch), uint64(to), uint64(fromState.Batch), uint64(batchLimit))
}

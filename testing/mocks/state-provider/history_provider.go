package stateprovider

import (
	"context"

	"github.com/OffchainLabs/bold/containers/option"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/common"
)

// Collects a list of machine hashes at a message number based on some configuration parameters.
func (s *L2StateBackend) CollectMachineHashes(
	ctx context.Context, cfg *l2stateprovider.HashCollectorConfig,
) ([]common.Hash, error) {
	// We step through the machine in our desired increments, and gather the
	// machine hashes along the way for the history commitment.
	machine, err := s.machineAtBlock(ctx, uint64(cfg.MessageNumber))
	if err != nil {
		return nil, err
	}
	// Advance the machine to the start index.
	if machErr := machine.Step(uint64(cfg.MachineStartIndex)); machErr != nil {
		return nil, machErr
	}
	hashes := make([]common.Hash, 0, cfg.NumDesiredHashes)
	hashes = append(hashes, s.getMachineHash(machine, uint64(cfg.MessageNumber)))
	for i := uint64(1); i < cfg.NumDesiredHashes; i++ {
		if stepErr := machine.Step(uint64(cfg.StepSize)); stepErr != nil {
			return nil, stepErr
		}
		hashes = append(hashes, s.getMachineHash(machine, uint64(cfg.MessageNumber)))
	}
	return hashes, nil
}

// Computes a block history commitment from a start L2 message to an end L2 message index
// and up to a required batch index. The hashes used for this commitment are the machine hashes
// at each message number.
func (s *L2StateBackend) L2MessageStatesUpTo(
	ctx context.Context,
	from l2stateprovider.Height,
	upTo option.Option[l2stateprovider.Height],
	batch l2stateprovider.Batch,
) ([]common.Hash, error) {
	var to l2stateprovider.Height
	if !upTo.IsNone() {
		to = upTo.Unwrap()
	} else {
		blockChallengeLeafHeight := s.challengeLeafHeights[0]
		to = l2stateprovider.Height(blockChallengeLeafHeight)
	}
	return s.statesUpTo(uint64(from), uint64(to), uint64(batch))
}

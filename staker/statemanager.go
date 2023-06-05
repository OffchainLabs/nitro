package staker

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"

	"github.com/OffchainLabs/challenge-protocol-v2/util"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
)

var AccumulatorNotFoundErr = errors.New("accumulator not found")

type StateManager struct {
	validator            *StatelessBlockValidator
	numOpcodesPerBigStep uint64
}

func NewStateManager(val *StatelessBlockValidator, numOpcodesPerBigStep uint64) (*StateManager, error) {
	return &StateManager{
		validator:            val,
		numOpcodesPerBigStep: numOpcodesPerBigStep,
	}, nil
}

// HistoryCommitmentUpTo Produces a block history commitment up to and including messageCount.
func (s *StateManager) HistoryCommitmentUpTo(ctx context.Context, messageCount uint64) (util.HistoryCommitment, error) {
	batch, err := s.findBatchAfterMessageCount(0)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	var stateRoots []common.Hash
	for i := arbutil.MessageIndex(0); i <= arbutil.MessageIndex(messageCount); i++ {
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(batch)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		if batchMsgCount <= i {
			batch++
		}
		root, err := s.getHashAtMessageCountAndBatch(ctx, i, batch)
		if err != nil {
			return util.HistoryCommitment{}, err
		}
		stateRoots = append(stateRoots, root)
	}
	return util.NewHistoryCommitment(messageCount, stateRoots)
}

// BigStepCommitmentUpTo Produces a big step history commitment from big step 0 to toBigStep within block
// challenge heights blockHeight and blockHeight+1.
func (s *StateManager) BigStepCommitmentUpTo(ctx context.Context, wasmModuleRoot common.Hash, blockHeight uint64, toBigStep uint64) (util.HistoryCommitment, error) {
	entry, err := s.validator.CreateReadyValidationEntry(ctx, arbutil.MessageIndex(blockHeight))
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	input, err := entry.ToInput()
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	execRun, err := s.validator.execSpawner.CreateExecutionRun(wasmModuleRoot, input).Await(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	bigStepCommitment := execRun.GetBigStepCommitmentUpTo(toBigStep, s.numOpcodesPerBigStep)
	result, err := bigStepCommitment.Await(ctx)
	if err != nil {
		return util.HistoryCommitment{}, err
	}
	return result, nil
}

func (s *StateManager) findBatchAfterMessageCount(msgCount arbutil.MessageIndex) (uint64, error) {
	if msgCount == 0 {
		return 0, nil
	}
	low := uint64(0)
	high := uint64(math.MaxUint64)
	for {
		// Binary search invariants:
		//   - messageCount(high) >= msgCount
		//   - messageCount(low-1) < msgCount
		//   - high >= low
		if high < low {
			return 0, fmt.Errorf("when attempting to find batch for message count %v high %v < low %v", msgCount, high, low)
		}
		mid := (low + high) / 2
		batchMsgCount, err := s.validator.inboxTracker.GetBatchMessageCount(mid)
		if err != nil {
			if errors.Is(err, AccumulatorNotFoundErr) {
				high = mid
			} else {
				return 0, fmt.Errorf("failed to get batch metadata while binary searching: %w", err)
			}
		}
		if batchMsgCount < msgCount {
			low = mid + 1
		} else if batchMsgCount == msgCount {
			return mid + 1, nil
		} else if mid == low { // batchMsgCount > msgCount
			return mid, nil
		} else { // batchMsgCount > msgCount
			high = mid
		}
	}
}

func (s *StateManager) getHashAtMessageCountAndBatch(_ context.Context, messageCount arbutil.MessageIndex, batch uint64) (common.Hash, error) {
	gs, err := s.getInfoAtMessageCountAndBatch(messageCount, batch)
	if err != nil {
		return common.Hash{}, err
	}
	return gs.Hash(), nil
}

func (s *StateManager) getInfoAtMessageCountAndBatch(messageCount arbutil.MessageIndex, batch uint64) (validator.GoGlobalState, error) {
	globalState, err := s.findGlobalStateFromMessageCountAndBatch(messageCount, batch)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return globalState, nil
}

func (s *StateManager) findGlobalStateFromMessageCountAndBatch(count arbutil.MessageIndex, batch uint64) (validator.GoGlobalState, error) {
	var prevBatchMsgCount arbutil.MessageIndex
	var err error
	if batch > 0 {
		prevBatchMsgCount, err = s.validator.inboxTracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return validator.GoGlobalState{}, err
		}
		if prevBatchMsgCount > count {
			return validator.GoGlobalState{}, errors.New("bad batch provided")
		}
	}
	res, err := s.validator.streamer.ResultAtCount(count)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return validator.GoGlobalState{
		BlockHash:  res.BlockHash,
		SendRoot:   res.SendRoot,
		Batch:      batch,
		PosInBatch: uint64(count - prevBatchMsgCount),
	}, nil
}

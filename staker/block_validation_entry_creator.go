package staker

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/validator"
)

// MELEnabledValidationEntryCreator is responsible for creating validation entries execution of
// messages whose extraction has been validated by a MEL validator.
type MELEnabledValidationEntryCreator struct {
	melValidator MELValidatorInterface
	txStreamer   TransactionStreamerInterface
	melRunner    MELRunnerInterface
}

// NewMELEnabledValidationEntryCreator creates a new instance of MELEnabledValidationEntryCreator.
func NewMELEnabledValidationEntryCreator(
	melValidator MELValidatorInterface,
	txStreamer TransactionStreamerInterface,
	melRunner MELRunnerInterface,
) *MELEnabledValidationEntryCreator {
	return &MELEnabledValidationEntryCreator{
		melValidator: melValidator,
		txStreamer:   txStreamer,
		melRunner:    melRunner,
	}
}

// CreateBlockValidationEntry creates a validation entry for the message at the
// given position whose extraction has been already validated by the MEL validator.
// It talks to the MEL validator to figure out if such a message's extraction has already been validated
// and prepares a validation entry to validate the execution of such a message into a block.
func (m *MELEnabledValidationEntryCreator) CreateBlockValidationEntry(
	ctx context.Context,
	startGlobalState validator.GoGlobalState,
	position arbutil.MessageIndex,
) (*validationEntry, bool, error) {
	var created bool
	latestValidatedMELState, err := m.melValidator.LatestValidatedMELState(ctx)
	if err != nil {
		return nil, created, err
	}
	if latestValidatedMELState == nil {
		log.Trace("create validation entry: no validated MEL state", "pos", position)
		return nil, created, nil
	}
	validatedMsgCount := latestValidatedMELState.MsgCount
	if uint64(position) >= validatedMsgCount {
		log.Trace("create validation entry: nothing to do", "pos", position, "validatedMsgCount", validatedMsgCount)
		return nil, created, nil
	}
	msg, err := m.txStreamer.GetMessage(arbutil.MessageIndex(position))
	if err != nil {
		return nil, created, err
	}
	melStateForMsg, err := m.melRunner.GetState(ctx, msg.Message.Header.BlockNumber)
	if err != nil {
		return nil, created, err
	}
	if melStateForMsg.MsgCount == 0 {
		return nil, created, fmt.Errorf("MEL state for msg at position %d has 0 msg count", position)
	}
	prevResult, err := m.txStreamer.ResultAtMessageIndex(arbutil.MessageIndex(position) - 1)
	if err != nil {
		return nil, created, err
	}
	executionResult, err := m.txStreamer.ResultAtMessageIndex(arbutil.MessageIndex(position))
	if err != nil {
		return nil, created, err
	}
	// Construct preimages.
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)

	// Add MEL state to the preimages map.
	encodedInitialState, err := rlp.EncodeToBytes(latestValidatedMELState)
	if err != nil {
		return nil, created, err
	}
	preimages[arbutil.Keccak256PreimageType][latestValidatedMELState.Hash()] = encodedInitialState

	// Fetch and add the msg releated preimages.
	msgPreimages, err := m.melValidator.FetchMsgPreimages(ctx, uint64(position), melStateForMsg.ParentChainBlockNumber)
	if err != nil {
		return nil, created, err
	}
	validator.CopyPreimagesInto(preimages, msgPreimages)
	startGs := validator.GoGlobalState{
		BlockHash:    prevResult.BlockHash,
		SendRoot:     prevResult.SendRoot,
		MELStateHash: latestValidatedMELState.Hash(),
		MELMsgHash:   msg.Hash(),
		PosInBatch:   uint64(position),
	}
	endGlobalState := validator.GoGlobalState{
		BlockHash:    executionResult.BlockHash,
		SendRoot:     executionResult.SendRoot,
		MELStateHash: latestValidatedMELState.Hash(),
		MELMsgHash:   msg.Hash(),
		PosInBatch:   uint64(position) + 1,
	}
	chainConfig := m.txStreamer.ChainConfig()
	created = true
	return &validationEntry{
		Stage:       ReadyForRecord,
		Pos:         arbutil.MessageIndex(position),
		Start:       startGs,
		End:         endGlobalState,
		msg:         msg,
		ChainConfig: chainConfig,
		Preimages:   preimages,
	}, created, nil
}

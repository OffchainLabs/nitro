package staker

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbnode/mel"
	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/validator"
)

var ErrNothingToDo = errors.New("nothing to do for block validation entry")

type BlockValidationEntryCreator interface {
	CreateBlockValidationEntry(ctx context.Context, blockNumber uint64) error
}

type ValidatedMELStateFetcher interface {
	LatestValidatedMELState(ctx context.Context) (*mel.State, error)
}

type MELEnabledValidationEntryCreator struct {
	melValidator ValidatedMELStateFetcher
	txStreamer   TransactionStreamerInterface
	melRunner    melrunner.MessageExtractor
	// melExtractor TODO
}

func (m *MELEnabledValidationEntryCreator) CreateBlockValidationEntry(
	ctx context.Context,
	position uint64,
	startGlobalState validator.GoGlobalState,
) (*validationEntry, error) {
	latestValidatedMELState, err := m.melValidator.LatestValidatedMELState(ctx)
	if err != nil {
		return nil, err
	}
	validatedMsgCount := latestValidatedMELState.MsgCount
	if position >= validatedMsgCount {
		log.Trace("create validation entry: nothing to do", "pos", position, "validatedMsgCount", validatedMsgCount)
		return nil, ErrNothingToDo
	}
	msg, err := m.txStreamer.GetMessage(arbutil.MessageIndex(position))
	if err != nil {
		return nil, err
	}
	melStateForMsg, err := m.melRunner.GetState(ctx, msg.Message.Header.BlockNumber)
	if err != nil {
		return nil, err
	}
	executionResult, err := m.txStreamer.ResultAtMessageIndex(arbutil.MessageIndex(position))
	if err != nil {
		return nil, err
	}
	// TODO: Fetch the mel result from the mel executor and preimages from the mel validator.
	// melResult, err := m.messageExtractor.MELStateAtCount(ctx, arbutil.MessageIndex(position+1))
	// if err != nil {
	// 	return nil, err
	// }
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	msgHash := msg.WithMELRelevantFields().Hash()
	encodedMsg, err := msg.WithMELRelevantFields().Message.Serialize()
	if err != nil {
		return nil, err
	}
	preimages[arbutil.Keccak256PreimageType][msgHash] = encodedMsg
	// TODO: Needs the message tree preimages so the unified replay binary can do state.ReadMessage
	// after the block is produced and we need to advance to the next message in execution.
	endGlobalState := validator.GoGlobalState{
		BlockHash:    executionResult.BlockHash,
		SendRoot:     executionResult.SendRoot,
		MELStateHash: melStateForMsg.Hash(),
		MELMsgHash:   msgHash,
		PosInBatch:   melStateForMsg.MsgCount, // TODO: Count or an index?
	}
	chainConfig := m.txStreamer.ChainConfig()
	return &validationEntry{
		Stage:       ReadyForRecord,
		Pos:         arbutil.MessageIndex(position),
		Start:       startGlobalState,
		End:         endGlobalState,
		msg:         msg,
		ChainConfig: chainConfig,
		Preimages:   preimages,
	}, nil
}

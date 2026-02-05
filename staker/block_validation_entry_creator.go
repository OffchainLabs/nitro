package staker

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/validator"
)

type MELRunnerInterface interface {
	GetState(ctx context.Context, blockNumber uint64) (*mel.State, error)
}

type blockValidationEntryCreator interface {
	createBlockValidationEntry(
		ctx context.Context,
		startGlobalState validator.GoGlobalState,
		position arbutil.MessageIndex,
	) (*validationEntry, bool, error)
}

type melEnabledValidationEntryCreator struct {
	melValidator MELValidatorInterface
	txStreamer   TransactionStreamerInterface
	melRunner    MELRunnerInterface
}

func newMELEnabledValidationEntryCreator(
	melValidator MELValidatorInterface,
	txStreamer TransactionStreamerInterface,
	melRunner MELRunnerInterface,
) *melEnabledValidationEntryCreator {
	return &melEnabledValidationEntryCreator{
		melValidator: melValidator,
		txStreamer:   txStreamer,
		melRunner:    melRunner,
	}
}

func (m *melEnabledValidationEntryCreator) createBlockValidationEntry(
	ctx context.Context,
	startGlobalState validator.GoGlobalState,
	position arbutil.MessageIndex,
) (*validationEntry, bool, error) {
	var created bool
	latestValidatedMELState, err := m.melValidator.LatestValidatedMELState(ctx)
	if err != nil {
		return nil, created, err
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
	executionResult, err := m.txStreamer.ResultAtMessageIndex(arbutil.MessageIndex(position))
	if err != nil {
		return nil, created, err
	}
	// Construct preimages
	preimages := make(daprovider.PreimagesMap)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	// Add MEL state to the preimages map
	encodedInitialState, err := rlp.EncodeToBytes(melStateForMsg)
	if err != nil {
		return nil, created, err
	}
	preimages[arbutil.Keccak256PreimageType][melStateForMsg.Hash()] = encodedInitialState
	// Fetch and add the msg releated preimages
	msgPreimages := m.melValidator.FetchMsgPreimages(melStateForMsg.ParentChainBlockNumber)
	validator.CopyPreimagesInto(preimages, msgPreimages)
	endGlobalState := validator.GoGlobalState{
		BlockHash:    executionResult.BlockHash,
		SendRoot:     executionResult.SendRoot,
		MELStateHash: melStateForMsg.Hash(),
		MELMsgHash:   msg.Hash(),
		PosInBatch:   melStateForMsg.MsgCount - 1,
	}
	chainConfig := m.txStreamer.ChainConfig()
	created = true
	return &validationEntry{
		Stage:       ReadyForRecord,
		Pos:         arbutil.MessageIndex(position),
		Start:       startGlobalState,
		End:         endGlobalState,
		msg:         msg,
		ChainConfig: chainConfig,
		Preimages:   preimages,
	}, created, nil
}

type preMELValidationEntryCreator struct {
	streamer       TransactionStreamerInterface
	blockValidator *BlockValidator
}

func newPreMELValidationEntryCreator(
	streamer TransactionStreamerInterface,
	blockValidator *BlockValidator,
) *preMELValidationEntryCreator {
	return &preMELValidationEntryCreator{
		streamer:       streamer,
		blockValidator: blockValidator,
	}
}

func (p *preMELValidationEntryCreator) createBlockValidationEntry(
	ctx context.Context,
	startGlobalState validator.GoGlobalState,
	position arbutil.MessageIndex,
) (*validationEntry, bool, error) {
	created := false
	pos := arbutil.MessageIndex(position)
	streamerMsgCount, err := p.streamer.GetProcessedMessageCount()
	if err != nil {
		return nil, created, err
	}
	if pos >= streamerMsgCount {
		log.Trace("create validation entry: nothing to do", "pos", pos, "streamerMsgCount", streamerMsgCount)
		return nil, created, nil
	}
	msg, err := p.streamer.GetMessage(pos)
	if err != nil {
		return nil, created, err
	}
	endRes, err := p.streamer.ResultAtMessageIndex(pos)
	if err != nil {
		return nil, created, err
	}
	if startGlobalState.PosInBatch == 0 || p.blockValidator.nextCreateBatchReread {
		// new batch
		found, fullBatchInfo, err := p.blockValidator.readFullBatch(ctx, p.blockValidator.nextCreateStartGS.Batch)
		if !found {
			return nil, created, err
		}
		if p.blockValidator.nextCreateBatch != nil {
			p.blockValidator.prevBatchCache[p.blockValidator.nextCreateBatch.Number] = p.blockValidator.nextCreateBatch.PostedData
		}
		p.blockValidator.nextCreateBatch = fullBatchInfo
		// #nosec G115
		validatorMsgCountCurrentBatch.Update(int64(fullBatchInfo.MsgCount))
		batchCacheLimit := p.blockValidator.config().BatchCacheLimit
		if len(p.blockValidator.prevBatchCache) > int(batchCacheLimit) {
			for num := range p.blockValidator.prevBatchCache {
				if num+uint64(batchCacheLimit) < p.blockValidator.nextCreateStartGS.Batch {
					delete(p.blockValidator.prevBatchCache, num)
				}
			}
		}
		p.blockValidator.nextCreateBatchReread = false
	}
	endGS := validator.GoGlobalState{
		BlockHash: endRes.BlockHash,
		SendRoot:  endRes.SendRoot,
	}
	if position+1 < p.blockValidator.nextCreateBatch.MsgCount {
		endGS.Batch = startGlobalState.Batch
		endGS.PosInBatch = startGlobalState.PosInBatch + 1
	} else if position+1 == p.blockValidator.nextCreateBatch.MsgCount {
		endGS.Batch = startGlobalState.Batch + 1
		endGS.PosInBatch = 0
	} else {
		return nil, created, fmt.Errorf("illegal batch msg count %d pos %d batch %d", p.blockValidator.nextCreateBatch.MsgCount, position, endGS.Batch)
	}
	chainConfig := p.streamer.ChainConfig()
	prevBatchNums, err := msg.Message.PastBatchesRequired()
	if err != nil {
		return nil, created, err
	}
	prevBatches := make([]validator.BatchInfo, 0, len(prevBatchNums))
	// prevBatchNums are only used for batch reports, each is only used once
	for _, batchNum := range prevBatchNums {
		data, found := p.blockValidator.prevBatchCache[batchNum]
		if found {
			delete(p.blockValidator.prevBatchCache, batchNum)
		} else {
			data, err = p.blockValidator.readPostedBatch(ctx, batchNum)
			if err != nil {
				return nil, created, err
			}
		}
		prevBatches = append(prevBatches, validator.BatchInfo{
			Number: batchNum,
			Data:   data,
		})
	}
	entry, err := newValidationEntry(
		pos, startGlobalState, endGS, msg, p.blockValidator.nextCreateBatch, prevBatches, p.blockValidator.nextCreatePrevDelayed, chainConfig,
	)
	if err != nil {
		return nil, created, err
	}
	created = true
	return entry, created, nil
}

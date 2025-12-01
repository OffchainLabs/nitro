package staker

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client/redis"
	"github.com/offchainlabs/nitro/validator/inputs"
)

var _ ValidatorInstance = (*BlockValidatorInstance)(nil)
var _ LegacyValidatorInstance = (*BlockValidatorInstance)(nil)

type BlockValidatorInstance struct {
	stateless *StatelessBlockValidator
	// can only be accessed from creation thread or if holding reorg-write
	nextCreateBatch       *FullBatchInfo
	nextCreateBatchReread bool
	prevBatchCache        map[uint64][]byte

	nextCreateStartGS     validator.GoGlobalState
	nextCreatePrevDelayed uint64

	// can only be accessed from validation thread or if holding reorg-write
	lastValidGS     validator.GoGlobalState
	legacyValidInfo *legacyLastBlockValidatedDbInfo

	config BlockValidatorConfigFetcher
}

func NewBlockValidatorInstance(
	stateless *StatelessBlockValidator,
	config BlockValidatorConfigFetcher,
) *BlockValidatorInstance {
	return &BlockValidatorInstance{
		stateless:      stateless,
		config:         config,
		prevBatchCache: make(map[uint64][]byte),
	}
}

func (bv *BlockValidatorInstance) ValidatorInputsWriter() (*inputs.Writer, error) {
	valInputsWriter, err := inputs.NewWriter(
		inputs.WithBaseDir(bv.stateless.stack.InstanceDir()),
		inputs.WithSlug("BlockValidator"))
	if err != nil {
		return nil, err
	}
	return valInputsWriter, nil
}

func (bv *BlockValidatorInstance) CurrentGlobalState() validator.GoGlobalState {
	return bv.nextCreateStartGS
}

func (bv *BlockValidatorInstance) LatestWasmModuleRoot() common.Hash {
	return bv.stateless.GetLatestWasmModuleRoot()
}

func (bv *BlockValidatorInstance) RedisValidator() *redis.ValidationClient {
	return bv.stateless.redisValidator
}

func (bv *BlockValidatorInstance) ExecSpawners() []validator.ExecutionSpawner {
	return bv.stateless.execSpawners
}

func (bv *BlockValidatorInstance) PositionsAtCount(count uint64) (GlobalStatePosition, GlobalStatePosition, error) {
	return bv.stateless.GlobalStatePositionsAtCount(arbutil.MessageIndex(count))
}

func (bv *BlockValidatorInstance) WriteLastValidatedInfo(info GlobalStateValidatedInfo) error {
	bv.lastValidGS = info.GlobalState
	encoded, err := rlp.EncodeToBytes(info)
	if err != nil {
		return err
	}
	err = bv.stateless.db.Put(lastGlobalStateValidatedInfoKey, encoded)
	if err != nil {
		return err
	}
	return nil
}

func (bv *BlockValidatorInstance) ReadLastValidatedInfo() (*GlobalStateValidatedInfo, error) {
	return ReadLastValidatedInfo(bv.stateless.db)
}

func (bv *BlockValidatorInstance) LegacyLastValidatedInfo() (*legacyLastBlockValidatedDbInfo, error) {
	exists, err := bv.stateless.db.Has(legacyLastBlockValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	var validated legacyLastBlockValidatedDbInfo
	if !exists {
		return nil, nil
	}
	gsBytes, err := bv.stateless.db.Get(legacyLastBlockValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	err = rlp.DecodeBytes(gsBytes, &validated)
	if err != nil {
		return nil, err
	}
	return &validated, nil
}

func (bv *BlockValidatorInstance) CountAtValidatedGlobalState(validatedGs validator.GoGlobalState) int64 {
	var count int64
	var batchMsgs arbutil.MessageIndex
	var err error
	if validatedGs.Batch > 0 {
		batchMsgs, err = bv.stateless.inboxTracker.GetBatchMessageCount(validatedGs.Batch - 1)
	}
	if err != nil {
		count = -1
	} else {
		// #nosec G115
		count = int64(batchMsgs) + int64(validatedGs.PosInBatch)
	}
	return count
}

func (bv *BlockValidatorInstance) RecordEntry(ctx context.Context, entry *validationEntry) error {
	return bv.stateless.ValidationEntryRecord(ctx, entry)
}

func (bv *BlockValidatorInstance) SetLegacyValidInfo(info *legacyLastBlockValidatedDbInfo) {
	bv.legacyValidInfo = info
}

func (bv *BlockValidatorInstance) SetLastValidatedGlobalState(gs validator.GoGlobalState) {
	bv.lastValidGS = gs
}

func (bv *BlockValidatorInstance) LastValidatedGlobalState() validator.GoGlobalState {
	return bv.lastValidGS
}

func (bv *BlockValidatorInstance) IsValidatedGlobalStateNew(gs validator.GoGlobalState) bool {
	if bv.legacyValidInfo != nil {
		if bv.legacyValidInfo.AfterPosition.BatchNumber > gs.Batch {
			return false
		}
		if bv.legacyValidInfo.AfterPosition.BatchNumber == gs.Batch && bv.legacyValidInfo.AfterPosition.PosInBatch >= gs.PosInBatch {
			return false
		}
		return true
	}
	if bv.lastValidGS.Batch > gs.Batch {
		return false
	}
	if bv.lastValidGS.Batch == gs.Batch && bv.lastValidGS.PosInBatch >= gs.PosInBatch {
		return false
	}
	return true
}

// Not thread safe against reorgs. Caller must hold a reorgMutex.
func (bv *BlockValidatorInstance) LastValidatedCount() (uint64, bool, error) {
	if bv.legacyValidInfo != nil {
		return 0, false, nil
	}
	if bv.lastValidGS.Batch == 0 {
		return 0, false, errors.New("lastValid not initialized. cannot validate genesis")
	}
	caughtUp, count, err := GlobalStateToMsgCount(
		bv.stateless.inboxTracker, bv.stateless.streamer, bv.lastValidGS,
	)
	if err != nil {
		return 0, false, err
	}
	if !caughtUp {
		batchCount, err := bv.stateless.inboxTracker.GetBatchCount()
		if err != nil {
			log.Error("failed reading batch count", "err", err)
			batchCount = 0
		}
		batchMsgCount, err := bv.stateless.inboxTracker.GetBatchMessageCount(batchCount - 1)
		if err != nil {
			log.Error("failed reading batchMsgCount", "err", err)
			batchMsgCount = 0
		}
		processedMsgCount, err := bv.stateless.streamer.GetProcessedMessageCount()
		if err != nil {
			log.Error("failed reading processedMsgCount", "err", err)
			processedMsgCount = 0
		}
		log.Info("validator catching up to last valid", "lastValid.Batch", bv.lastValidGS.Batch, "lastValid.PosInBatch", bv.lastValidGS.PosInBatch, "batchCount", batchCount, "batchMsgCount", batchMsgCount, "processedMsgCount", processedMsgCount)
		return 0, false, nil
	}
	msg, err := bv.stateless.streamer.GetMessage(count - 1)
	if err != nil {
		return 0, false, err
	}
	bv.nextCreateBatchReread = true
	bv.nextCreateStartGS = bv.lastValidGS
	bv.nextCreatePrevDelayed = msg.DelayedMessagesRead
	return uint64(count), true, nil
}

func (bv *BlockValidatorInstance) CheckLegacyValid() error {
	if bv.legacyValidInfo == nil {
		return nil
	}
	batchCount, err := bv.stateless.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}
	requiredBatchCount := bv.legacyValidInfo.AfterPosition.BatchNumber + 1
	if bv.legacyValidInfo.AfterPosition.PosInBatch == 0 {
		requiredBatchCount -= 1
	}
	if batchCount < requiredBatchCount {
		log.Warn("legacy valid batch ahead of db", "current", batchCount, "required", requiredBatchCount)
		return nil
	}
	var msgCount arbutil.MessageIndex
	if bv.legacyValidInfo.AfterPosition.BatchNumber > 0 {
		msgCount, err = bv.stateless.inboxTracker.GetBatchMessageCount(bv.legacyValidInfo.AfterPosition.BatchNumber - 1)
		if err != nil {
			return err
		}
	}
	msgCount += arbutil.MessageIndex(bv.legacyValidInfo.AfterPosition.PosInBatch)
	processedCount, err := bv.stateless.streamer.GetProcessedMessageCount()
	if err != nil {
		return err
	}
	if processedCount < msgCount {
		log.Warn("legacy valid message count ahead of db", "current", processedCount, "required", msgCount)
		return nil
	}

	result := &execution.MessageResult{}
	if msgCount > 0 {
		result, err = bv.stateless.streamer.ResultAtMessageIndex(msgCount - 1)
		if err != nil {
			return err
		}
	}

	if result.BlockHash != bv.legacyValidInfo.BlockHash {
		log.Error("legacy validated blockHash does not fit chain", "info.BlockHash", bv.legacyValidInfo.BlockHash, "chain", result.BlockHash, "count", msgCount)
		return fmt.Errorf("legacy validated blockHash does not fit chain")
	}
	validGS := validator.GoGlobalState{
		BlockHash:  result.BlockHash,
		SendRoot:   result.SendRoot,
		Batch:      bv.legacyValidInfo.AfterPosition.BatchNumber,
		PosInBatch: bv.legacyValidInfo.AfterPosition.PosInBatch,
	}
	err = bv.WriteLastValidatedInfo(GlobalStateValidatedInfo{validGS, nil})
	if err == nil {
		err = bv.stateless.db.Delete(legacyLastBlockValidatedInfoKey)
		if err != nil {
			err = fmt.Errorf("deleting legacy: %w", err)
		}
	}
	if err != nil {
		log.Error("failed writing initial lastValid on upgrade from legacy", "new-info", bv.lastValidGS, "err", err)
	} else {
		log.Info("updated last-valid from legacy", "lastValid", bv.lastValidGS)
	}
	bv.legacyValidInfo = nil
	return nil
}

func (bv *BlockValidatorInstance) CanCreateValidationEntry(position uint64) (bool, error) {
	streamerMsgCount, err := bv.stateless.streamer.GetProcessedMessageCount()
	if err != nil {
		return false, err
	}
	msgPos := arbutil.MessageIndex(position)
	if msgPos >= streamerMsgCount {
		log.Trace("create validation entry: nothing to do", "pos", msgPos, "streamerMsgCount", streamerMsgCount)
		return false, nil
	}
	batchCount, err := bv.stateless.inboxTracker.GetBatchCount()
	if err != nil {
		return false, err
	}
	if batchCount <= bv.nextCreateStartGS.Batch {
		return false, nil
	}
	return true, nil
}

func (bv *BlockValidatorInstance) CreateValidationEntry(
	ctx context.Context,
	position uint64,
) (*validationEntry, error) {
	msgPos := arbutil.MessageIndex(position)
	msg, err := bv.stateless.streamer.GetMessage(msgPos)
	if err != nil {
		return nil, err
	}
	endRes, err := bv.stateless.streamer.ResultAtMessageIndex(msgPos)
	if err != nil {
		return nil, err
	}
	if bv.nextCreateStartGS.PosInBatch == 0 || bv.nextCreateBatchReread {
		// new batch
		found, fullBatchInfo, err := bv.stateless.readFullBatch(ctx, bv.nextCreateStartGS.Batch)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("CanCreateValidationEntry returned true but CreateValidationEntry failed at finding the batch")
		}
		if bv.nextCreateBatch != nil {
			bv.prevBatchCache[bv.nextCreateBatch.Number] = bv.nextCreateBatch.PostedData
		}
		bv.nextCreateBatch = fullBatchInfo
		// #nosec G115
		validatorMsgCountCurrentBatch.Update(int64(fullBatchInfo.MsgCount))
		batchCacheLimit := bv.config().BatchCacheLimit
		if len(bv.prevBatchCache) > int(batchCacheLimit) {
			for num := range bv.prevBatchCache {
				if num+uint64(batchCacheLimit) < bv.nextCreateStartGS.Batch {
					delete(bv.prevBatchCache, num)
				}
			}
		}
		bv.nextCreateBatchReread = false
	}
	endGS := validator.GoGlobalState{
		BlockHash: endRes.BlockHash,
		SendRoot:  endRes.SendRoot,
	}
	if msgPos+1 < bv.nextCreateBatch.MsgCount {
		endGS.Batch = bv.nextCreateStartGS.Batch
		endGS.PosInBatch = bv.nextCreateStartGS.PosInBatch + 1
	} else if msgPos+1 == bv.nextCreateBatch.MsgCount {
		endGS.Batch = bv.nextCreateStartGS.Batch + 1
		endGS.PosInBatch = 0
	} else {
		return nil, fmt.Errorf("illegal batch msg count %d pos %d batch %d", bv.nextCreateBatch.MsgCount, msgPos, endGS.Batch)
	}
	chainConfig := bv.stateless.streamer.ChainConfig()
	prevBatchNums, err := msg.Message.PastBatchesRequired()
	if err != nil {
		return nil, err
	}
	prevBatches := make([]validator.BatchInfo, 0, len(prevBatchNums))
	// prevBatchNums are only used for batch reports, each is only used once
	for _, batchNum := range prevBatchNums {
		data, found := bv.prevBatchCache[batchNum]
		if found {
			delete(bv.prevBatchCache, batchNum)
		} else {
			data, err = bv.stateless.readPostedBatch(ctx, batchNum)
			if err != nil {
				return nil, err
			}
		}
		prevBatches = append(prevBatches, validator.BatchInfo{
			Number: batchNum,
			Data:   data,
		})
	}
	entry, err := newValidationEntry(
		msgPos, bv.nextCreateStartGS, endGS, msg, bv.nextCreateBatch, prevBatches, bv.nextCreatePrevDelayed, chainConfig,
	)
	if err != nil {
		return nil, err
	}
	bv.nextCreateStartGS = entry.End
	bv.nextCreatePrevDelayed = entry.msg.DelayedMessagesRead
	return entry, nil
}

func (bv *BlockValidatorInstance) ResetContextByCount(count uint64) error {
	msgCount := arbutil.MessageIndex(count)
	_, endPosition, err := bv.stateless.GlobalStatePositionsAtCount(msgCount)
	if err != nil {
		return err
	}
	res, err := bv.stateless.streamer.ResultAtMessageIndex(msgCount - 1)
	if err != nil {
		return err
	}
	msg, err := bv.stateless.streamer.GetMessage(msgCount - 1)
	if err != nil {
		return err
	}
	bv.UpdateNextCreation(
		BuildGlobalState(*res, endPosition),
		msg.DelayedMessagesRead,
		true, /* Reread next batch on creation */
	)
	bv.ResetCaches()
	return nil
}

func (bv *BlockValidatorInstance) ResetContextByGlobalStateAndCount(
	globalState validator.GoGlobalState,
	count uint64,
) {
	msg, err := bv.stateless.streamer.GetMessage(arbutil.MessageIndex(count) - 1)
	if err != nil {
		log.Error("getMessage error", "err", err, "count", count)
		return
	}
	bv.UpdateNextCreation(
		globalState,
		msg.DelayedMessagesRead,
		true, /* Should reread next batch on creation */
	)
}

func (bv *BlockValidatorInstance) UpdateNextCreation(
	nextCreateStartGS validator.GoGlobalState,
	nextCreatePrevDelayed uint64,
	nextCreateBatchReread bool,
) {
	bv.nextCreateStartGS = nextCreateStartGS
	bv.nextCreatePrevDelayed = nextCreatePrevDelayed
	bv.nextCreateBatchReread = nextCreateBatchReread
	if bv.nextCreateBatch != nil {
		bv.prevBatchCache[bv.nextCreateBatch.Number] = bv.nextCreateBatch.PostedData
	}
}

func (bv *BlockValidatorInstance) ResetCaches() {
	bv.nextCreateBatchReread = true
	bv.prevBatchCache = make(map[uint64][]byte)
}

func (bv *BlockValidatorInstance) LatestProcessedMessageCount() (uint64, error) {
	count, err := bv.stateless.streamer.GetProcessedMessageCount()
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}

// called from NewBlockValidator, doesn't need to catch locks
func ReadLastValidatedInfo(db ethdb.Database) (*GlobalStateValidatedInfo, error) {
	exists, err := db.Has(lastGlobalStateValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	var validated GlobalStateValidatedInfo
	if !exists {
		return nil, nil
	}
	gsBytes, err := db.Get(lastGlobalStateValidatedInfoKey)
	if err != nil {
		return nil, err
	}
	err = rlp.DecodeBytes(gsBytes, &validated)
	if err != nil {
		return nil, err
	}
	return &validated, nil
}

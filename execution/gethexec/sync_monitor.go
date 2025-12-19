package gethexec

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type syncDataEntry struct {
	maxMessageCount arbutil.MessageIndex
	timestamp       time.Time
}

// syncHistory maintains a time-based sliding window of sync data
type syncHistory struct {
	mutex   sync.RWMutex
	entries []syncDataEntry
	msgLag  time.Duration
}

func newSyncHistory(msgLag time.Duration) *syncHistory {
	return &syncHistory{
		entries: make([]syncDataEntry, 0),
		msgLag:  msgLag,
	}
}

// add adds a new entry and trims old entries beyond msgLag
func (h *syncHistory) add(maxMessageCount arbutil.MessageIndex, timestamp time.Time) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.entries = append(h.entries, syncDataEntry{
		maxMessageCount: maxMessageCount,
		timestamp:       timestamp,
	})

	// Trim entries older than msgLag
	cutoff := timestamp.Add(-h.msgLag)
	i := 0
	for i < len(h.entries) && h.entries[i].timestamp.Before(cutoff) {
		i++
	}
	if i > 0 {
		h.entries = h.entries[i:]
	}
}

// getSyncTarget returns the sync target based on msgLag timing.
// The sync target is the consensusMaxMessageCount from the oldest
// syncDataEntry that was received more recently that than 1 msgLag ago.
// There may be no entries if the syncHistory has not been updated recently.
// Returns 0 if no appropriate entry is found
func (h *syncHistory) getSyncTarget(now time.Time) arbutil.MessageIndex {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if len(h.entries) == 0 {
		return 0
	}

	// Find oldest entry newer than now-msgLag
	windowStart := now.Add(-h.msgLag)

	for _, entry := range h.entries {
		if !entry.timestamp.Before(windowStart) {
			return entry.maxMessageCount
		}
	}

	return 0
}

type SyncMonitorConfig struct {
	SafeBlockWaitForBlockValidator      bool          `koanf:"safe-block-wait-for-block-validator"`
	FinalizedBlockWaitForBlockValidator bool          `koanf:"finalized-block-wait-for-block-validator"`
	MsgLag                              time.Duration `koanf:"msg-lag"`
}

var DefaultSyncMonitorConfig = SyncMonitorConfig{
	SafeBlockWaitForBlockValidator:      false,
	FinalizedBlockWaitForBlockValidator: false,
	MsgLag:                              time.Second,
}

func SyncMonitorConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".safe-block-wait-for-block-validator", DefaultSyncMonitorConfig.SafeBlockWaitForBlockValidator, "wait for block validator to complete before returning safe block number")
	f.Bool(prefix+".finalized-block-wait-for-block-validator", DefaultSyncMonitorConfig.FinalizedBlockWaitForBlockValidator, "wait for block validator to complete before returning finalized block number")
	f.Duration(prefix+".msg-lag", DefaultSyncMonitorConfig.MsgLag, "allowed message lag while still considered in sync")
}

type SyncMonitor struct {
	config    *SyncMonitorConfig
	consensus execution.ConsensusInfo
	exec      *ExecutionEngine

	consensusSyncData atomic.Pointer[execution.ConsensusSyncData]
	syncHistory       *syncHistory
}

func NewSyncMonitor(config *SyncMonitorConfig, exec *ExecutionEngine) *SyncMonitor {
	return &SyncMonitor{
		config:      config,
		exec:        exec,
		syncHistory: newSyncHistory(config.MsgLag),
	}
}

// SetConsensusSyncData updates the sync data pushed from consensus
func (s *SyncMonitor) SetConsensusSyncData(syncData *execution.ConsensusSyncData) {
	s.consensusSyncData.Store(syncData)

	// Add the max message count to history for sync target calculation
	if syncData != nil && syncData.MaxMessageCount > 0 {
		syncTime := time.Now()
		if syncTime.After(syncData.UpdatedAt) {
			syncTime = syncData.UpdatedAt
		}

		s.syncHistory.add(syncData.MaxMessageCount, syncTime)
	}
}

func (s *SyncMonitor) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	data := s.consensusSyncData.Load()
	if data == nil {
		return map[string]interface{}{"error": "no consensus sync data available"}
	}

	res := make(map[string]interface{})

	// Copy sync progress map if it exists (may be nil when synced)
	if data.SyncProgressMap != nil {
		for k, v := range data.SyncProgressMap {
			res[k] = v
		}
	}

	// Always add the max message count
	res["consensusMaxMessageCount"] = data.MaxMessageCount

	// Add execution-calculated sync target
	now := time.Now()
	executionSyncTarget := s.syncHistory.getSyncTarget(now)
	res["executionSyncTarget"] = executionSyncTarget

	// Add execution-specific data
	header, err := s.exec.getCurrentHeader()
	if err != nil {
		res["currentHeaderError"] = err
	} else {
		blockNum := header.Number.Uint64()
		res["blockNum"] = blockNum
		messageNum, err := s.exec.BlockNumberToMessageIndex(blockNum)
		if err != nil {
			res["messageOfLastBlockError"] = err
		} else {
			res["messageOfLastBlock"] = messageNum
		}
	}

	return res
}

func (s *SyncMonitor) SyncProgressMap(ctx context.Context) map[string]interface{} {
	if s.Synced(ctx) {
		return make(map[string]interface{})
	}
	return s.FullSyncProgressMap(ctx)
}

func (s *SyncMonitor) Synced(ctx context.Context) bool {
	data := s.consensusSyncData.Load()
	if data == nil {
		return false
	}

	// Check that the sync data is fresh (not older than MsgLag)
	now := time.Now()
	if now.Sub(data.UpdatedAt) > s.config.MsgLag {
		return false
	}

	// Consensus must report being synced
	if !data.Synced {
		return false
	}

	// Get execution's current message index
	built, err := s.exec.HeadMessageIndex()
	if err != nil {
		log.Error("Error getting head message index", "err", err)
		return false
	}

	// Calculate the sync target based on historical data
	syncTarget := s.syncHistory.getSyncTarget(now)
	if syncTarget == 0 {
		// No valid sync target available yet
		return false
	}

	// Check if execution has reached the calculated sync target
	return built+1 >= syncTarget
}

func (s *SyncMonitor) SetConsensusInfo(consensus execution.ConsensusInfo) {
	s.consensus = consensus
}

func (s *SyncMonitor) BlockMetadataByNumber(ctx context.Context, blockNum uint64) (common.BlockMetadata, error) {
	genesis := s.exec.GetGenesisBlockNumber()
	if blockNum < genesis { // Arbitrum classic block
		return nil, nil
	}
	msgIdx := arbutil.MessageIndex(blockNum - genesis)
	if s.consensus != nil {
		return s.consensus.BlockMetadataAtMessageIndex(msgIdx).Await(ctx)
	}
	log.Debug("FullConsensusClient is not accessible to execution, BlockMetadataByNumber will return nil")
	return nil, nil
}

func (s *SyncMonitor) getFinalityBlockHeader(
	waitForBlockValidator bool,
	validatedFinalityData *arbutil.FinalityData,
	finalityFinalityData *arbutil.FinalityData,
) (*types.Header, error) {
	if finalityFinalityData == nil {
		return nil, nil
	}

	finalityMsgIdx := finalityFinalityData.MsgIdx
	finalityBlockHash := finalityFinalityData.BlockHash
	if waitForBlockValidator {
		if validatedFinalityData == nil {
			return nil, errors.New("block validator not set")
		}
		if finalityFinalityData.MsgIdx > validatedFinalityData.MsgIdx {
			finalityMsgIdx = validatedFinalityData.MsgIdx
			finalityBlockHash = validatedFinalityData.BlockHash
		}
	}

	finalityBlockNumber := s.exec.MessageIndexToBlockNumber(finalityMsgIdx)
	finalityBlock := s.exec.bc.GetBlockByNumber(finalityBlockNumber)
	if finalityBlock == nil {
		log.Debug("Finality block not found", "blockNumber", finalityBlockNumber)
		return nil, nil
	}
	if finalityBlock.Hash() != finalityBlockHash {
		errorMsg := fmt.Sprintf(
			"finality block hash mismatch, blockNumber=%v, block hash provided by consensus=%v, block hash from execution=%v",
			finalityBlockNumber,
			finalityBlockHash,
			finalityBlock.Hash(),
		)
		return nil, errors.New(errorMsg)
	}
	return finalityBlock.Header(), nil
}

func (s *SyncMonitor) SetFinalityData(
	safeFinalityData *arbutil.FinalityData,
	finalizedFinalityData *arbutil.FinalityData,
	validatedFinalityData *arbutil.FinalityData,
) error {
	finalizedBlockHeader, err := s.getFinalityBlockHeader(
		s.config.FinalizedBlockWaitForBlockValidator,
		validatedFinalityData,
		finalizedFinalityData,
	)
	if err != nil {
		return err
	}
	s.exec.bc.SetFinalized(finalizedBlockHeader)

	safeBlockHeader, err := s.getFinalityBlockHeader(
		s.config.SafeBlockWaitForBlockValidator,
		validatedFinalityData,
		safeFinalityData,
	)
	if err != nil {
		return err
	}
	s.exec.bc.SetSafe(safeBlockHeader)

	return nil
}

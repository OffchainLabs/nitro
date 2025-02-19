package gethexec

import (
	"context"
	"errors"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type SyncMonitorConfig struct {
	SafeBlockWaitForBlockValidator      bool `koanf:"safe-block-wait-for-block-validator"`
	FinalizedBlockWaitForBlockValidator bool `koanf:"finalized-block-wait-for-block-validator"`
}

var DefaultSyncMonitorConfig = SyncMonitorConfig{
	SafeBlockWaitForBlockValidator:      false,
	FinalizedBlockWaitForBlockValidator: false,
}

func SyncMonitorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".safe-block-wait-for-block-validator", DefaultSyncMonitorConfig.SafeBlockWaitForBlockValidator, "wait for block validator to complete before returning safe block number")
	f.Bool(prefix+".finalized-block-wait-for-block-validator", DefaultSyncMonitorConfig.FinalizedBlockWaitForBlockValidator, "wait for block validator to complete before returning finalized block number")
}

type SyncMonitor struct {
	config    *SyncMonitorConfig
	consensus execution.ConsensusInfo
	exec      *ExecutionEngine
}

func NewSyncMonitor(config *SyncMonitorConfig, exec *ExecutionEngine) *SyncMonitor {
	return &SyncMonitor{
		config: config,
		exec:   exec,
	}
}

func (s *SyncMonitor) FullSyncProgressMap() map[string]interface{} {
	res := s.consensus.FullSyncProgressMap()

	res["consensusSyncTarget"] = s.consensus.SyncTargetMessageCount()

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

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	if s.Synced() {
		return make(map[string]interface{})
	}
	return s.FullSyncProgressMap()
}

func (s *SyncMonitor) Synced() bool {
	if s.consensus.Synced() {
		built, err := s.exec.HeadMessageNumber()
		consensusSyncTarget := s.consensus.SyncTargetMessageCount()
		if err == nil && built+1 >= consensusSyncTarget {
			return true
		}
	}
	return false
}

func (s *SyncMonitor) SetConsensusInfo(consensus execution.ConsensusInfo) {
	s.consensus = consensus
}

func (s *SyncMonitor) BlockMetadataByNumber(blockNum uint64) (common.BlockMetadata, error) {
	genesis := s.exec.GetGenesisBlockNumber()
	if blockNum < genesis { // Arbitrum classic block
		return nil, nil
	}
	pos := arbutil.MessageIndex(blockNum - genesis)
	if s.consensus != nil {
		return s.consensus.BlockMetadataAtCount(pos + 1)
	}
	log.Debug("FullConsensusClient is not accessible to execution, BlockMetadataByNumber will return nil")
	return nil, nil
}

func (s *SyncMonitor) getFinalityBlock(
	waitForBlockValidator bool,
	validatedMsgCount *arbutil.MessageIndex,
	finalityMsgCount arbutil.MessageIndex,
) (*types.Block, error) {
	if waitForBlockValidator {
		if validatedMsgCount == nil {
			return nil, errors.New("block validator not set")
		}
		if finalityMsgCount > *validatedMsgCount {
			finalityMsgCount = *validatedMsgCount
		}
	}
	finalityBlockNumber := s.exec.MessageIndexToBlockNumber(finalityMsgCount - 1)
	finalityBlock := s.exec.bc.GetBlockByNumber(finalityBlockNumber)
	if finalityBlock == nil {
		return nil, errors.New("unable to get block by number")
	}
	return finalityBlock, nil
}

func (s *SyncMonitor) SetFinalityData(ctx context.Context, finalityData *arbutil.FinalityData) error {
	finalizedBlock, err := s.getFinalityBlock(
		s.config.FinalizedBlockWaitForBlockValidator,
		finalityData.ValidatedMsgCount,
		finalityData.FinalizedMsgCount,
	)
	if err != nil {
		return err
	}
	s.exec.bc.SetFinalized(finalizedBlock.Header())

	safeBlock, err := s.getFinalityBlock(
		s.config.SafeBlockWaitForBlockValidator,
		finalityData.ValidatedMsgCount,
		finalityData.SafeMsgCount,
	)
	if err != nil {
		return err
	}
	s.exec.bc.SetSafe(safeBlock.Header())

	return nil
}

package gethexec

import (
	"context"
	"errors"
	"sync/atomic"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/headerreader"
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

	finalityData atomic.Pointer[arbutil.FinalityData]
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

func (s *SyncMonitor) SafeBlockNumber(ctx context.Context) (uint64, error) {
	finalityData := s.finalityData.Load()
	if finalityData == nil {
		return 0, errors.New("safe block number not synced")
	}
	if !finalityData.FinalitySupported {
		return 0, headerreader.ErrBlockNumberNotSupported
	}
	msg := finalityData.SafeMsgCount

	if s.config.SafeBlockWaitForBlockValidator {
		if !finalityData.BlockValidatorSet {
			return 0, errors.New("block validator not set")
		}
		if msg > finalityData.ValidatedMsgCount {
			msg = finalityData.ValidatedMsgCount
		}
	}
	block := s.exec.MessageIndexToBlockNumber(msg - 1)
	return block, nil
}

func (s *SyncMonitor) FinalizedBlockNumber(ctx context.Context) (uint64, error) {
	finalityData := s.finalityData.Load()
	if finalityData == nil {
		return 0, errors.New("finalized block number not synced")
	}
	if !finalityData.FinalitySupported {
		return 0, headerreader.ErrBlockNumberNotSupported
	}
	msg := finalityData.FinalizedMsgCount

	if s.config.FinalizedBlockWaitForBlockValidator {
		if !finalityData.BlockValidatorSet {
			return 0, errors.New("block validator not set")
		}
		if msg > finalityData.ValidatedMsgCount {
			msg = finalityData.ValidatedMsgCount
		}
	}
	block := s.exec.MessageIndexToBlockNumber(msg - 1)
	return block, nil
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

func (s *SyncMonitor) StoreFinalityData(ctx context.Context, finalityData *arbutil.FinalityData) error {
	s.finalityData.Store(finalityData)

	finalizedBlockNumber, err := s.FinalizedBlockNumber(ctx)
	if errors.Is(err, headerreader.ErrBlockNumberNotSupported) {
		log.Warn("Finality not supported so not setting finalized block number")
	} else if err != nil {
		return err
	} else {
		err = s.exec.SetFinalized(finalizedBlockNumber)
		if err != nil {
			return err
		}
	}

	return nil
}

// Used for testing
func (s *SyncMonitor) GetFinalityData() *arbutil.FinalityData {
	return s.finalityData.Load()
}

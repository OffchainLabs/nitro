package gethexec

import (
	"context"

	"github.com/offchainlabs/nitro/execution"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
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
	consensusSyncTarget := s.consensus.SyncTargetMessageCount()

	built, err := s.exec.HeadMessageNumber()
	if err != nil {
		res["headMsgNumberError"] = err
	}

	res["builtBlock"] = built
	res["consensusSyncTarget"] = consensusSyncTarget

	return res
}

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	if s.consensus.Synced() {
		built, err := s.exec.HeadMessageNumber()
		consensusSyncTarget := s.consensus.SyncTargetMessageCount()
		if err == nil && built+1 >= consensusSyncTarget {
			return make(map[string]interface{})
		}
	}
	return s.FullSyncProgressMap()
}

func (s *SyncMonitor) SafeBlockNumber(ctx context.Context) (uint64, error) {
	if s.consensus == nil {
		return 0, errors.New("not set up for safeblock")
	}
	msg, err := s.consensus.GetSafeMsgCount(ctx)
	if err != nil {
		return 0, err
	}
	if s.config.SafeBlockWaitForBlockValidator {
		latestValidatedCount, err := s.consensus.ValidatedMessageCount()
		if err != nil {
			return 0, err
		}
		if msg > latestValidatedCount {
			msg = latestValidatedCount
		}
	}
	block := s.exec.MessageIndexToBlockNumber(msg - 1)
	return block, nil
}

func (s *SyncMonitor) FinalizedBlockNumber(ctx context.Context) (uint64, error) {
	if s.consensus == nil {
		return 0, errors.New("not set up for safeblock")
	}
	msg, err := s.consensus.GetFinalizedMsgCount(ctx)
	if err != nil {
		return 0, err
	}
	if s.config.FinalizedBlockWaitForBlockValidator {
		latestValidatedCount, err := s.consensus.ValidatedMessageCount()
		if err != nil {
			return 0, err
		}
		if msg > latestValidatedCount {
			msg = latestValidatedCount
		}
	}
	block := s.exec.MessageIndexToBlockNumber(msg - 1)
	return block, nil
}

func (s *SyncMonitor) Synced() bool {
	return len(s.SyncProgressMap()) == 0
}

func (s *SyncMonitor) SetConsensusInfo(consensus execution.ConsensusInfo) {
	s.consensus = consensus
}

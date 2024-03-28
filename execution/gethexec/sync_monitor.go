package gethexec

import (
	"context"
	"time"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type SyncMonitorConfig struct {
	SyncMapTimout                       time.Duration `koanf:"syncmap-timeout"`
	SafeBlockWaitForBlockValidator      bool          `koanf:"safe-block-wait-for-block-validator"`
	FinalizedBlockWaitForBlockValidator bool          `koanf:"finalized-block-wait-for-block-validator"`
}

var DefaultSyncMonitorConfig = SyncMonitorConfig{
	SyncMapTimout:                       time.Second * 5,
	SafeBlockWaitForBlockValidator:      false,
	FinalizedBlockWaitForBlockValidator: false,
}

func SyncMonitorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".syncmap-timeout", DefaultSyncMonitorConfig.SyncMapTimout, "timeout for requests to get sync map")
	f.Bool(prefix+".safe-block-wait-for-block-validator", DefaultSyncMonitorConfig.SafeBlockWaitForBlockValidator, "wait for block validator to complete before returning safe block number")
	f.Bool(prefix+".finalized-block-wait-for-block-validator", DefaultSyncMonitorConfig.FinalizedBlockWaitForBlockValidator, "wait for block validator to complete before returning finalized block number")
}

type SyncMonitor struct {
	stopwaiter.StopWaiter
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

func (s *SyncMonitor) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
}

func (s *SyncMonitor) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	res, err := s.consensus.FullSyncProgressMap().Await(ctx)
	if err != nil {
		res = make(map[string]interface{})
		res["consensusSyncErr"] = err
		return res
	}
	consensusSyncTarget, err := s.consensus.SyncTargetMessageCount().Await(ctx)
	if err != nil {
		res["syncTargetError"] = err
		return res
	}

	built, err := s.exec.HeadMessageNumber().Await(ctx)
	if err != nil {
		res["headMsgNumberError"] = err
	}

	res["builtBlock"] = built
	res["consensusSyncTarget"] = consensusSyncTarget

	return res
}

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	ctx, cancel := context.WithTimeout(s.GetContext(), s.config.SyncMapTimout)
	defer cancel()
	consensusSynced, err := s.consensus.Synced().Await(ctx)
	if err == nil && consensusSynced {
		built, err := s.exec.HeadMessageNumber().Await(ctx)
		var consensusSyncTarget arbutil.MessageIndex
		if err != nil {
			consensusSyncTarget, err = s.consensus.SyncTargetMessageCount().Await(ctx)
		}
		if err != nil && built+1 >= consensusSyncTarget {
			return make(map[string]interface{})
		}
	}
	return s.FullSyncProgressMap(ctx)
}

func (s *SyncMonitor) SafeBlockNumber(ctx context.Context) (uint64, error) {
	if s.consensus == nil {
		return 0, errors.New("not set up for safeblock")
	}
	msg, err := s.consensus.GetSafeMsgCount().Await(ctx)
	if err != nil {
		return 0, err
	}
	if s.config.SafeBlockWaitForBlockValidator {
		latestValidatedCount, err := s.consensus.ValidatedMessageCount().Await(ctx)
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
	msg, err := s.consensus.GetFinalizedMsgCount().Await(ctx)
	if err != nil {
		return 0, err
	}
	if s.config.FinalizedBlockWaitForBlockValidator {
		latestValidatedCount, err := s.consensus.ValidatedMessageCount().Await(ctx)
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

func (s *SyncMonitor) SetConsensusInfo(consensus execution.ConsensusInfo) {
	s.consensus = consensus
}

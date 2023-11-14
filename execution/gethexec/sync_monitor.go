package gethexec

import (
	"context"
	"time"

	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type SyncMonitorConfig struct {
	ConsensusTimeout time.Duration `koanf:"consensus-timeout" reload:"hot"`
}

var DefaultSyncMonitorConfig = SyncMonitorConfig{
	ConsensusTimeout: time.Second * 5,
}

type SyncMonitorConfigFetcher func() *SyncMonitorConfig

type SyncMonitor struct {
	stopwaiter.StopWaiter
	consensus consensus.ConsensusInfo
	exec      *ExecutionEngine
	config    SyncMonitorConfigFetcher
}

func NewSyncMonitor(exec *ExecutionEngine, config SyncMonitorConfigFetcher, consensus consensus.ConsensusInfo) *SyncMonitor {
	return &SyncMonitor{
		exec:      exec,
		config:    config,
		consensus: consensus,
	}
}

func SyncMonitorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".consensus-timeout", DefaultSyncMonitorConfig.ConsensusTimeout, "timeout for requests to consensus client")
}

func (s *SyncMonitor) Start(ctx_in context.Context) {
	s.StopWaiter.Start(ctx_in, s)
}

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	ctx, cancel := context.WithTimeout(s.GetContext(), s.config().ConsensusTimeout)
	defer cancel()
	res, err := s.consensus.SyncProgressMap().Await(ctx)
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

	built, err := s.exec.HeadMessageNumber().Await(s.GetContext())
	if err != nil {
		res["headMsgNumberError"] = err
	}

	if built+1 >= consensusSyncTarget && len(res) == 0 {
		return res
	}

	res["builtBlock"] = built
	res["consensusSyncTarget"] = consensusSyncTarget

	return res
}

func (s *SyncMonitor) SafeBlockNumber(ctx context.Context) (uint64, error) {
	if s.consensus == nil {
		return 0, errors.New("not set up for safeblock")
	}
	msg, err := s.consensus.GetSafeMsgCount().Await(ctx)
	if err != nil {
		return 0, err
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
	block := s.exec.MessageIndexToBlockNumber(msg - 1)
	return block, nil
}

func (s *SyncMonitor) Synced() bool {
	return len(s.SyncProgressMap()) == 0
}

func (s *SyncMonitor) SetConsensusInfo(consensus consensus.ConsensusInfo) error {
	if s.consensus != nil {
		return errors.New("trying to set consensus in sync-monitor while already set")
	}
	s.consensus = consensus
	return nil
}

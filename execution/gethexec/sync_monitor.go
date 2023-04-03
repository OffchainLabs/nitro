package gethexec

import (
	"context"

	"github.com/offchainlabs/nitro/execution"
	"github.com/pkg/errors"
)

type SyncMonitor struct {
	consensus execution.ConsensusInfo
	exec      *ExecutionEngine
}

func NewSyncMonitor(exec *ExecutionEngine) *SyncMonitor {
	return &SyncMonitor{
		exec: exec,
	}
}

func (s *SyncMonitor) SyncProgressMap() map[string]interface{} {
	res := s.consensus.SyncProgressMap()
	consensusSyncTarget := s.consensus.SyncTargetMessageCount()

	built, err := s.exec.HeadMessageNumber()
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
	msg, err := s.consensus.GetSafeMsgCount(ctx)
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
	msg, err := s.consensus.GetFinalizedMsgCount(ctx)
	if err != nil {
		return 0, err
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

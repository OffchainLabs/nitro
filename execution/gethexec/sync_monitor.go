package gethexec

import (
	"context"
	"errors"
	"fmt"

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
		built, err := s.exec.HeadMessageIndex()
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
	msgIdx := arbutil.MessageIndex(blockNum - genesis)
	if s.consensus != nil {
		return s.consensus.BlockMetadataAtMessageIndex(msgIdx)
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
		return nil, errors.New("unable to get block by number, number=" + fmt.Sprint(finalityBlockNumber))
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
	ctx context.Context,
	safeFinalityData *arbutil.FinalityData,
	finalizedFinalityData *arbutil.FinalityData,
	validatedFinalityData *arbutil.FinalityData,
) error {
	s.exec.createBlocksMutex.Lock()
	defer s.exec.createBlocksMutex.Unlock()

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

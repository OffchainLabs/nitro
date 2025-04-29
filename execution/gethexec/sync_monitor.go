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

func (s *SyncMonitor) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	res, err := s.consensus.FullSyncProgressMap().Await(ctx)
	if err != nil {
		res = make(map[string]interface{})
		res["fullSyncProgressMapError"] = err
	}

	consensusSyncTarget, err := s.consensus.SyncTargetMessageCount().Await(ctx)
	if err != nil {
		res["consensusSyncTargetError"] = err
	} else {
		res["consensusSyncTarget"] = consensusSyncTarget
	}

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
	synced, err := s.consensus.Synced().Await(ctx)
	if err != nil {
		log.Error("Error checking if consensus is synced", "err", err)
		return false
	}
	if synced {
		built, err := s.exec.HeadMessageIndex()
		if err != nil {
			log.Error("Error getting head message index", "err", err)
			return false
		}

		consensusSyncTarget, err := s.consensus.SyncTargetMessageCount().Await(ctx)
		if err != nil {
			log.Error("Error getting consensus sync target", "err", err)
			return false
		}
		if consensusSyncTarget == 0 {
			return false
		}

		if built+1 >= consensusSyncTarget {
			return true
		}
	}
	return false
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

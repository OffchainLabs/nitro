// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

// EagerBlockRecorderConfig configures the eager preimage recording system.
type EagerBlockRecorderConfig struct {
	Enable          bool `koanf:"enable"`
	RetentionBlocks int  `koanf:"retention-blocks"` // how many blocks' preimages to retain
	CacheSize       int  `koanf:"cache-size"`       // in-memory cache entries
}

var DefaultEagerBlockRecorderConfig = EagerBlockRecorderConfig{
	Enable:          false,
	RetentionBlocks: 100000, // ~1 day at ~1s blocks
	CacheSize:       1000,
}

func EagerBlockRecorderConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultEagerBlockRecorderConfig.Enable, "enable eager preimage recording for PathDB block validation")
	f.Int(prefix+".retention-blocks", DefaultEagerBlockRecorderConfig.RetentionBlocks, "number of blocks to retain eager preimage records for")
	f.Int(prefix+".cache-size", DefaultEagerBlockRecorderConfig.CacheSize, "number of eager preimage records to cache in memory")
}

// EagerBlockRecorder captures preimages during normal block production and
// stores them persistently. When the validator requests preimages via
// RecordBlockCreation, it serves the pre-stored data instead of replaying
// the block. This enables block validation with PathDB.
type EagerBlockRecorder struct {
	config     *EagerBlockRecorderConfig
	store      *arbitrum.EagerPreimageStore
	execEngine *ExecutionEngine
	bc         *core.BlockChain
}

// NewEagerBlockRecorder creates a new EagerBlockRecorder.
func NewEagerBlockRecorder(config *EagerBlockRecorderConfig, execEngine *ExecutionEngine, ethDb ethdb.Database) *EagerBlockRecorder {
	return &EagerBlockRecorder{
		config:     config,
		store:      arbitrum.NewEagerPreimageStore(ethDb, config.CacheSize),
		execEngine: execEngine,
		bc:         execEngine.bc,
	}
}

// StoreBlockPreimages persists the captured preimages and user WASMs after
// a block is successfully produced.
func (r *EagerBlockRecorder) StoreBlockPreimages(
	blockHash common.Hash,
	blockNumber uint64,
	preimages map[common.Hash][]byte,
	userWasms state.UserWasms,
	minBlockAccessed uint64,
) error {
	record := &arbitrum.EagerBlockRecord{
		Preimages:        preimages,
		UserWasms:        userWasms,
		MinBlockAccessed: minBlockAccessed,
	}
	return r.store.Store(blockHash, blockNumber, record)
}

// RecordBlockCreation implements the validator's request for preimages by
// serving pre-stored data instead of replaying the block. It also adds
// RLP-encoded block headers to the preimage map (matching the behavior
// of the existing RecordingDatabase.PreimagesFromRecording).
func (r *EagerBlockRecorder) RecordBlockCreation(
	ctx context.Context,
	pos arbutil.MessageIndex,
	msg *arbostypes.MessageWithMetadata,
	wasmTargets []rawdb.WasmTarget,
) (*execution.RecordResult, error) {
	_ = ctx
	_ = msg
	_ = wasmTargets
	blockNum := r.execEngine.MessageIndexToBlockNumber(pos)
	blockHash := r.bc.GetCanonicalHash(uint64(blockNum))
	if blockHash == (common.Hash{}) {
		return nil, fmt.Errorf("canonical hash not found for block %d", blockNum)
	}

	record, err := r.store.Get(blockHash)
	if err != nil {
		return nil, fmt.Errorf("eager preimages not found for block %d (hash %v): %w", blockNum, blockHash, err)
	}

	log.Debug("EagerBlockRecorder.RecordBlockCreation",
		"pos", pos, "blockNum", blockNum, "blockHash", blockHash,
		"numPreimages", len(record.Preimages), "minBlockAccessed", record.MinBlockAccessed,
		"numUserWasms", len(record.UserWasms))

	// Make a copy of preimages so we can add headers without mutating the stored record
	preimages := make(map[common.Hash][]byte, len(record.Preimages))
	for k, v := range record.Preimages {
		preimages[k] = v
	}

	// Add RLP-encoded block headers as preimages (same logic as
	// RecordingDatabase.PreimagesFromRecording in recordingdb.go)
	prevBlockNum := uint64(blockNum) - 1
	if pos > 0 && record.MinBlockAccessed <= prevBlockNum {
		for i := record.MinBlockAccessed; i <= prevBlockNum; i++ {
			header := r.bc.GetHeaderByNumber(i)
			if header == nil {
				log.Warn("EagerBlockRecorder: header not found", "blockNum", i)
				continue
			}
			hash := header.Hash()
			headerBytes, err := rlp.EncodeToBytes(header)
			if err != nil {
				return nil, fmt.Errorf("error RLP encoding header %d: %w", i, err)
			}
			preimages[hash] = headerBytes
			log.Debug("EagerBlockRecorder: added header preimage", "blockNum", i, "hash", hash)
		}
	}

	return &execution.RecordResult{
		Pos:       pos,
		BlockHash: blockHash,
		Preimages: preimages,
		UserWasms: record.UserWasms,
	}, nil
}

// PrepareForRecord is a no-op for eager recording since preimages are already
// stored during block production.
func (r *EagerBlockRecorder) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	return nil
}

// ReorgTo cleans up eagerly recorded preimages for blocks beyond the reorg target.
func (r *EagerBlockRecorder) ReorgTo(hdr *types.Header) {
	if hdr == nil {
		return
	}
	reorgTarget := hdr.Number.Uint64()
	// Delete preimages for blocks after the reorg target
	for blockNum := reorgTarget + 1; ; blockNum++ {
		hash := r.bc.GetCanonicalHash(blockNum)
		if hash == (common.Hash{}) {
			break
		}
		if err := r.store.Delete(hash); err != nil {
			log.Warn("failed to delete eager preimage on reorg", "block", blockNum, "err", err)
		}
	}
}

// GarbageCollect removes preimage records for blocks older than the retention window.
func (r *EagerBlockRecorder) GarbageCollect(validatedBlockNum uint64) {
	if validatedBlockNum <= uint64(r.config.RetentionBlocks) {
		return
	}
	cutoff := validatedBlockNum - uint64(r.config.RetentionBlocks)
	if err := r.store.GarbageCollect(cutoff, func(num uint64) common.Hash {
		return r.bc.GetCanonicalHash(num)
	}); err != nil {
		log.Warn("eager preimage garbage collection failed", "err", err)
	}
}

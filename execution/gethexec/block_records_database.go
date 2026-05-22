// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

var blockRecordKeyPrefix = []byte("ctbr:")

type blockRecordsDatabase interface {
	writeRecording(recording *chainTipRecording) error
	readRecording(pos arbutil.MessageIndex) (*chainTipRecording, bool, error)
}

type keyValueBlockRecordsDatabase struct {
	db ethdb.KeyValueStore
}

func newBlockRecordsDatabase(db ethdb.KeyValueStore) blockRecordsDatabase {
	if db == nil {
		return nil
	}
	return &keyValueBlockRecordsDatabase{db: db}
}

func blockRecordKey(pos arbutil.MessageIndex) []byte {
	key := make([]byte, len(blockRecordKeyPrefix)+8)
	copy(key, blockRecordKeyPrefix)
	binary.BigEndian.PutUint64(key[len(blockRecordKeyPrefix):], uint64(pos))
	return key
}

type persistedChainTipRecording struct {
	BlockHash         common.Hash
	Preimages         []persistedPreimage
	FirstHeaderNumber uint64
	CodeHashes        []common.Hash
	WasmKeys          []persistedWasmKey
}

type persistedPreimage struct {
	Hash     common.Hash
	Preimage []byte
}

type persistedWasmKey struct {
	ModuleHash common.Hash
	Target     rawdb.WasmTarget
}

func persistableChainTipRecording(recording *chainTipRecording) (*persistedChainTipRecording, error) {
	if recording == nil || recording.record == nil {
		return nil, fmt.Errorf("cannot persist nil chain-tip recording")
	}
	persisted := &persistedChainTipRecording{
		BlockHash:         recording.record.BlockHash,
		Preimages:         make([]persistedPreimage, 0, len(recording.record.Preimages)),
		FirstHeaderNumber: recording.firstHeaderNumber,
		CodeHashes:        recording.codeHashes,
		WasmKeys:          make([]persistedWasmKey, 0, len(recording.wasmKeys)),
	}
	for hash, preimage := range recording.record.Preimages {
		persisted.Preimages = append(persisted.Preimages, persistedPreimage{
			Hash:     hash,
			Preimage: preimage,
		})
	}
	for _, key := range recording.wasmKeys {
		persisted.WasmKeys = append(persisted.WasmKeys, persistedWasmKey{
			ModuleHash: key.moduleHash,
			Target:     key.target,
		})
	}
	return persisted, nil
}

func chainTipRecordingFromPersisted(pos arbutil.MessageIndex, persisted *persistedChainTipRecording) *chainTipRecording {
	preimages := make(map[common.Hash][]byte, len(persisted.Preimages))
	for _, preimage := range persisted.Preimages {
		preimages[preimage.Hash] = common.CopyBytes(preimage.Preimage)
	}
	wasmKeys := make([]wasmKey, 0, len(persisted.WasmKeys))
	for _, key := range persisted.WasmKeys {
		wasmKeys = append(wasmKeys, wasmKey{
			moduleHash: key.ModuleHash,
			target:     key.Target,
		})
	}
	return &chainTipRecording{
		record: &execution.RecordResult{
			Pos:       pos,
			BlockHash: persisted.BlockHash,
			Preimages: preimages,
		},
		firstHeaderNumber: persisted.FirstHeaderNumber,
		codeHashes:        persisted.CodeHashes,
		wasmKeys:          wasmKeys,
	}
}

func (d *keyValueBlockRecordsDatabase) writeRecording(recording *chainTipRecording) error {
	persisted, err := persistableChainTipRecording(recording)
	if err != nil {
		return err
	}
	encoded, err := rlp.EncodeToBytes(persisted)
	if err != nil {
		return fmt.Errorf("failed to encode chain-tip block record: %w", err)
	}
	if err := d.db.Put(blockRecordKey(recording.record.Pos), encoded); err != nil {
		return fmt.Errorf("failed to write chain-tip block record: %w", err)
	}
	return nil
}

func (d *keyValueBlockRecordsDatabase) readRecording(pos arbutil.MessageIndex) (*chainTipRecording, bool, error) {
	key := blockRecordKey(pos)
	exists, err := d.db.Has(key)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check chain-tip block record: %w", err)
	}
	if !exists {
		return nil, false, nil
	}
	encoded, err := d.db.Get(key)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read chain-tip block record: %w", err)
	}
	var persisted persistedChainTipRecording
	if err := rlp.DecodeBytes(encoded, &persisted); err != nil {
		return nil, false, fmt.Errorf("failed to decode chain-tip block record: %w", err)
	}
	return chainTipRecordingFromPersisted(pos, &persisted), true, nil
}

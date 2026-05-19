// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

const chainTipBlockRecordHotRetention = 256

var (
	blockRecordKeyPrefix      = []byte("ctbr:")
	blockRecordFreezerBaseKey = []byte("ctbr-freezer-base")
)

type blockRecordsDatabase interface {
	writeRecording(recording *chainTipRecording) error
	readRecording(pos arbutil.MessageIndex) (*chainTipRecording, bool, error)
	Close() error
}

type keyValueBlockRecordsDatabase struct {
	db      ethdb.KeyValueStore
	freezer ethdb.AncientStore
	lock    sync.Mutex
}

func newBlockRecordsDatabase(db ethdb.KeyValueStore) blockRecordsDatabase {
	return newBlockRecordsDatabaseWithFreezer(db, nil)
}

func newBlockRecordsDatabaseWithFreezer(db ethdb.KeyValueStore, freezer ethdb.AncientStore) blockRecordsDatabase {
	if db == nil {
		return nil
	}
	return &keyValueBlockRecordsDatabase{db: db, freezer: freezer}
}

func blockRecordKey(pos arbutil.MessageIndex) []byte {
	key := make([]byte, len(blockRecordKeyPrefix)+8)
	copy(key, blockRecordKeyPrefix)
	binary.BigEndian.PutUint64(key[len(blockRecordKeyPrefix):], uint64(pos))
	return key
}

func blockRecordPosFromKey(key []byte) arbutil.MessageIndex {
	return arbutil.MessageIndex(binary.BigEndian.Uint64(key[len(blockRecordKeyPrefix):]))
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
	d.lock.Lock()
	defer d.lock.Unlock()

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
	if err := d.freezeRecordings(recording.record.Pos); err != nil {
		return fmt.Errorf("failed to freeze chain-tip block records: %w", err)
	}
	return nil
}

func (d *keyValueBlockRecordsDatabase) Close() error {
	if d.freezer != nil {
		return d.freezer.Close()
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
		return d.readFrozenRecording(pos)
	}
	encoded, err := d.db.Get(key)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read chain-tip block record: %w", err)
	}
	return decodeChainTipRecording(encoded, pos)
}

func decodeChainTipRecording(encoded []byte, pos arbutil.MessageIndex) (*chainTipRecording, bool, error) {
	var persisted persistedChainTipRecording
	if err := rlp.DecodeBytes(encoded, &persisted); err != nil {
		return nil, false, fmt.Errorf("failed to decode chain-tip block record: %w", err)
	}
	return chainTipRecordingFromPersisted(pos, &persisted), true, nil
}

func (d *keyValueBlockRecordsDatabase) readFrozenRecording(pos arbutil.MessageIndex) (*chainTipRecording, bool, error) {
	if d.freezer == nil {
		return nil, false, nil
	}
	base, ok, err := d.readFreezerBase()
	if err != nil {
		return nil, false, err
	}
	if !ok || uint64(pos) < base {
		return nil, false, nil
	}
	count, err := d.freezer.Ancients()
	if err != nil {
		return nil, false, fmt.Errorf("failed to read chain-tip block records freezer count: %w", err)
	}
	if uint64(pos) >= base+count {
		return nil, false, nil
	}
	var encoded []byte
	if err := d.freezer.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		var err error
		encoded, err = reader.Ancient(rawdb.ChainTipBlockRecordsFreezerTable, uint64(pos)-base)
		return err
	}); err != nil {
		return nil, false, fmt.Errorf("failed to read frozen chain-tip block record: %w", err)
	}
	return decodeChainTipRecording(encoded, pos)
}

func (d *keyValueBlockRecordsDatabase) freezeRecordings(latestPos arbutil.MessageIndex) error {
	if d.freezer == nil || uint64(latestPos) <= chainTipBlockRecordHotRetention {
		return nil
	}
	cutoff := uint64(latestPos) - chainTipBlockRecordHotRetention
	base, hasBase, err := d.readFreezerBase()
	if err != nil {
		return err
	}
	count, err := d.freezer.Ancients()
	if err != nil {
		return fmt.Errorf("failed to read chain-tip block records freezer count: %w", err)
	}
	if !hasBase {
		if count != 0 {
			return fmt.Errorf("chain-tip block records freezer has %d items without a base", count)
		}
		firstPos, ok, err := d.firstFreezablePosition(cutoff)
		if err != nil || !ok {
			return err
		}
		base = firstPos
		if err := d.writeFreezerBase(base); err != nil {
			return err
		}
	} else if count > 0 && base+count-1 >= cutoff {
		return nil
	}
	nextPos := base + count
	if nextPos > cutoff {
		return nil
	}

	var (
		encodedRecords [][]byte
		positions      []uint64
	)
	for pos := nextPos; pos <= cutoff; pos++ {
		encoded, ok, err := d.readKeyValueRecordingBytes(arbutil.MessageIndex(pos))
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		encodedRecords = append(encodedRecords, encoded)
		positions = append(positions, pos)
	}
	if len(encodedRecords) == 0 {
		return nil
	}
	if _, err := d.freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		item := count
		for _, encoded := range encodedRecords {
			if err := writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, item, encoded); err != nil {
				return err
			}
			item++
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to append chain-tip block records to freezer: %w", err)
	}
	for _, pos := range positions {
		if err := d.db.Delete(blockRecordKey(arbutil.MessageIndex(pos))); err != nil {
			return fmt.Errorf("failed to delete frozen chain-tip block record %d from key-value database: %w", pos, err)
		}
	}
	return nil
}

func (d *keyValueBlockRecordsDatabase) firstFreezablePosition(cutoff uint64) (uint64, bool, error) {
	iterator := d.db.NewIterator(blockRecordKeyPrefix, nil)
	defer iterator.Release()
	if !iterator.Next() {
		return 0, false, iterator.Error()
	}
	pos := uint64(blockRecordPosFromKey(iterator.Key()))
	if pos > cutoff {
		return 0, false, nil
	}
	return pos, true, nil
}

func (d *keyValueBlockRecordsDatabase) readKeyValueRecordingBytes(pos arbutil.MessageIndex) ([]byte, bool, error) {
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
	return encoded, true, nil
}

func (d *keyValueBlockRecordsDatabase) readFreezerBase() (uint64, bool, error) {
	exists, err := d.db.Has(blockRecordFreezerBaseKey)
	if err != nil {
		return 0, false, fmt.Errorf("failed to check chain-tip block records freezer base: %w", err)
	}
	if !exists {
		return 0, false, nil
	}
	encoded, err := d.db.Get(blockRecordFreezerBaseKey)
	if err != nil {
		return 0, false, fmt.Errorf("failed to read chain-tip block records freezer base: %w", err)
	}
	if len(encoded) != 8 {
		return 0, false, fmt.Errorf("invalid chain-tip block records freezer base length %d", len(encoded))
	}
	return binary.BigEndian.Uint64(encoded), true, nil
}

func (d *keyValueBlockRecordsDatabase) writeFreezerBase(base uint64) error {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, base)
	if err := d.db.Put(blockRecordFreezerBaseKey, encoded); err != nil {
		return fmt.Errorf("failed to write chain-tip block records freezer base: %w", err)
	}
	return nil
}

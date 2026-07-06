// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package gethexec

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type freezerBlockRecordsDatabase struct {
	freezer ethdb.ResettableAncientStore

	lock                     sync.Mutex
	firstRecordedMsgIdx      uint64
	firstRecordedMsgIdxKnown bool
}

func newBlockRecordsDatabase(freezer ethdb.ResettableAncientStore) *freezerBlockRecordsDatabase {
	if freezer == nil {
		return nil
	}
	return &freezerBlockRecordsDatabase{freezer: freezer}
}

type persistedChainTipRecording struct {
	Pos               uint64
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
		Pos:               uint64(recording.record.Pos),
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

func chainTipRecordingFromPersisted(persisted *persistedChainTipRecording) *chainTipRecording {
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
			Pos:       arbutil.MessageIndex(persisted.Pos),
			BlockHash: persisted.BlockHash,
			Preimages: preimages,
		},
		firstHeaderNumber: persisted.FirstHeaderNumber,
		codeHashes:        persisted.CodeHashes,
		wasmKeys:          wasmKeys,
	}
}

func decodeChainTipRecording(encoded []byte) (*persistedChainTipRecording, error) {
	var persisted persistedChainTipRecording
	if err := rlp.DecodeBytes(encoded, &persisted); err != nil {
		return nil, fmt.Errorf("failed to decode chain-tip block record: %w", err)
	}
	return &persisted, nil
}

func (d *freezerBlockRecordsDatabase) freezerBounds() (uint64, uint64, error) {
	tail, err := d.freezer.Tail()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read chain-tip block records freezer tail: %w", err)
	}
	head, err := d.freezer.Ancients()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read chain-tip block records freezer count: %w", err)
	}
	if head < tail {
		return 0, 0, fmt.Errorf("chain-tip block records freezer head %d below tail %d", head, tail)
	}
	return tail, head, nil
}

func (d *freezerBlockRecordsDatabase) loadFirstRecordedMsgIdxLocked() (uint64, bool, error) {
	if d.firstRecordedMsgIdxKnown {
		return d.firstRecordedMsgIdx, true, nil
	}
	tail, head, err := d.freezerBounds()
	if err != nil {
		return 0, false, err
	}
	for item := tail; item < head; item++ {
		encoded, err := d.readFreezerItem(item)
		if err != nil {
			log.Error("Skipping unreadable chain-tip block record during recovery", "item", item, "err", err)
			continue
		}
		persisted, err := decodeChainTipRecording(encoded)
		if err != nil {
			log.Error("Skipping undecodable chain-tip block record during recovery", "item", item, "err", err)
			continue
		}
		if persisted.Pos < item {
			log.Error("Skipping inconsistent chain-tip block record during recovery", "item", item, "pos", persisted.Pos)
			continue
		}
		d.firstRecordedMsgIdx = persisted.Pos - item
		d.firstRecordedMsgIdxKnown = true
		return d.firstRecordedMsgIdx, true, nil
	}
	return 0, false, nil
}

func (d *freezerBlockRecordsDatabase) readFreezerItem(item uint64) ([]byte, error) {
	var encoded []byte
	if err := d.freezer.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		var err error
		encoded, err = reader.Ancient(rawdb.ChainTipBlockRecordsFreezerTable, item)
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to read chain-tip block record from freezer item %d: %w", item, err)
	}
	return encoded, nil
}

func (d *freezerBlockRecordsDatabase) writeRecording(recording *chainTipRecording) error {
	persisted, err := persistableChainTipRecording(recording)
	if err != nil {
		return err
	}
	encoded, err := rlp.EncodeToBytes(persisted)
	if err != nil {
		return fmt.Errorf("failed to encode chain-tip block record: %w", err)
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	pos := uint64(recording.record.Pos)
	firstRecordedMsgIdx, known, err := d.loadFirstRecordedMsgIdxLocked()
	if err != nil {
		return err
	}
	tail, head, err := d.freezerBounds()
	if err != nil {
		return err
	}
	switch {
	case known && pos == firstRecordedMsgIdx+head:
		// Continuous append
	case known && pos >= firstRecordedMsgIdx+tail && pos < firstRecordedMsgIdx+head:
		// The position is already recorded, messages are being replayed is rewriting history
		if _, err := d.freezer.TruncateHead(pos - firstRecordedMsgIdx); err != nil {
			return fmt.Errorf("failed to roll back chain-tip block records freezer to message index %d: %w", pos, err)
		}
	default:
		if head > 0 {
			log.Warn("Resetting chain-tip block records freezer to restart recording", "pos", pos, "tail", tail, "head", head)
			d.firstRecordedMsgIdxKnown = false
			if err := d.freezer.Reset(); err != nil {
				return fmt.Errorf("failed to reset chain-tip block records freezer: %w", err)
			}
		}
		firstRecordedMsgIdx = pos
	}
	if _, err := d.freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, pos-firstRecordedMsgIdx, encoded)
	}); err != nil {
		return fmt.Errorf("failed to append chain-tip block record to freezer: %w", err)
	}
	// Ensure the freezer on disk is within replayable bounds
	if err := d.freezer.SyncAncient(); err != nil {
		return fmt.Errorf("failed to sync chain-tip block records freezer: %w", err)
	}
	d.firstRecordedMsgIdx = firstRecordedMsgIdx
	d.firstRecordedMsgIdxKnown = true
	return nil
}

func (d *freezerBlockRecordsDatabase) readRecording(pos arbutil.MessageIndex) (*chainTipRecording, bool, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	firstRecordedMsgIdx, known, err := d.loadFirstRecordedMsgIdxLocked()
	if err != nil {
		return nil, false, err
	}
	if !known {
		return nil, false, nil
	}
	tail, head, err := d.freezerBounds()
	if err != nil {
		return nil, false, err
	}
	if uint64(pos) < firstRecordedMsgIdx+tail || uint64(pos) >= firstRecordedMsgIdx+head {
		return nil, false, nil
	}
	encoded, err := d.readFreezerItem(uint64(pos) - firstRecordedMsgIdx)
	if err != nil {
		log.Error("Treating unreadable chain-tip block record as missing", "pos", pos, "err", err)
		return nil, false, nil
	}
	persisted, err := decodeChainTipRecording(encoded)
	if err != nil {
		log.Error("Treating undecodable chain-tip block record as missing", "pos", pos, "err", err)
		return nil, false, nil
	}
	if persisted.Pos != uint64(pos) {
		return nil, false, fmt.Errorf("chain-tip block record at freezer item %d has message index %d, expected %d", uint64(pos)-firstRecordedMsgIdx, persisted.Pos, pos)
	}
	return chainTipRecordingFromPersisted(persisted), true, nil
}

func (d *freezerBlockRecordsDatabase) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.freezer.Close()
}

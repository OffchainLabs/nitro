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

type blockRecordsFreezer struct {
	freezer ethdb.ResettableAncientStore

	lock sync.Mutex
}

func newBlockRecordsFreezer(freezer ethdb.ResettableAncientStore) *blockRecordsFreezer {
	if freezer == nil {
		return nil
	}
	return &blockRecordsFreezer{freezer: freezer}
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

func (d *blockRecordsFreezer) freezerBounds() (uint64, uint64, error) {
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

func (d *blockRecordsFreezer) readFreezerItem(item uint64) ([]byte, error) {
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

func (d *blockRecordsFreezer) writeRecording(recording *chainTipRecording) error {
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
	tail, head, err := d.freezerBounds()
	if err != nil {
		return err
	}
	switch {
	case pos == head:
		// Continuous append
	case pos >= tail && pos < head:
		// The position is already recorded, messages are being replayed is rewriting history
		if _, err := d.freezer.TruncateHead(pos); err != nil {
			return fmt.Errorf("failed to roll back chain-tip block records freezer to message index %d: %w", pos, err)
		}
	case pos > head && head == tail:
		// The freezer is empty and behind pos advance the empty window so the next append lands at pos.
		if _, err := d.freezer.TruncateTail(pos); err != nil {
			return fmt.Errorf("failed to advance chain-tip block records freezer to message index %d: %w", pos, err)
		}
	case pos > head:
		// Recordings for [head, pos) were lost refuse to record so the stored recordings survive and the operator can recover.
		return fmt.Errorf("chain-tip block records freezer is missing recordings for message indexes %d to %d and refuses to record message index %d; reorg the node to message index %d or below to re-record the gap, or remove the chain-tip block records ancient database to restart recording from the chain head", head, pos-1, pos, head)
	default:
		// pos is below the pruned tail everything stored is stale after the reorg, restart recording at pos.
		log.Warn("Resetting chain-tip block records freezer to restart recording", "pos", pos, "tail", tail, "head", head)
		if err := d.freezer.Reset(); err != nil {
			return fmt.Errorf("failed to reset chain-tip block records freezer: %w", err)
		}
		if pos > 0 {
			if _, err := d.freezer.TruncateTail(pos); err != nil {
				return fmt.Errorf("failed to advance chain-tip block records freezer to message index %d: %w", pos, err)
			}
		}
	}
	if _, err := d.freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, pos, encoded)
	}); err != nil {
		return fmt.Errorf("failed to append chain-tip block record to freezer: %w", err)
	}
	// Ensure the freezer on disk is within replayable bounds
	if err := d.freezer.SyncAncient(); err != nil {
		return fmt.Errorf("failed to sync chain-tip block records freezer: %w", err)
	}
	return nil
}

func (d *blockRecordsFreezer) readRecording(pos arbutil.MessageIndex) (*chainTipRecording, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	tail, head, err := d.freezerBounds()
	if err != nil {
		return nil, err
	}
	if uint64(pos) < tail || uint64(pos) >= head {
		return nil, nil
	}
	encoded, err := d.readFreezerItem(uint64(pos))
	if err != nil {
		log.Error("Treating unreadable chain-tip block record as missing", "pos", pos, "err", err)
		return nil, nil
	}
	persisted, err := decodeChainTipRecording(encoded)
	if err != nil {
		log.Error("Treating undecodable chain-tip block record as missing", "pos", pos, "err", err)
		return nil, nil
	}
	if persisted.Pos != uint64(pos) {
		return nil, fmt.Errorf("chain-tip block record at freezer item %d has message index %d, expected %d", uint64(pos), persisted.Pos, pos)
	}
	return chainTipRecordingFromPersisted(persisted), nil
}

func (d *blockRecordsFreezer) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.freezer.Close()
}

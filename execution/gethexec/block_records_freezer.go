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

func (s *blockRecordsFreezer) freezerBounds() (uint64, uint64, error) {
	tail, err := s.freezer.Tail(rawdb.ChainTipBlockRecordsGroup)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read chain-tip block records freezer tail: %w", err)
	}
	head, err := s.freezer.Ancients()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read chain-tip block records freezer count: %w", err)
	}
	if head < tail {
		return 0, 0, fmt.Errorf("chain-tip block records freezer head %d below tail %d", head, tail)
	}
	return tail, head, nil
}

func (s *blockRecordsFreezer) readFreezerItem(item uint64) ([]byte, error) {
	var encoded []byte
	if err := s.freezer.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		var err error
		encoded, err = reader.Ancient(rawdb.ChainTipBlockRecordsFreezerTable, item)
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to read chain-tip block record from freezer item %d: %w", item, err)
	}
	return encoded, nil
}

func (s *blockRecordsFreezer) writeRecording(recording *chainTipRecording) error {
	persisted, err := persistableChainTipRecording(recording)
	if err != nil {
		return err
	}
	encoded, err := rlp.EncodeToBytes(persisted)
	if err != nil {
		return fmt.Errorf("failed to encode chain-tip block record: %w", err)
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	pos := uint64(recording.record.Pos)
	tail, head, err := s.freezerBounds()
	if err != nil {
		return err
	}
	switch {
	case pos == head:
		// Continuous append
	case pos >= tail && pos < head:
		// The position is already recorded, messages are being replayed is rewriting history
		if _, err := s.freezer.TruncateHead(pos); err != nil {
			return fmt.Errorf("failed to roll back chain-tip block records freezer to message index %d: %w", pos, err)
		}
	case pos > head && head == tail:
		// The freezer is empty and behind pos advance the empty window so the next append lands at pos.
		if _, err := s.freezer.TruncateTail(rawdb.ChainTipBlockRecordsGroup, pos); err != nil {
			return fmt.Errorf("failed to advance chain-tip block records freezer to message index %d: %w", pos, err)
		}
		if head > 0 || pos > 1 {
			log.Warn("Chain-tip block records freezer is starting above the messages below it, which can no longer be validated",
				"unrecordedFrom", head, "unrecordedTo", pos-1, "recordingFrom", pos)
		}
	case pos > head:
		// Recordings for [head, pos) were lost refuse to record so the stored recordings survive and the operator can recover.
		return fmt.Errorf("chain-tip block records freezer is missing recordings for message indexes %d to %d, so message index %d cannot be recorded: reorg the node to the last recorded message index or below to re-record the gap, or remove the chain-tip block records ancient database to restart recording from the chain head", head, pos-1, pos)
	default:
		// pos is below the pruned tail everything stored is stale after the reorg, restart recording at pos.
		if head == tail {
			log.Warn("Restarting chain-tip block recording below the empty freezer window", "tail", tail, "recordingFrom", pos)
		} else {
			log.Warn("Discarding all stored chain-tip block records to restart recording below the freezer tail, so the discarded messages can no longer be validated",
				"discardedFrom", tail, "discardedTo", head-1, "recordingFrom", pos)
		}
		if err := s.freezer.Reset(); err != nil {
			return fmt.Errorf("failed to reset chain-tip block records freezer: %w", err)
		}
		if pos > 0 {
			if _, err := s.freezer.TruncateTail(rawdb.ChainTipBlockRecordsGroup, pos); err != nil {
				return fmt.Errorf("failed to advance chain-tip block records freezer to message index %d: %w", pos, err)
			}
		}
	}
	if _, err := s.freezer.ModifyAncients(func(writer ethdb.AncientWriteOp) error {
		return writer.AppendRaw(rawdb.ChainTipBlockRecordsFreezerTable, pos, encoded)
	}); err != nil {
		return fmt.Errorf("failed to append chain-tip block record to freezer: %w", err)
	}
	// Ensure the freezer on disk is within replayable bounds
	if err := s.freezer.SyncAncient(); err != nil {
		return fmt.Errorf("failed to sync chain-tip block records freezer: %w", err)
	}
	return nil
}

func (s *blockRecordsFreezer) readRecording(pos arbutil.MessageIndex) (*chainTipRecording, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	tail, head, err := s.freezerBounds()
	if err != nil {
		return nil, err
	}
	if uint64(pos) < tail {
		return nil, fmt.Errorf("chain-tip block recording for message index %d is below the freezer tail and can never be served: recordings begin at message index %d", pos, tail)
	}
	if uint64(pos) >= head {
		return nil, nil
	}
	encoded, err := s.readFreezerItem(uint64(pos))
	if err != nil {
		return nil, err
	}
	var persisted persistedChainTipRecording
	if err := rlp.DecodeBytes(encoded, &persisted); err != nil {
		return nil, fmt.Errorf("failed to decode chain-tip block record for message index %d: %w", pos, err)
	}
	if persisted.Pos != uint64(pos) {
		return nil, fmt.Errorf("chain-tip block record at freezer item %d has message index %d, expected %d", uint64(pos), persisted.Pos, pos)
	}
	return chainTipRecordingFromPersisted(&persisted), nil
}

// pruneRecordingsBefore deletes the recordings for all message indices below
// pos by advancing the freezer tail. The tail cannot move past the head, so
// pruning targets beyond the last recording empty the freezer instead.
func (s *blockRecordsFreezer) pruneRecordingsBefore(pos arbutil.MessageIndex) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, head, err := s.freezerBounds()
	if err != nil {
		return err
	}
	if _, err := s.freezer.TruncateTail(rawdb.ChainTipBlockRecordsGroup, min(uint64(pos), head)); err != nil {
		return fmt.Errorf("failed to prune chain-tip block records freezer: %w", err)
	}
	return nil
}

func (s *blockRecordsFreezer) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.freezer.Close()
}

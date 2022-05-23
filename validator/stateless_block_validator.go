// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"fmt"
	"github.com/offchainlabs/nitro/arbutil"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/pkg/errors"
)

type StatelessBlockValidator struct {
	MachineLoader   *NitroMachineLoader
	inboxReader     InboxReaderInterface
	inboxTracker    InboxTrackerInterface
	streamer        TransactionStreamerInterface
	blockchain      *core.BlockChain
	db              ethdb.Database
	daService       arbstate.SimpleDASReader
	genesisBlockNum uint64
}

type BlockValidatorRegistrer interface {
	SetBlockValidator(*BlockValidator)
}

type InboxTrackerInterface interface {
	BlockValidatorRegistrer
	GetDelayedMessageBytes(uint64) ([]byte, error)
	GetBatchMessageCount(seqNum uint64) (arbutil.MessageIndex, error)
	GetBatchAcc(seqNum uint64) (common.Hash, error)
	GetBatchCount() (uint64, error)
}

type TransactionStreamerInterface interface {
	BlockValidatorRegistrer
	GetMessage(seqNum arbutil.MessageIndex) (arbstate.MessageWithMetadata, error)
	GetGenesisBlockNumber() (uint64, error)
	PauseReorgs()
	ResumeReorgs()
}

type InboxReaderInterface interface {
	GetSequencerMessageBytes(ctx context.Context, seqNum uint64) ([]byte, error)
}

type L1ReaderInterface interface {
	Client() arbutil.L1Interface
	Subscribe(bool) (<-chan *types.Header, func())
	WaitForTxApproval(ctx context.Context, tx *types.Transaction) (*types.Receipt, error)
}

type GlobalStatePosition struct {
	BatchNumber uint64
	PosInBatch  uint64
}

func GlobalStatePositionsFor(tracker InboxTrackerInterface, pos arbutil.MessageIndex, batch uint64) (GlobalStatePosition, GlobalStatePosition, error) {
	msgCountInBatch, err := tracker.GetBatchMessageCount(batch)
	if err != nil {
		return GlobalStatePosition{}, GlobalStatePosition{}, err
	}
	var firstInBatch arbutil.MessageIndex
	if batch > 0 {
		firstInBatch, err = tracker.GetBatchMessageCount(batch - 1)
		if err != nil {
			return GlobalStatePosition{}, GlobalStatePosition{}, err
		}
	}
	if msgCountInBatch <= pos {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d has up to message %d, failed getting for %d", batch, msgCountInBatch-1, pos)
	}
	if firstInBatch > pos {
		return GlobalStatePosition{}, GlobalStatePosition{}, fmt.Errorf("batch %d starts from %d, failed getting for %d", batch, firstInBatch, pos)
	}
	startPos := GlobalStatePosition{batch, uint64(pos - firstInBatch)}
	if msgCountInBatch == pos+1 {
		return startPos, GlobalStatePosition{batch + 1, 0}, nil
	}
	return startPos, GlobalStatePosition{batch, uint64(pos + 1 - firstInBatch)}, nil
}

func FindBatchContainingMessageIndex(tracker InboxTrackerInterface, pos arbutil.MessageIndex, high uint64) (uint64, error) {
	var low uint64
	// Iteration preconditions:
	// - high >= low
	// - msgCount(low - 1) <= pos implies low <= target
	// - msgCount(high) > pos implies high >= target
	// Therefore, if low == high, then low == high == target
	for high > low {
		// Due to integer rounding, mid >= low && mid < high
		mid := (low + high) / 2
		count, err := tracker.GetBatchMessageCount(mid)
		if err != nil {
			return 0, err
		}
		if count < pos {
			// Must narrow as mid >= low, therefore mid + 1 > low, therefore newLow > oldLow
			// Keeps low precondition as msgCount(mid) < pos
			low = mid + 1
		} else if count == pos {
			return mid + 1, nil
		} else if count == pos+1 || mid == low { // implied: count > pos
			return mid, nil
		} else { // implied: count > pos + 1
			// Must narrow as mid < high, therefore newHigh < lowHigh
			// Keeps high precondition as msgCount(mid) > pos
			high = mid
		}
	}
	return low, nil
}

type validationEntry struct {
	BlockNumber   uint64
	PrevBlockHash common.Hash
	BlockHash     common.Hash
	SendRoot      common.Hash
	PrevSendRoot  common.Hash
	BlockHeader   *types.Header
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	StartPosition GlobalStatePosition
	EndPosition   GlobalStatePosition
	Preimages     map[common.Hash][]byte
}

func (v *validationEntry) start() GoGlobalState {
	start := v.StartPosition
	return GoGlobalState{
		Batch:      start.BatchNumber,
		PosInBatch: start.PosInBatch,
		BlockHash:  v.PrevBlockHash,
		SendRoot:   v.PrevSendRoot,
	}
}

func (v *validationEntry) expectedEnd() GoGlobalState {
	end := v.EndPosition
	return GoGlobalState{
		Batch:      end.BatchNumber,
		PosInBatch: end.PosInBatch,
		BlockHash:  v.BlockHash,
		SendRoot:   v.SendRoot,
	}
}

func newValidationEntry(
	prevHeader *types.Header,
	header *types.Header,
	hasDelayed bool,
	delayedMsgNr uint64,
	preimages map[common.Hash][]byte,
) (*validationEntry, error) {
	extraInfo, err := types.DeserializeHeaderExtraInformation(header)
	if err != nil {
		return nil, err
	}
	prevExtraInfo, err := types.DeserializeHeaderExtraInformation(prevHeader)
	if err != nil {
		return nil, err
	}
	return &validationEntry{
		BlockNumber:   header.Number.Uint64(),
		BlockHash:     header.Hash(),
		SendRoot:      extraInfo.SendRoot,
		PrevSendRoot:  prevExtraInfo.SendRoot,
		PrevBlockHash: header.ParentHash,
		BlockHeader:   header,
		HasDelayedMsg: hasDelayed,
		DelayedMsgNr:  delayedMsgNr,
		Preimages:     preimages,
	}, nil
}

func NewStatelessBlockValidator(
	machineLoader *NitroMachineLoader,
	inboxReader InboxReaderInterface,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	blockchain *core.BlockChain,
	db ethdb.Database,
	das arbstate.SimpleDASReader,
) (*StatelessBlockValidator, error) {
	genesisBlockNum, err := streamer.GetGenesisBlockNumber()
	if err != nil {
		return nil, err
	}
	validator := &StatelessBlockValidator{
		MachineLoader:   machineLoader,
		inboxReader:     inboxReader,
		inboxTracker:    inbox,
		streamer:        streamer,
		blockchain:      blockchain,
		db:              db,
		daService:       das,
		genesisBlockNum: genesisBlockNum,
	}
	return validator, nil
}

// If msg is nil, this will record block creation up to the point where message would be accessed (for a "too far" proof)
func RecordBlockCreation(blockchain *core.BlockChain, prevHeader *types.Header, msg *arbstate.MessageWithMetadata) (common.Hash, map[common.Hash][]byte, error) {
	recordingdb, chaincontext, recordingKV, err := arbitrum.PrepareRecording(blockchain, prevHeader)
	if err != nil {
		return common.Hash{}, nil, err
	}

	chainConfig := blockchain.Config()

	// Get the chain ID, both to validate and because the replay binary also gets the chain ID,
	// so we need to populate the recordingdb with preimages for retrieving the chain ID.
	if prevHeader != nil {
		initialArbosState, err := arbosState.OpenSystemArbosState(recordingdb, nil, true)
		if err != nil {
			return common.Hash{}, nil, fmt.Errorf("error opening initial ArbOS state: %w", err)
		}
		chainId, err := initialArbosState.ChainId()
		if err != nil {
			return common.Hash{}, nil, fmt.Errorf("error getting chain ID from initial ArbOS state: %w", err)
		}
		if chainId.Cmp(chainConfig.ChainID) != 0 {
			return common.Hash{}, nil, fmt.Errorf("unexpected chain ID %v in ArbOS state, expected %v", chainId, chainConfig.ChainID)
		}
	}

	var blockHash common.Hash
	if msg != nil {
		block, _ := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			prevHeader,
			recordingdb,
			chaincontext,
			chainConfig,
		)
		blockHash = block.Hash()
	}

	preimages, err := arbitrum.PreimagesFromRecording(chaincontext, recordingKV)

	return blockHash, preimages, err
}

func BlockDataForValidation(blockchain *core.BlockChain, header, prevHeader *types.Header, msg arbstate.MessageWithMetadata, producePreimages bool) (preimages map[common.Hash][]byte, hasDelayedMessage bool, delayedMsgNr uint64, err error) {
	var prevHash common.Hash
	if prevHeader != nil {
		prevHash = prevHeader.Hash()
	}
	if header.ParentHash != prevHash {
		err = fmt.Errorf("bad arguments: prev does not match")
		return
	}

	if prevHeader != nil && producePreimages {
		var blockhash common.Hash
		blockhash, preimages, err = RecordBlockCreation(blockchain, prevHeader, &msg)
		if err != nil {
			return
		}
		if blockhash != header.Hash() {
			err = fmt.Errorf("wrong hash expected %s got %s", header.Hash(), blockhash)
			return
		}
	}

	if prevHeader == nil || header.Nonce != prevHeader.Nonce {
		hasDelayedMessage = true
		if prevHeader != nil {
			delayedMsgNr = prevHeader.Nonce.Uint64()
		}
	}

	return
}

func SetMachinePreimageResolver(ctx context.Context, mach *ArbitratorMachine, preimages map[common.Hash][]byte, seqMsg []byte, bc *core.BlockChain, das arbstate.SimpleDASReader) error {
	recordNewPreimages := true
	if preimages == nil {
		preimages = make(map[common.Hash][]byte)
		recordNewPreimages = false
	}

	if arbstate.IsDASMessageHeaderByte(seqMsg[40]) {
		if das == nil {
			log.Error("No DAS configured, but sequencer message found with DAS header")
			if bc.Config().ArbitrumChainParams.DataAvailabilityCommittee {
				return errors.New("processing data availability chain without DAS configured")
			}
		} else {
			_, err := arbstate.RecoverPayloadFromDasBatch(ctx, seqMsg, das, preimages)
			if err != nil {
				return err
			}
		}
	}

	db := bc.StateCache().TrieDB()
	return mach.SetPreimageResolver(func(hash common.Hash) ([]byte, error) {
		// Check if it's a known preimage
		if preimage, ok := preimages[hash]; ok {
			return preimage, nil
		}
		// Check if it's part of the state trie
		preimage, err := db.Node(hash)
		if err != nil {
			// Check if it's a code hash
			codeKey := append([]byte{}, rawdb.CodePrefix...)
			codeKey = append(codeKey, hash.Bytes()...)
			preimage, err = db.DiskDB().Get(codeKey)
		}
		if err != nil {
			// Check if it's a block hash
			header := bc.GetHeaderByHash(hash)
			if header != nil {
				preimage, err = rlp.EncodeToBytes(header)
			}
		}
		if err == nil && recordNewPreimages {
			preimages[hash] = preimage
		}
		return preimage, err
	})
}

func (v *StatelessBlockValidator) executeBlock(ctx context.Context, entry *validationEntry, seqMsg []byte, moduleRoot common.Hash) (GoGlobalState, []byte, error) {
	start := entry.StartPosition
	gsStart := entry.start()

	basemachine, err := v.MachineLoader.GetMachine(ctx, moduleRoot, true)
	if err != nil {
		return GoGlobalState{}, nil, fmt.Errorf("unabled to get WASM machine: %w", err)
	}
	mach := basemachine.Clone()
	err = SetMachinePreimageResolver(ctx, mach, entry.Preimages, seqMsg, v.blockchain, v.daService)
	if err != nil {
		return GoGlobalState{}, nil, err
	}
	err = mach.SetGlobalState(gsStart)
	if err != nil {
		log.Error("error while setting global state for proving", "err", err, "gsStart", gsStart)
		return GoGlobalState{}, nil, errors.New("error while setting global state for proving")
	}
	err = mach.AddSequencerInboxMessage(start.BatchNumber, seqMsg)
	if err != nil {
		log.Error("error while trying to add sequencer msg for proving", "err", err, "seq", start.BatchNumber, "blockNr", entry.BlockNumber)
		return GoGlobalState{}, nil, errors.New("error while trying to add sequencer msg for proving")
	}
	var delayedMsg []byte
	if entry.HasDelayedMsg {
		delayedMsg, err = v.inboxTracker.GetDelayedMessageBytes(entry.DelayedMsgNr)
		if err != nil {
			log.Error("error while trying to read delayed msg for proving", "err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.BlockNumber)
			return GoGlobalState{}, nil, errors.New("error while trying to read delayed msg for proving")
		}
		err = mach.AddDelayedInboxMessage(entry.DelayedMsgNr, delayedMsg)
		if err != nil {
			log.Error("error while trying to add delayed msg for proving", "err", err, "seq", entry.DelayedMsgNr, "blockNr", entry.BlockNumber)
			return GoGlobalState{}, nil, errors.New("error while trying to add delayed msg for proving")
		}
	}

	var steps uint64
	for mach.IsRunning() {
		var count uint64 = 500000000
		err = mach.Step(ctx, count)
		if steps > 0 {
			log.Debug("validation", "moduleRoot", moduleRoot, "block", entry.BlockNumber, "steps", steps)
		}
		if err != nil {
			return GoGlobalState{}, nil, fmt.Errorf("machine execution failed with error: %w", err)
		}
		steps += count
	}
	if mach.IsErrored() {
		log.Error("machine entered errored state during attempted validation", "block", entry.BlockNumber)
		return GoGlobalState{}, nil, errors.New("machine entered errored state during attempted validation")
	}
	return mach.GetGlobalState(), delayedMsg, nil
}

func (v *StatelessBlockValidator) ValidateBlock(ctx context.Context, header *types.Header, moduleRoot common.Hash) (bool, error) {
	if header == nil {
		return false, errors.New("header not found")
	}
	blockNum := header.Number.Uint64()
	msgIndex := arbutil.BlockNumberToMessageCount(blockNum, v.genesisBlockNum) - 1
	prevHeader := v.blockchain.GetHeaderByNumber(blockNum - 1)
	if prevHeader == nil {
		return false, errors.New("prev header not found")
	}
	msg, err := v.streamer.GetMessage(msgIndex)
	if err != nil {
		return false, err
	}
	preimages, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(v.blockchain, header, prevHeader, msg, false)
	if err != nil {
		return false, fmt.Errorf("failed to get block data to validate: %w", err)
	}

	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return false, err
	}
	batch, err := FindBatchContainingMessageIndex(v.inboxTracker, msgIndex, batchCount)
	if err != nil {
		return false, err
	}

	startPos, endPos, err := GlobalStatePositionsFor(v.inboxTracker, msgIndex, batch)
	if err != nil {
		return false, fmt.Errorf("failed calculating position for validation: %w", err)
	}

	entry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead, preimages)
	if err != nil {
		return false, fmt.Errorf("failed to create validation entry %w", err)
	}
	entry.StartPosition = startPos
	entry.EndPosition = endPos

	seqMsg, err := v.inboxReader.GetSequencerMessageBytes(ctx, startPos.BatchNumber)
	if err != nil {
		return false, err
	}

	gsEnd, _, err := v.executeBlock(ctx, entry, seqMsg, moduleRoot)
	if err != nil {
		return false, err
	}
	return gsEnd == entry.expectedEnd(), nil
}

//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das"
	"github.com/pkg/errors"
)

type StatelessBlockValidator struct {
	inboxReader     InboxReaderInterface
	inboxTracker    InboxTrackerInterface
	streamer        TransactionStreamerInterface
	blockchain      *core.BlockChain
	db              ethdb.Database
	das             das.DataAvailabilityService
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
	SeqMsgNr      uint64
	StartPosition GlobalStatePosition
	EndPosition   GlobalStatePosition
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
	}, nil
}

func NewStatelessBlockValidator(
	inboxReader InboxReaderInterface,
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
	blockchain *core.BlockChain,
	db ethdb.Database,
	das das.DataAvailabilityService,
) (*StatelessBlockValidator, error) {
	CreateHostIoMachine()
	genesisBlockNum, err := streamer.GetGenesisBlockNumber()
	if err != nil {
		return nil, err
	}
	validator := &StatelessBlockValidator{
		inboxReader:     inboxReader,
		inboxTracker:    inbox,
		streamer:        streamer,
		blockchain:      blockchain,
		db:              db,
		das:             das,
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
		initialArbosState, err := arbosState.OpenSystemArbosState(recordingdb, true)
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

func BlockDataForValidation(blockchain *core.BlockChain, header, prevHeader *types.Header, msg arbstate.MessageWithMetadata) (preimages map[common.Hash][]byte, hasDelayedMessage bool, delayedMsgNr uint64, err error) {
	var prevHash common.Hash
	if prevHeader != nil {
		prevHash = prevHeader.Hash()
	}
	if header.ParentHash != prevHash {
		err = fmt.Errorf("bad arguments: prev does not match")
		return
	}

	var blockhash common.Hash
	blockhash, preimages, err = RecordBlockCreation(blockchain, prevHeader, &msg)
	if err != nil {
		return
	}
	if blockhash != header.Hash() {
		err = fmt.Errorf("wrong hash expected %s got %s", header.Hash(), blockhash)
		return
	}
	if prevHeader == nil || header.Nonce != prevHeader.Nonce {
		hasDelayedMessage = true
		if prevHeader != nil {
			delayedMsgNr = prevHeader.Nonce.Uint64()
		}
	}
	return
}

func (v *StatelessBlockValidator) executeBlock(ctx context.Context, entry *validationEntry, preimages map[common.Hash][]byte, seqMsg []byte) (GoGlobalState, []byte, error) {
	start := entry.StartPosition
	gsStart := entry.start()

	if arbstate.IsDASMessageHeaderByte(seqMsg[40]) {
		if v.das == nil {
			log.Error("No DAS configured, but sequencer message found with DAS header")
			if v.blockchain.Config().ArbitrumChainParams.DataAvailabilityCommittee {
				return GoGlobalState{}, nil, errors.New("processing data availability chain without DAS configured")
			}
		} else {
			cert, _, err := arbstate.DeserializeDASCertFrom(seqMsg[40:])
			if err != nil {
				log.Error("Failed to deserialize DAS message", "err", err)
			} else {
				preimages[common.BytesToHash(cert.DataHash[:])], err = v.das.Retrieve(ctx, seqMsg[40:])
				if err != nil {
					return GoGlobalState{}, nil, fmt.Errorf("couldn't retrieve message from DAS %w", err)
				}
			}
		}
	}

	basemachine, err := GetHostIoMachine(ctx)
	if err != nil {
		return GoGlobalState{}, nil, fmt.Errorf("unabled to get WASM machine: %w", err)
	}
	mach := basemachine.Clone()
	err = mach.AddPreimages(preimages)
	if err != nil {
		log.Error("error while adding preimage for proving", "err", err, "gsStart", gsStart)
		return GoGlobalState{}, nil, errors.New("error while adding preimage for proving ")
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
			log.Info("validation", "block", entry.BlockNumber, "steps", steps)
		}
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				log.Error("running machine failed", "err", err)
				panic("Failed to run machine: " + err.Error())
			}
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

func (v *StatelessBlockValidator) ValidateBlock(ctx context.Context, blockNum uint64) error {
	msgIndex := arbutil.BlockNumberToMessageCount(blockNum, v.genesisBlockNum)
	header := v.blockchain.GetHeaderByNumber(blockNum)
	if header == nil {
		return errors.New("header not found")
	}
	prevHeader := v.blockchain.GetHeaderByNumber(blockNum - 1)
	if prevHeader == nil {
		return errors.New("prev header not found")
	}
	msg, err := v.streamer.GetMessage(msgIndex)
	if err != nil {
		return err
	}
	preimages, hasDelayedMessage, delayedMsgToRead, err := BlockDataForValidation(v.blockchain, header, prevHeader, msg)
	if err != nil {
		return errors.New("failed to get block data to validate")
	}

	batchCount, err := v.inboxTracker.GetBatchCount()
	if err != nil {
		return err
	}
	batch, err := FindBatchContainingMessageIndex(v.inboxTracker, msgIndex, batchCount)
	if err != nil {
		return err
	}

	startPos, endPos, err := GlobalStatePositionsFor(v.inboxTracker, msgIndex, batch)
	if err != nil {
		return fmt.Errorf("failed calculating position for validation: %w", err)
	}

	entry, err := newValidationEntry(prevHeader, header, hasDelayedMessage, delayedMsgToRead)
	if err != nil {
		return fmt.Errorf("failed to create validation entry %w", err)
	}
	entry.SeqMsgNr = startPos.BatchNumber
	entry.StartPosition = startPos
	entry.EndPosition = endPos

	seqMsg, err := v.inboxReader.GetSequencerMessageBytes(ctx, startPos.BatchNumber)
	if err != nil {
		return err
	}

	gsEnd, _, err := v.executeBlock(ctx, entry, preimages, seqMsg)
	if err != nil {
		return err
	}
	fmt.Println("Finished validation:", gsEnd == entry.expectedEnd())
	return nil
}

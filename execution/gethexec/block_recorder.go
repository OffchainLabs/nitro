package gethexec

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/validator"
)

// BlockRecorder uses a separate statedatabase from the blockchain.
// It has access to any state in the ethdb (hard-disk) database, and can compute state as needed.
// We keep references for state of:
// Any block that matches PrepareForRecord that was done recently (according to PrepareDelay config)
// Most recent/advanced header we ever computed (lastHdr)
// Hopefully - some recent valid block. For that we always keep one candidate block until it becomes validated.
type BlockRecorder struct {
	recordingDatabase *arbitrum.RecordingDatabase
	execEngine        *ExecutionEngine

	lastHdr     *types.Header
	lastHdrLock sync.Mutex

	validHdrCandidate *types.Header
	validHdr          *types.Header
	validHdrLock      sync.Mutex

	preparedQueue []*types.Header
	preparedLock  sync.Mutex
}

func NewBlockRecorder(config *arbitrum.RecordingDatabaseConfig, execEngine *ExecutionEngine, ethDb ethdb.Database) *BlockRecorder {
	recorder := &BlockRecorder{
		execEngine:        execEngine,
		recordingDatabase: arbitrum.NewRecordingDatabase(config, ethDb, execEngine.bc),
	}
	execEngine.SetRecorder(recorder)
	return recorder
}

func stateLogFunc(targetHeader, header *types.Header, hasState bool) {
	if targetHeader == nil || header == nil {
		return
	}
	gap := targetHeader.Number.Int64() - header.Number.Int64()
	step := int64(500)
	stage := "computing state"
	if !hasState {
		step = 3000
		stage = "looking for full block"
	}
	if (gap >= step) && (gap%step == 0) {
		log.Info("Setting up validation", "stage", stage, "current", header.Number, "target", targetHeader.Number)
	}
}

// If msg is nil, this will record block creation up to the point where message would be accessed (for a "too far" proof)
// If keepreference == true, reference to state of prevHeader is added (no reference added if an error is returned)
func (r *BlockRecorder) RecordBlockCreation(
	ctx context.Context,
	pos arbutil.MessageIndex,
	msg *arbostypes.MessageWithMetadata,
) (*execution.RecordResult, error) {

	blockNum := r.execEngine.MessageIndexToBlockNumber(pos)

	var prevHeader *types.Header
	if pos != 0 {
		prevHeader = r.execEngine.bc.GetHeaderByNumber(uint64(blockNum - 1))
		if prevHeader == nil {
			return nil, fmt.Errorf("pos %d prevHeader not found", pos)
		}
	}

	recordingdb, chaincontext, recordingKV, err := r.recordingDatabase.PrepareRecording(ctx, prevHeader, stateLogFunc)
	if err != nil {
		return nil, err
	}
	defer func() { r.recordingDatabase.Dereference(prevHeader) }()

	chainConfig := r.execEngine.bc.Config()

	// Get the chain ID, both to validate and because the replay binary also gets the chain ID,
	// so we need to populate the recordingdb with preimages for retrieving the chain ID.
	if prevHeader != nil {
		initialArbosState, err := arbosState.OpenSystemArbosState(recordingdb, nil, true)
		if err != nil {
			return nil, fmt.Errorf("error opening initial ArbOS state: %w", err)
		}
		chainId, err := initialArbosState.ChainId()
		if err != nil {
			return nil, fmt.Errorf("error getting chain ID from initial ArbOS state: %w", err)
		}
		if chainId.Cmp(chainConfig.ChainID) != 0 {
			return nil, fmt.Errorf("unexpected chain ID %r in ArbOS state, expected %r", chainId, chainConfig.ChainID)
		}
		genesisNum, err := initialArbosState.GenesisBlockNum()
		if err != nil {
			return nil, fmt.Errorf("error getting genesis block number from initial ArbOS state: %w", err)
		}
		_, err = initialArbosState.ChainConfig()
		if err != nil {
			return nil, fmt.Errorf("error getting chain config from initial ArbOS state: %w", err)
		}
		expectedNum := chainConfig.ArbitrumChainParams.GenesisBlockNum
		if genesisNum != expectedNum {
			return nil, fmt.Errorf("unexpected genesis block number %v in ArbOS state, expected %v", genesisNum, expectedNum)
		}
	}

	var blockHash common.Hash
	var readBatchInfo []validator.BatchInfo
	if msg != nil {
		batchFetcher := func(batchNum uint64) ([]byte, error) {
			data, err := r.execEngine.consensus.FetchBatch(ctx, batchNum)
			if err != nil {
				return nil, err
			}
			readBatchInfo = append(readBatchInfo, validator.BatchInfo{
				Number: batchNum,
				Data:   data,
			})
			return data, nil
		}
		// Re-fetch the batch instead of using our cached cost,
		// as the replay binary won't have the cache populated.
		msg.Message.BatchGasCost = nil
		block, _, err := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			prevHeader,
			recordingdb,
			chaincontext,
			chainConfig,
			batchFetcher,
		)
		if err != nil {
			return nil, err
		}
		blockHash = block.Hash()
	}

	preimages, err := r.recordingDatabase.PreimagesFromRecording(chaincontext, recordingKV)
	if err != nil {
		return nil, err
	}

	// check we got the canonical hash
	canonicalHash := r.execEngine.bc.GetCanonicalHash(uint64(blockNum))
	if canonicalHash != blockHash {
		return nil, fmt.Errorf("Blockhash doesn't match when recording got %v canonical %v", blockHash, canonicalHash)
	}

	// these won't usually do much here (they will in preparerecording), but doesn't hurt to check
	r.updateLastHdr(prevHeader)
	r.updateValidCandidateHdr(prevHeader)

	return &execution.RecordResult{
		Pos:       pos,
		BlockHash: blockHash,
		Preimages: preimages,
		BatchInfo: readBatchInfo,
	}, err
}

func (r *BlockRecorder) updateLastHdr(hdr *types.Header) {
	if hdr == nil {
		return
	}
	r.lastHdrLock.Lock()
	defer r.lastHdrLock.Unlock()
	if r.lastHdr != nil {
		if hdr.Number.Cmp(r.lastHdr.Number) <= 0 {
			return
		}
	}
	_, err := r.recordingDatabase.StateFor(hdr)
	if err != nil {
		log.Warn("failed to get state in updateLastHdr", "err", err)
		return
	}
	r.recordingDatabase.Dereference(r.lastHdr)
	r.lastHdr = hdr
}

func (r *BlockRecorder) updateValidCandidateHdr(hdr *types.Header) {
	if hdr == nil {
		return
	}
	r.validHdrLock.Lock()
	defer r.validHdrLock.Unlock()
	// don't need a candidate that's newer than the current one (else it will never become valid)
	if r.validHdrCandidate != nil && r.validHdrCandidate.Number.Cmp(hdr.Number) <= 0 {
		return
	}
	// don't need a candidate that's older than known valid
	if r.validHdr != nil && r.validHdr.Number.Cmp(hdr.Number) >= 0 {
		return
	}
	_, err := r.recordingDatabase.StateFor(hdr)
	if err != nil {
		log.Warn("failed to get state in updateLastHdr", "err", err)
		return
	}
	if r.validHdrCandidate != nil {
		r.recordingDatabase.Dereference(r.validHdrCandidate)
	}
	r.validHdrCandidate = hdr
}

func (r *BlockRecorder) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	r.validHdrLock.Lock()
	defer r.validHdrLock.Unlock()
	if r.validHdrCandidate == nil {
		return
	}
	validNum := r.execEngine.MessageIndexToBlockNumber(pos)
	if r.validHdrCandidate.Number.Uint64() > validNum {
		return
	}
	// make sure the valid is canonical
	canonicalResultHash := r.execEngine.bc.GetCanonicalHash(uint64(validNum))
	if canonicalResultHash != resultHash {
		log.Warn("markvalid hash not canonical", "pos", pos, "result", resultHash, "canonical", canonicalResultHash)
		return
	}
	// make sure the candidate is still canonical
	canonicalHash := r.execEngine.bc.GetCanonicalHash(r.validHdrCandidate.Number.Uint64())
	candidateHash := r.validHdrCandidate.Hash()
	if canonicalHash != candidateHash {
		log.Error("vlid candidate hash not canonical", "number", r.validHdrCandidate.Number, "candidate", candidateHash, "canonical", canonicalHash)
		r.recordingDatabase.Dereference(r.validHdrCandidate)
		r.validHdrCandidate = nil
		return
	}
	r.recordingDatabase.Dereference(r.validHdr)
	r.validHdr = r.validHdrCandidate
	r.validHdrCandidate = nil
}

// TODO: use config
func (r *BlockRecorder) preparedAddTrim(newRefs []*types.Header, size int) {
	var oldRefs []*types.Header
	r.preparedLock.Lock()
	r.preparedQueue = append(r.preparedQueue, newRefs...)
	if len(r.preparedQueue) > size {
		oldRefs = r.preparedQueue[:len(r.preparedQueue)-size]
		r.preparedQueue = r.preparedQueue[len(r.preparedQueue)-size:]
	}
	r.preparedLock.Unlock()
	for _, ref := range oldRefs {
		r.recordingDatabase.Dereference(ref)
	}
}

func (r *BlockRecorder) preparedTrimBeyond(hdr *types.Header) {
	var oldRefs []*types.Header
	var validRefs []*types.Header
	r.preparedLock.Lock()
	for _, queHdr := range r.preparedQueue {
		if queHdr.Number.Cmp(hdr.Number) > 0 {
			oldRefs = append(oldRefs, queHdr)
		} else {
			validRefs = append(validRefs, queHdr)
		}
	}
	r.preparedQueue = validRefs
	r.preparedLock.Unlock()
	for _, ref := range oldRefs {
		r.recordingDatabase.Dereference(ref)
	}
}

func (r *BlockRecorder) TrimAllPrepared(t *testing.T) {
	r.preparedAddTrim(nil, 0)
}

func (r *BlockRecorder) RecordingDBReferenceCount() int64 {
	return r.recordingDatabase.ReferenceCount()
}

func (r *BlockRecorder) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	var references []*types.Header
	if end < start {
		return fmt.Errorf("illegal range start %d > end %d", start, end)
	}
	numOfBlocks := uint64(end + 1 - start)
	hdrNum := r.execEngine.MessageIndexToBlockNumber(start)
	if start > 0 {
		hdrNum-- // need to get previous
	} else {
		numOfBlocks-- // genesis block doesn't need preparation, so recording one less block
	}
	lastHdrNum := hdrNum + numOfBlocks
	for hdrNum <= lastHdrNum {
		header := r.execEngine.bc.GetHeaderByNumber(uint64(hdrNum))
		if header == nil {
			log.Warn("prepareblocks asked for non-found block", "hdrNum", hdrNum)
			break
		}
		_, err := r.recordingDatabase.GetOrRecreateState(ctx, header, stateLogFunc)
		if err != nil {
			log.Warn("prepareblocks failed to get state for block", "hdrNum", hdrNum, "err", err)
			break
		}
		references = append(references, header)
		r.updateValidCandidateHdr(header)
		r.updateLastHdr(header)
		hdrNum++
	}
	r.preparedAddTrim(references, 1000)
	return nil
}

func (r *BlockRecorder) ReorgTo(hdr *types.Header) {
	r.validHdrLock.Lock()
	if r.validHdr != nil && r.validHdr.Number.Cmp(hdr.Number) > 0 {
		log.Warn("block recorder: reorging past previously-marked valid block", "reorg target num", hdr.Number, "hash", hdr.Hash(), "reorged past num", r.validHdr.Number, "hash", r.validHdr.Hash())
		r.recordingDatabase.Dereference(r.validHdr)
		r.validHdr = nil
	}
	if r.validHdrCandidate != nil && r.validHdrCandidate.Number.Cmp(hdr.Number) > 0 {
		r.recordingDatabase.Dereference(r.validHdrCandidate)
		r.validHdrCandidate = nil
	}
	r.validHdrLock.Unlock()
	r.lastHdrLock.Lock()
	if r.lastHdr != nil && r.lastHdr.Number.Cmp(hdr.Number) > 0 {
		r.recordingDatabase.Dereference(r.lastHdr)
		r.lastHdr = nil
	}
	r.lastHdrLock.Unlock()
	r.preparedTrimBeyond(hdr)
}

func (r *BlockRecorder) WriteValidStateToDb() error {
	r.validHdrLock.Lock()
	defer r.validHdrLock.Unlock()
	if r.validHdr == nil {
		return nil
	}
	err := r.recordingDatabase.WriteStateToDatabase(r.validHdr)
	r.recordingDatabase.Dereference(r.validHdr)
	return err
}

func (r *BlockRecorder) OrderlyShutdown() {
	err := r.WriteValidStateToDb()
	if err != nil {
		log.Error("failed writing latest valid block state to DB", "err", err)
	}
}

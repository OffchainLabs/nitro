package gethexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

// BlockRecorder uses a separate statedatabase from the blockchain.
// It has access to any state in the ethdb (hard-disk) database, and can compute state as needed.
// We keep references for state of:
// Any block that matches PrepareForRecord that was done recently (according to PrepareDelay config)
// Most recent/advanced header we ever computed (lastHdr)
// Hopefully - some recent valid block. For that we always keep one candidate block until it becomes validated.
type BlockRecorder struct {
	config *BlockRecorderConfig

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

type BlockRecorderConfig struct {
	TrieDirtyCache int `koanf:"trie-dirty-cache"`
	TrieCleanCache int `koanf:"trie-clean-cache"`
	MaxPrepared    int `koanf:"max-prepared"`
}

var DefaultBlockRecorderConfig = BlockRecorderConfig{
	TrieDirtyCache: 1024,
	TrieCleanCache: 16,
	MaxPrepared:    1000,
}

func BlockRecorderConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".trie-dirty-cache", DefaultBlockRecorderConfig.TrieDirtyCache, "like trie-dirty-cache for the separate, recording database (used for validation)")
	f.Int(prefix+".trie-clean-cache", DefaultBlockRecorderConfig.TrieCleanCache, "like trie-clean-cache for the separate, recording database (used for validation)")
	f.Int(prefix+".max-prepared", DefaultBlockRecorderConfig.MaxPrepared, "max references to store in the recording database")
}

func NewBlockRecorder(config *BlockRecorderConfig, execEngine *ExecutionEngine, ethDb ethdb.Database) *BlockRecorder {
	dbConfig := arbitrum.RecordingDatabaseConfig{
		TrieDirtyCache: config.TrieDirtyCache,
		TrieCleanCache: config.TrieCleanCache,
	}
	recorder := &BlockRecorder{
		config:            config,
		execEngine:        execEngine,
		recordingDatabase: arbitrum.NewRecordingDatabase(&dbConfig, ethDb, execEngine.bc),
	}
	execEngine.SetRecorder(recorder)
	return recorder
}

func stateLogFunc(targetHeader *types.Header) arbitrum.StateBuildingLogFunction {
	return func(header *types.Header, hasState bool) {
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
}

// RecordBlocks records the execution of a sequence of messages into blocks.
// This method processes messages sequentially, producing blocks and collecting
// preimage data for validation purposes. It maintains state consistency and
// validates chain configuration throughout the recording process.
//
// The method performs the following steps:
// 1. Validates input parameters and retrieves the starting block header
// 2. Prepares the recording database with the initial state
// 3. Processes each message to produce blocks
// 4. Validates chain configuration and state consistency
// 5. Collects preimage data for verification
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - pos: The starting message index (0-based)
//   - msgs: Array of messages to process (first message at index pos+1)
//
// Returns a RecordResult containing the final position, block hash, and preimage data.
func (r *BlockRecorder) RecordBlocks(
	ctx context.Context,
	// pos is the start message index, i.e. the start block number
	pos arbutil.MessageIndex,
	// the first msg is at index pos+1 and used to derive block pos+1
	// the last msg is at index pos+len(msgs) and used to derive block pos+len(msgs)
	msgs []*arbostypes.MessageWithMetadata,
) (*execution.RecordResult, error) {
	// Validate input parameters
	if len(msgs) == 0 {
		return nil, errors.New("cannot record blocks: message array is empty")
	}

	// Get the block number corresponding to the starting message index
	blockNum := r.execEngine.MessageIndexToBlockNumber(pos)
	header := r.execEngine.bc.GetHeaderByNumber(uint64(blockNum))
	if header == nil {
		return nil, fmt.Errorf("header not found for message index %d (block %d)", pos, blockNum)
	}

	// Prepare the recording database with the initial state
	recordingdb, chaincontext, recordingKV, err := r.recordingDatabase.PrepareRecording(ctx, header, stateLogFunc(header))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare recording database: %w", err)
	}
	defer r.recordingDatabase.Dereference(header)

	// Process each message to produce blocks
	for i, msg := range msgs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled while processing message %d: %w", i, err)
		}

		// Validate chain configuration
		if err := validateChainConfiguration(recordingdb, chaincontext.Config()); err != nil {
			return nil, fmt.Errorf("chain configuration validation failed: %w", err)
		}

		// Produce block from the message
		block, _, err := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			header,
			recordingdb,
			chaincontext,
			false,
			core.MessageReplayMode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to produce block %d from message: %w", header.Number.Uint64(), err)
		}
		header = block.Header()

		// Commit state changes to the trie database
		result, err := recordingdb.Commit(block.NumberU64(), true, false)
		if err != nil {
			return nil, fmt.Errorf("failed to commit state for block %d: %w", block.NumberU64(), err)
		}

		// Validate that the committed state root matches the block header
		if result != header.Root {
			return nil, fmt.Errorf("state root mismatch for block %d: expected %v, got %v", block.NumberU64(), header.Root, result)
		}

		// Create new state database at the new state root
		recordingdb, err = state.NewRecording(header.Root, recordingdb.Database())
		if err != nil {
			return nil, fmt.Errorf("failed to create new recording state for block %d: %w", block.NumberU64(), err)
		}
	}

	// Validate that the final block hash matches the canonical chain
	canonicalHash := r.execEngine.bc.GetCanonicalHash(blockNum + uint64(len(msgs)))
	if canonicalHash != header.Hash() {
		return nil, fmt.Errorf("block hash mismatch: recorded %v, canonical %v", header.Hash(), canonicalHash)
	}

	// Collect preimage data from the recording
	preimages, err := r.recordingDatabase.PreimagesFromRecording(chaincontext, recordingKV)
	if err != nil {
		return nil, fmt.Errorf("failed to collect preimage data: %w", err)
	}

	return &execution.RecordResult{
		Pos:       pos + arbutil.MessageIndex(len(msgs)),
		BlockHash: header.Hash(),
		Preimages: preimages,
		UserWasms: recordingdb.UserWasms(),
	}, nil
}

// validateChainConfiguration validates that the ArbOS state matches the expected chain configuration.
// This includes checking the chain ID, genesis block number, and chain config parameters.
// This validation is important for ensuring consistency between the recording database and the main chain.
func validateChainConfiguration(recordingdb *state.StateDB, chainConfig *params.ChainConfig) error {
	initialArbosState, err := arbosState.OpenSystemArbosState(recordingdb, nil, true)
	if err != nil {
		return fmt.Errorf("failed to open initial ArbOS state: %w", err)
	}

	// Validate chain ID
	chainId, err := initialArbosState.ChainId()
	if err != nil {
		return fmt.Errorf("failed to get chain ID from ArbOS state: %w", err)
	}
	if chainId.Cmp(chainConfig.ChainID) != 0 {
		return fmt.Errorf("chain ID mismatch: got %v, expected %v", chainId, chainConfig.ChainID)
	}

	// Validate genesis block number
	genesisNum, err := initialArbosState.GenesisBlockNum()
	if err != nil {
		return fmt.Errorf("failed to get genesis block number from ArbOS state: %w", err)
	}
	expectedNum := chainConfig.ArbitrumChainParams.GenesisBlockNum
	if genesisNum != expectedNum {
		return fmt.Errorf("genesis block number mismatch: got %v, expected %v", genesisNum, expectedNum)
	}

	// Validate chain config
	returnedConfigBytes, err := initialArbosState.ChainConfig()
	if err != nil {
		return fmt.Errorf("failed to get chain config from ArbOS state: %w", err)
	}

	// Unmarshal the chain config
	returnedConfig := new(params.ChainConfig)
	if err := json.Unmarshal(returnedConfigBytes, returnedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal chain config: %w", err)
	}

	// Ignore initial arbos version & chain owner - they are only used for the genesis block
	returnedConfig.ArbitrumChainParams.InitialArbOSVersion = chainConfig.ArbitrumChainParams.InitialArbOSVersion
	returnedConfig.ArbitrumChainParams.InitialChainOwner = chainConfig.ArbitrumChainParams.InitialChainOwner

	// Compare the returned config with the expected config
	if !reflect.DeepEqual(returnedConfig, chainConfig) {
		return fmt.Errorf("chain config mismatch: got %+v, expected %+v", returnedConfig, chainConfig)
	}

	return nil
}

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

	recordingdb, chaincontext, recordingKV, err := r.recordingDatabase.PrepareRecording(ctx, prevHeader, stateLogFunc(prevHeader))
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
	if msg != nil {
		block, _, err := arbos.ProduceBlock(
			msg.Message,
			msg.DelayedMessagesRead,
			prevHeader,
			recordingdb,
			chaincontext,
			false,
			core.MessageReplayMode,
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
		UserWasms: recordingdb.UserWasms(),
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
		_, err := r.recordingDatabase.GetOrRecreateState(ctx, header, stateLogFunc(header))
		if err != nil {
			log.Warn("prepareblocks failed to get state for block", "hdrNum", hdrNum, "err", err)
			break
		}
		references = append(references, header)
		r.updateValidCandidateHdr(header)
		r.updateLastHdr(header)
		hdrNum++
	}
	r.preparedAddTrim(references, r.config.MaxPrepared)
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

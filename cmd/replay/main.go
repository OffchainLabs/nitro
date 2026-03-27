// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/anytrust/tree"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/wavmio"
)

func getBlockHeaderByHash(hash common.Hash) *types.Header {
	enc, err := wavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	if err != nil {
		panic(fmt.Errorf("Error resolving preimage: %w", err))
	}
	header := &types.Header{}
	err = rlp.DecodeBytes(enc, &header)
	if err != nil {
		panic(fmt.Errorf("Error parsing resolved block header: %w", err))
	}
	return header
}

func getLastBlockHeader() *types.Header {
	lastBlockHash := wavmio.GetLastBlockHash()
	if lastBlockHash == (common.Hash{}) {
		return nil
	}
	return getBlockHeaderByHash(lastBlockHash)
}

type WavmChainContext struct {
	chainConfig *params.ChainConfig
}

func (c WavmChainContext) CurrentHeader() *types.Header {
	return getLastBlockHeader()
}

func (c WavmChainContext) GetHeaderByNumber(number uint64) *types.Header {
	panic("GetHeaderByNumber should not be called in WavmChainContext")
}

func (c WavmChainContext) GetHeaderByHash(hash common.Hash) *types.Header {
	return getBlockHeaderByHash(hash)
}

func (c WavmChainContext) Config() *params.ChainConfig {
	return c.chainConfig
}

func (c WavmChainContext) Engine() consensus.Engine {
	return arbos.Engine{}
}

func (c WavmChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	header := getBlockHeaderByHash(hash)
	if !header.Number.IsUint64() || header.Number.Uint64() != num {
		panic(fmt.Sprintf("Retrieved wrong block number for header hash %v -- requested %v but got %v", hash, num, header.Number.String()))
	}
	return header
}

type WavmInbox struct{}

func (i WavmInbox) PeekSequencerInbox() ([]byte, common.Hash, error) {
	pos := wavmio.GetInboxPosition()
	res := wavmio.ReadInboxMessage(pos)
	log.Info("PeekSequencerInbox", "pos", pos, "res[:8]", res[:8])
	// Our BlobPreimageReader doesn't need the block hash
	return res, common.Hash{}, nil
}

func (i WavmInbox) GetSequencerInboxPosition() uint64 {
	pos := wavmio.GetInboxPosition()
	log.Info("GetSequencerInboxPosition", "pos", pos)
	return pos
}

func (i WavmInbox) AdvanceSequencerInbox() {
	log.Info("AdvanceSequencerInbox")
	wavmio.AdvanceInboxMessage()
}

func (i WavmInbox) GetPositionWithinMessage() uint64 {
	pos := wavmio.GetPositionWithinMessage()
	log.Info("GetPositionWithinMessage", "pos", pos)
	return pos
}

func (i WavmInbox) SetPositionWithinMessage(pos uint64) {
	log.Info("SetPositionWithinMessage", "pos", pos)
	wavmio.SetPositionWithinMessage(pos)
}

func (i WavmInbox) ReadDelayedInbox(seqNum uint64) (*arbostypes.L1IncomingMessage, error) {
	log.Info("ReadDelayedMsg", "seqNum", seqNum)
	data := wavmio.ReadDelayedInboxMessage(seqNum)
	return arbostypes.ParseIncomingL1Message(bytes.NewReader(data), func(batchNum uint64) ([]byte, error) {
		return wavmio.ReadInboxMessage(batchNum), nil
	})
}

type AnyTrustPreimageReader struct {
}

func (*AnyTrustPreimageReader) String() string {
	return "AnyTrustPreimageReader"
}

func (r *AnyTrustPreimageReader) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	oracle := func(hash common.Hash) ([]byte, error) {
		return wavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	}
	return tree.Content(hash, oracle)
}

func (r *AnyTrustPreimageReader) GetKeysetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	return r.GetByHash(ctx, hash)
}

func (r *AnyTrustPreimageReader) HealthCheck(ctx context.Context) error {
	return nil
}

func (r *AnyTrustPreimageReader) ExpirationPolicy(ctx context.Context) (anytrustutil.ExpirationPolicy, error) {
	return anytrustutil.DiscardImmediately, nil
}

type BlobPreimageReader struct {
}

func (r *BlobPreimageReader) GetBlobs(
	ctx context.Context,
	batchBlockHash common.Hash,
	versionedHashes []common.Hash,
) ([]kzg4844.Blob, error) {
	var blobs []kzg4844.Blob
	for _, h := range versionedHashes {
		var blob kzg4844.Blob
		preimage, err := wavmio.ResolveTypedPreimage(arbutil.EthVersionedHashPreimageType, h)
		if err != nil {
			return nil, err
		}
		if len(preimage) != len(blob) {
			return nil, fmt.Errorf("for blob %v got back preimage of length %v but expected blob length %v", h, len(preimage), len(blob))
		}
		copy(blob[:], preimage)
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

func (r *BlobPreimageReader) Initialize(ctx context.Context) error {
	return nil
}

type DACertificatePreimageReader struct {
}

func (r *DACertificatePreimageReader) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
		if len(sequencerMsg) <= 40 {
			return daprovider.PayloadResult{}, fmt.Errorf("sequencer message too small")
		}
		certificate := sequencerMsg[40:]

		// Hash the entire sequencer message to get the preimage key
		customDAPreimageHash := crypto.Keccak256Hash(certificate)

		// Validate the certificate before trying to read it
		if !wavmio.ValidateCertificate(arbutil.DACertificatePreimageType, customDAPreimageHash) {
			// Preimage is not available - treat as invalid batch
			log.Warn("DACertificate preimage validation failed, treating as invalid batch",
				"batchNum", batchNum,
				"batchBlockHash", batchBlockHash,
				"hash", customDAPreimageHash.Hex())
			return daprovider.PayloadResult{Payload: []byte{}}, nil
		}

		// Read the preimage (which contains the actual batch data)
		payload, err := wavmio.ResolveTypedPreimage(arbutil.DACertificatePreimageType, customDAPreimageHash)
		if err != nil {
			// This should not happen after successful validation
			panic(fmt.Errorf("failed to resolve DACertificate preimage after validation: %w", err))
		}

		log.Info("DACertificate batch recovered",
			"batchNum", batchNum,
			"hash", customDAPreimageHash.Hex(),
			"payloadSize", len(payload))

		return daprovider.PayloadResult{Payload: payload}, nil
	})
}

func (r *DACertificatePreimageReader) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
		// Stub implementation: CollectPreimages is only called by the stateless validator
		// to gather preimages before replay. In replay context, preimages have already been
		// collected and injected into the execution environment.
		return daprovider.PreimagesResult{Preimages: make(daprovider.PreimagesMap)}, nil
	})
}

func (r *DACertificatePreimageReader) RecoverPayloadAndPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadAndPreimagesResult, error) {
		// Stub implementation: RecoverPayloadAndPreimages is only called
		// by the MEL validator to gather preimages before validation
		return daprovider.PayloadAndPreimagesResult{Preimages: make(daprovider.PreimagesMap), Payload: nil}, nil
	})
}

func main() {
	wavmio.OnInit()
	gethhook.RequireHookedGeth()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))

	wavmio.PopulateEcdsaCaches()

	raw := rawdb.NewDatabase(PreimageDb{})
	db := state.NewDatabase(triedb.NewDatabase(raw, nil), nil)

	wavmio.OnReady()
	lastBlockHash := wavmio.GetLastBlockHash()

	var lastBlockHeader *types.Header
	var lastBlockStateRoot common.Hash
	if lastBlockHash != (common.Hash{}) {
		lastBlockHeader = getBlockHeaderByHash(lastBlockHash)
		lastBlockStateRoot = lastBlockHeader.Root
	}

	log.Info("Initial State", "lastBlockHash", lastBlockHash, "lastBlockStateRoot", lastBlockStateRoot)
	statedb, err := state.NewDeterministic(lastBlockStateRoot, db)
	if err != nil {
		panic(fmt.Sprintf("Error opening state db: %v", err.Error()))
	}

	batchFetcher := func(batchNum uint64) ([]byte, error) {
		currentBatch := wavmio.GetInboxPosition()
		if batchNum > currentBatch {
			return nil, fmt.Errorf("invalid batch fetch request %d, max %d", batchNum, currentBatch)
		}
		return wavmio.ReadInboxMessage(batchNum), nil
	}
	readMessage := func(anyTrustEnabled bool, chainConfig *params.ChainConfig) *arbostypes.MessageWithMetadata {
		var delayedMessagesRead uint64
		if lastBlockHeader != nil {
			delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
		}
		var anyTrustReader anytrustutil.Reader
		var anyTrustKeysetFetcher anytrustutil.KeysetFetcher
		if anyTrustEnabled {
			// AnyTrust batch and keysets are all together in the same preimage binary.
			anyTrustReader = &AnyTrustPreimageReader{}
			anyTrustKeysetFetcher = &AnyTrustPreimageReader{}
		}
		backend := WavmInbox{}
		var keysetValidationMode = daprovider.KeysetPanicIfInvalid
		if backend.GetPositionWithinMessage() > 0 {
			keysetValidationMode = daprovider.KeysetDontValidate
		}
		dapReaders := daprovider.NewDAProviderRegistry()
		if anyTrustReader != nil {
			err = dapReaders.SetupAnyTrustReader(anytrustutil.NewReader(anyTrustReader, anyTrustKeysetFetcher, keysetValidationMode), nil)
			if err != nil {
				panic(fmt.Sprintf("Failed to register AnyTrust reader: %v", err))
			}
		}
		err = dapReaders.SetupBlobReader(daprovider.NewReaderForBlobReader(&BlobPreimageReader{}))
		if err != nil {
			panic(fmt.Sprintf("Failed to register blob reader: %v", err))
		}

		err = dapReaders.SetupDACertificateReader(&DACertificatePreimageReader{}, nil)
		if err != nil {
			panic(fmt.Sprintf("Failed to register DA Certificate reader: %v", err))
		}

		inboxMultiplexer := arbstate.NewInboxMultiplexer(backend, delayedMessagesRead, dapReaders, keysetValidationMode, chainConfig)
		ctx := context.Background()
		message, err := inboxMultiplexer.Pop(ctx)
		if err != nil {
			panic(fmt.Sprintf("Error reading from inbox multiplexer: %v", err.Error()))
		}

		err = message.Message.FillInBatchGasFields(batchFetcher)
		if err != nil {
			message.Message = arbostypes.InvalidL1Message
		}
		return message
	}

	var newBlock *types.Block
	if lastBlockStateRoot != (common.Hash{}) {
		// ArbOS has already been initialized.
		// Load the chain config and then produce a block normally.

		initialArbosState, err := arbosState.OpenSystemArbosState(statedb, nil, true)
		if err != nil {
			panic(fmt.Sprintf("Error opening initial ArbOS state: %v", err.Error()))
		}
		chainId, err := initialArbosState.ChainId()
		if err != nil {
			panic(fmt.Sprintf("Error getting chain ID from initial ArbOS state: %v", err.Error()))
		}
		genesisBlockNum, err := initialArbosState.GenesisBlockNum()
		if err != nil {
			panic(fmt.Sprintf("Error getting genesis block number from initial ArbOS state: %v", err.Error()))
		}
		chainConfigJson, err := initialArbosState.ChainConfig()
		if err != nil {
			panic(fmt.Sprintf("Error getting chain config from initial ArbOS state: %v", err.Error()))
		}
		var chainConfig *params.ChainConfig
		if len(chainConfigJson) > 0 {
			chainConfig = &params.ChainConfig{}
			err = json.Unmarshal(chainConfigJson, chainConfig)
			if err != nil {
				panic(fmt.Sprintf("Error parsing chain config: %v", err.Error()))
			}
			if chainConfig.ChainID.Cmp(chainId) != 0 {
				panic(fmt.Sprintf("Error: chain id mismatch, chainID: %v, chainConfig.ChainID: %v", chainId, chainConfig.ChainID))
			}
			if chainConfig.ArbitrumChainParams.GenesisBlockNum != genesisBlockNum {
				panic(fmt.Sprintf("Error: genesis block number mismatch, genesisBlockNum: %v, chainConfig.ArbitrumParams.GenesisBlockNum: %v", genesisBlockNum, chainConfig.ArbitrumChainParams.GenesisBlockNum))
			}
		} else {
			log.Info("Falling back to hardcoded chain config.")
			chainConfig, err = chaininfo.GetChainConfig(chainId, "", genesisBlockNum, []string{}, "")
			if err != nil {
				panic(err)
			}
		}

		message := readMessage(chainConfig.ArbitrumChainParams.DataAvailabilityCommittee, chainConfig)

		chainContext := WavmChainContext{chainConfig: chainConfig}
		newBlock, _, _, err = arbos.ProduceBlock(message.Message, message.DelayedMessagesRead, lastBlockHeader, statedb, chainContext, false, core.NewMessageReplayContext(), false)
		if err != nil {
			panic(err)
		}
	} else {
		// Initialize ArbOS with this init message and create the genesis block.

		// Currently, the only use of `chainConfig` argument is to get a limit on the uncompressed batch size.
		// However, the init message is never compressed, so we can safely pass nil here.
		message := readMessage(false, nil)

		initMessage, err := message.Message.ParseInitMessage()
		if err != nil {
			panic(err)
		}
		chainConfig := initMessage.ChainConfig
		if chainConfig == nil {
			log.Info("No chain config in the init message. Falling back to hardcoded chain config.")
			chainConfig, err = chaininfo.GetChainConfig(initMessage.ChainId, "", 0, []string{}, "")
			if err != nil {
				panic(err)
			}
		}

		_, err = arbosState.InitializeArbosState(statedb, burn.NewSystemBurner(nil, false), chainConfig, nil, initMessage)
		if err != nil {
			panic(fmt.Sprintf("Error initializing ArbOS: %v", err.Error()))
		}

		newBlock = arbosState.MakeGenesisBlock(common.Hash{}, 0, 0, statedb.IntermediateRoot(true), chainConfig)

	}

	newBlockHash := newBlock.Hash()

	log.Info("Final State", "newBlockHash", newBlockHash, "StateRoot", newBlock.Root())

	extraInfo := types.DeserializeHeaderExtraInformation(newBlock.Header())
	if extraInfo.ArbOSFormatVersion == 0 {
		panic(fmt.Sprintf("Error deserializing header extra info: %+v", newBlock.Header()))
	}
	wavmio.SetLastBlockHash(newBlockHash)
	wavmio.SetSendRoot(extraInfo.SendRoot)

	wavmio.OnFinal()
}

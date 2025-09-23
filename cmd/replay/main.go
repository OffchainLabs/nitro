// Copyright 2021-2022, Offchain Labs, Inc.
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
	"github.com/offchainlabs/nitro/daprovider/das/dastree"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/gethhook"
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

type WavmChainContext struct {
	chainConfig *params.ChainConfig
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

type PreimageDASReader struct {
}

func (*PreimageDASReader) String() string {
	return "PreimageDASReader"
}

func (dasReader *PreimageDASReader) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	oracle := func(hash common.Hash) ([]byte, error) {
		return wavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	}
	return dastree.Content(hash, oracle)
}

func (dasReader *PreimageDASReader) GetKeysetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	return dasReader.GetByHash(ctx, hash)
}

func (dasReader *PreimageDASReader) HealthCheck(ctx context.Context) error {
	return nil
}

func (dasReader *PreimageDASReader) ExpirationPolicy(ctx context.Context) (dasutil.ExpirationPolicy, error) {
	return dasutil.DiscardImmediately, nil
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

func main() {
	wavmio.StubInit()
	gethhook.RequireHookedGeth()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))

	wavmio.PopulateEcdsaCaches()

	raw := rawdb.NewDatabase(PreimageDb{})
	db := state.NewDatabase(triedb.NewDatabase(raw, nil), nil)

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
	readMessage := func(dasEnabled bool) *arbostypes.MessageWithMetadata {
		var delayedMessagesRead uint64
		if lastBlockHeader != nil {
			delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
		}
		var dasReader dasutil.DASReader
		var dasKeysetFetcher dasutil.DASKeysetFetcher
		if dasEnabled {
			// DAS batch and keysets are all together in the same preimage binary.
			dasReader = &PreimageDASReader{}
			dasKeysetFetcher = &PreimageDASReader{}
		}
		backend := WavmInbox{}
		var keysetValidationMode = daprovider.KeysetPanicIfInvalid
		if backend.GetPositionWithinMessage() > 0 {
			keysetValidationMode = daprovider.KeysetDontValidate
		}
		var dapReaders []daprovider.Reader
		if dasReader != nil {
			dapReaders = append(dapReaders, dasutil.NewReaderForDAS(dasReader, dasKeysetFetcher))
		}
		dapReaders = append(dapReaders, daprovider.NewReaderForBlobReader(&BlobPreimageReader{}))
		inboxMultiplexer := arbstate.NewInboxMultiplexer(backend, delayedMessagesRead, dapReaders, keysetValidationMode)
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

		message := readMessage(chainConfig.ArbitrumChainParams.DataAvailabilityCommittee)

		chainContext := WavmChainContext{chainConfig: chainConfig}
		newBlock, _, err = arbos.ProduceBlock(message.Message, message.DelayedMessagesRead, lastBlockHeader, statedb, chainContext, false, core.NewMessageReplayContext(), false)
		if err != nil {
			panic(err)
		}
	} else {
		// Initialize ArbOS with this init message and create the genesis block.

		message := readMessage(false)

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

	wavmio.StubFinal()
}

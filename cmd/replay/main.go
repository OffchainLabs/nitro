// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/espressocrypto"
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

type WavmChainContext struct{}

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

func (dasReader *PreimageDASReader) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	oracle := func(hash common.Hash) ([]byte, error) {
		return wavmio.ResolveTypedPreimage(arbutil.Keccak256PreimageType, hash)
	}
	return dastree.Content(hash, oracle)
}

func (dasReader *PreimageDASReader) HealthCheck(ctx context.Context) error {
	return nil
}

func (dasReader *PreimageDASReader) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	return daprovider.DiscardImmediately, nil
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

// To generate:
// key, _ := crypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
// sig, _ := crypto.Sign(make([]byte, 32), key)
// println(hex.EncodeToString(sig))
const sampleSignature = "a0b37f8fba683cc68f6574cd43b39f0343a50008bf6ccea9d13231d9e7e2e1e411edc8d307254296264aebfc3dc76cd8b668373a072fd64665b50000e9fcce5201"

// We call this early to populate the secp256k1 ecc basepoint cache in the cached early machine state.
// That means we don't need to re-compute it for every block.
func populateEcdsaCaches() {
	signature, err := hex.DecodeString(sampleSignature)
	if err != nil {
		log.Warn("failed to decode sample signature to populate ECDSA cache", "err", err)
		return
	}
	_, err = crypto.Ecrecover(make([]byte, 32), signature)
	if err != nil {
		log.Warn("failed to recover signature to populate ECDSA cache", "err", err)
		return
	}
}

func main() {
	wavmio.StubInit()
	gethhook.RequireHookedGeth()

	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelError)
	log.SetDefault(log.NewLogger(glogger))

	populateEcdsaCaches()

	raw := rawdb.NewDatabase(PreimageDb{})
	db := state.NewDatabase(raw)

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

	readMessage := func(dasEnabled bool) *arbostypes.MessageWithMetadata {
		var delayedMessagesRead uint64
		if lastBlockHeader != nil {
			delayedMessagesRead = lastBlockHeader.Nonce.Uint64()
		}
		var dasReader daprovider.DASReader
		if dasEnabled {
			dasReader = &PreimageDASReader{}
		}
		backend := WavmInbox{}
		var keysetValidationMode = daprovider.KeysetPanicIfInvalid
		if backend.GetPositionWithinMessage() > 0 {
			keysetValidationMode = daprovider.KeysetDontValidate
		}
		var dapReaders []daprovider.Reader
		if dasReader != nil {
			dapReaders = append(dapReaders, daprovider.NewReaderForDAS(dasReader))
		}
		dapReaders = append(dapReaders, daprovider.NewReaderForBlobReader(&BlobPreimageReader{}))
		inboxMultiplexer := arbstate.NewInboxMultiplexer(backend, delayedMessagesRead, dapReaders, keysetValidationMode)
		ctx := context.Background()
		message, err := inboxMultiplexer.Pop(ctx)
		if err != nil {
			panic(fmt.Sprintf("Error reading from inbox multiplexer: %v", err.Error()))
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

		chainContext := WavmChainContext{}
		batchFetcher := func(batchNum uint64) ([]byte, error) {
			return wavmio.ReadInboxMessage(batchNum), nil
		}

		validatingAgainstEspresso := arbos.IsEspressoMsg(message.Message) && chainConfig.ArbitrumChainParams.EnableEspresso
		if validatingAgainstEspresso {
			_, jst, err := arbos.ParseEspressoMsg(message.Message)
			if err != nil {
				panic(err)
			}
			if jst == nil {
				panic("batch missing espresso justification")
			}

			hotshotHeader := jst.Header
			height := hotshotHeader.Height
			commitment := espressoTypes.Commitment(wavmio.ReadHotShotCommitment(height))
			validatedHeight := wavmio.GetEspressoHeight()
			if validatedHeight == 0 {
				// Validators can choose their own trusted starting point to start their validation.
				// TODO: Check the starting point is greater than the first valid hotshot block number.
				wavmio.SetEspressoHeight(height)
			} else if validatedHeight+1 == height {
				wavmio.SetEspressoHeight(height)
			} else {
				panic(fmt.Sprintf("invalid hotshot block height: %v, got: %v", height, validatedHeight+1))
			}
			if jst.BlockMerkleJustification == nil {
				panic("block merkle justification missing")
			}
			jsonHeader, err := json.Marshal(hotshotHeader)
			if err != nil {
				panic("unable to serialize header")
			}
			// TODO https://github.com/EspressoSystems/nitro-espresso-integration/issues/116
			// Uncomment when validation is fixed
			// espressocrypto.VerifyNamespace(chainConfig.ChainID.Uint64(), *jst.Proof, *jst.Header.PayloadCommitment, *jst.Header.NsTable, txs)

			espressocrypto.VerifyMerkleProof(jst.BlockMerkleJustification.BlockMerkleProof.Proof, jsonHeader, *jst.BlockMerkleJustification.BlockMerkleComm, commitment)
		}

		newBlock, _, err = arbos.ProduceBlock(message.Message, message.DelayedMessagesRead, lastBlockHeader, statedb, chainContext, chainConfig, batchFetcher, false)
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

		_, err = arbosState.InitializeArbosState(statedb, burn.NewSystemBurner(nil, false), chainConfig, initMessage)
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

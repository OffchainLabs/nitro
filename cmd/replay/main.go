// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/wavmio"
	"github.com/pkg/errors"
)

func getBlockHeaderByHash(hash common.Hash) *types.Header {
	enc, err := wavmio.ResolvePreImage(hash)
	if err != nil {
		panic(errors.Wrap(err, "Error resolving preimage"))
	}
	header := &types.Header{}
	err = rlp.DecodeBytes(enc, &header)
	if err != nil {
		panic(errors.Wrap(err, "Error parsing resolved block header"))
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

func (i WavmInbox) PeekSequencerInbox() ([]byte, error) {
	pos := wavmio.GetInboxPosition()
	res := wavmio.ReadInboxMessage(pos)
	log.Info("PeekSequencerInbox", "pos", pos, "res[:8]", res[:8])
	return res, nil
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
	return dastree.Content(hash, wavmio.ResolvePreImage)
}

func (dasReader *PreimageDASReader) HealthCheck(ctx context.Context) error {
	return nil
}

func (dasReader *PreimageDASReader) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.DiscardImmediately, nil
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

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlError)
	log.Root().SetHandler(glogger)

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
		var dasReader arbstate.DataAvailabilityReader
		if dasEnabled {
			dasReader = &PreimageDASReader{}
		}
		backend := WavmInbox{}
		var keysetValidationMode = arbstate.KeysetPanicIfInvalid
		if backend.GetPositionWithinMessage() > 0 {
			keysetValidationMode = arbstate.KeysetDontValidate
		}
		inboxMultiplexer := arbstate.NewInboxMultiplexer(backend, delayedMessagesRead, dasReader, keysetValidationMode)
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
			panic(fmt.Sprintf("Error getting chain ID from initial ArbOS state: %v", err.Error()))
		}
		chainConfig, err := chaininfo.GetChainConfig(chainId, genesisBlockNum, []string{})
		if err != nil {
			panic(err)
		}

		message := readMessage(chainConfig.ArbitrumChainParams.DataAvailabilityCommittee)

		chainContext := WavmChainContext{}
		batchFetcher := func(batchNum uint64) ([]byte, error) {
			return wavmio.ReadInboxMessage(batchNum), nil
		}
		newBlock, _, err = arbos.ProduceBlock(message.Message, message.DelayedMessagesRead, lastBlockHeader, statedb, chainContext, chainConfig, batchFetcher)
		if err != nil {
			panic(err)
		}

	} else {
		// Initialize ArbOS with this init message and create the genesis block.

		message := readMessage(false)

		chainId, err := message.Message.ParseInitMessage()
		if err != nil {
			panic(err)
		}
		chainConfig, err := chaininfo.GetChainConfig(chainId, 0, []string{})
		if err != nil {
			panic(err)
		}
		_, err = arbosState.InitializeArbosState(statedb, burn.NewSystemBurner(nil, false), chainConfig)
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

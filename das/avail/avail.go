package avail

import (
	"context"
	"fmt"
	"strings"

	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/das/avail/vectorx"
	"github.com/offchainlabs/nitro/das/dastree"
)

// AvailMessageHeaderFlag indicates that this data is a Blob Pointer
// which will be used to retrieve data from Avail
const AvailMessageHeaderFlag byte = 0x0a

func IsAvailMessageHeaderByte(header byte) bool {
	return (AvailMessageHeaderFlag & header) > 0
}

type AvailDA struct {
	enable              bool
	vectorx             vectorx.VectorX
	finalizationTimeout time.Duration
	appID               int
	api                 *gsrpc.SubstrateAPI
	meta                *gsrpc_types.Metadata
	genesisHash         gsrpc_types.Hash
	rv                  *gsrpc_types.RuntimeVersion
	keyringPair         signature.KeyringPair
	key                 gsrpc_types.StorageKey
	bridgeApiBaseURL    string
	bridgeApiTimeout    time.Duration
	vectorXTimeout      time.Duration
}

func NewAvailDA(cfg DAConfig, l1Client arbutil.L1Interface) (*AvailDA, error) {

	Seed := cfg.Seed
	AppID := cfg.AppID

	appID := 0
	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}

	// Creating new substrate api
	api, err := gsrpc.NewSubstrateAPI(cfg.AvailApiURL)
	if err != nil {
		return nil, err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		log.Warn("⚠️ cannot get metadata", "error", err)
		return nil, err
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		log.Warn("⚠️ cannot get block hash", "error", err)
		return nil, err
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		log.Warn("⚠️ cannot get runtime version", "error", err)
		return nil, err
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		log.Warn("⚠️ cannot create LeyPair", "error", err)
		return nil, err
	}

	key, err := gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		log.Warn("⚠️ cannot create storage key", "error", err)
		return nil, err
	}

	// Contract address
	contractAddress := common.HexToAddress(cfg.VectorX)

	// Parse the contract ABI
	abi, err := abi.JSON(strings.NewReader(vectorx.VectorxABI))
	if err != nil {
		log.Warn("⚠️ cannot create abi for vectorX", "error", err)
		return nil, err
	}

	// Connect to L1 node thru web socket
	client, err := ethclient.Dial(cfg.ArbSepoliaRPC)
	if err != nil {
		return nil, err
	}

	// Create a filter query to listen for events
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{abi.Events["HeadUpdate"].ID}},
	}

	return &AvailDA{
		enable:              cfg.Enable,
		vectorx:             vectorx.VectorX{Abi: abi, Client: client, Query: query},
		finalizationTimeout: time.Duration(cfg.Timeout),
		appID:               appID,
		api:                 api,
		meta:                meta,
		genesisHash:         genesisHash,
		rv:                  rv,
		keyringPair:         keyringPair,
		key:                 key,
		bridgeApiBaseURL:    "https://turing-bridge-api.fra.avail.so/",
		bridgeApiTimeout:    time.Duration(1200),
		vectorXTimeout:      time.Duration(10000),
	}, nil
}

func (a *AvailDA) Store(ctx context.Context, message []byte) ([]byte, error) {

	finalizedblockHash, nonce, err := submitData(a, message)
	if err != nil {
		return nil, fmt.Errorf("cannot submit data to avail:%w", err)
	}

	header, err := a.api.RPC.Chain.GetHeader(finalizedblockHash)
	if err != nil {
		return nil, fmt.Errorf("cannot get header for finalized block:%w", err)
	}

	err = a.vectorx.SubscribeForHeaderUpdate(int(header.Number), a.vectorXTimeout)
	if err != nil {
		return nil, fmt.Errorf("cannot get the event for header update on vectorx:%w", err)
	}

	extrinsicIndex, err := GetExtrinsicIndex(a.api, finalizedblockHash, a.keyringPair.Address, nonce)
	if err != nil {
		return nil, err
	}
	log.Info("Finalized extrinsic", "extrinsicIndex", extrinsicIndex)

	merkleProofInput, err := QueryMerkleProofInput(a.bridgeApiBaseURL, finalizedblockHash.Hex(), extrinsicIndex, a.bridgeApiTimeout)
	if err != nil {
		return nil, err
	}

	// Creating BlobPointer to submit over settlement layer
	blobPointer := BlobPointer{BlockHash: finalizedblockHash, Sender: a.keyringPair.Address, Nonce: uint32(nonce.Int64()), DasTreeRootHash: dastree.Hash(message), MerkleProofInput: merkleProofInput}
	log.Info("✅  Sucesfully included in block data to Avail", "BlobPointer:", blobPointer)
	blobPointerData, err := blobPointer.MarshalToBinary()
	if err != nil {
		log.Warn("⚠️ BlobPointer MashalBinary error", "err", err)
		return nil, err
	}

	return blobPointerData, nil
}

func (a *AvailDA) Read(ctx context.Context, blobPointer BlobPointer) ([]byte, error) {
	log.Info("ℹ️ Requesting data from Avail", "BlobPointer", blobPointer)

	// Intitializing variables
	BlockHash := blobPointer.BlockHash
	Address := blobPointer.Sender
	Nonce := gsrpc_types.NewUCompactFromUInt(uint64(blobPointer.Nonce))

	// Fetching block based on block hash
	avail_blk, err := a.api.RPC.Chain.GetBlock(BlockHash)
	if err != nil {
		return []byte{}, fmt.Errorf("❌ cannot get block for hash:%v and getting error:%w", BlockHash.Hex(), err)
	}

	// Extracting the required extrinsic according to the reference
	data, err := extractExtrinsic(Address, Nonce.Int64(), avail_blk)
	if err != nil {
		return nil, err
	}

	log.Info("✅  Succesfully fetched data from Avail")
	return data, nil
}

func submitData(a *AvailDA, message []byte) (gsrpc_types.Hash, gsrpc_types.UCompact, error) {
	c, err := gsrpc_types.NewCall(a.meta, "DataAvailability.submit_data", gsrpc_types.NewBytes(message))
	if err != nil {
		log.Warn("⚠️ cannot create new call", "error", err)
		return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, err
	}

	// Create the extrinsic
	ext := gsrpc_types.NewExtrinsic(c)

	var accountInfo gsrpc_types.AccountInfo
	ok, err := a.api.RPC.State.GetStorageLatest(a.key, &accountInfo)
	if err != nil || !ok {
		log.Warn("⚠️ cannot get latest storage", "error", err)
		return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, err
	}

	o := gsrpc_types.SignatureOptions{
		BlockHash:          a.genesisHash,
		Era:                gsrpc_types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        a.genesisHash,
		Nonce:              gsrpc_types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        a.rv.SpecVersion,
		Tip:                gsrpc_types.NewUCompactFromUInt(0),
		AppID:              gsrpc_types.NewUCompactFromUInt(uint64(a.appID)),
		TransactionVersion: a.rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(a.keyringPair, o)
	if err != nil {
		log.Warn("⚠️ cannot sign", "error", err)
		return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, err
	}

	// Send the extrinsic
	sub, err := a.api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		log.Warn("⚠️ cannot submit extrinsic", "error", err)
		return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, err
	}

	log.Info("✅  Tx batch is submitted to Avail", "length", len(message), "address", a.keyringPair.Address, "appID", a.appID)

	defer sub.Unsubscribe()
	timeout := time.After(a.finalizationTimeout * time.Second)
	var finalizedblockHash gsrpc_types.Hash

outer:
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				log.Info("📥  Submit data extrinsic included in block", "blockHash", status.AsInBlock.Hex())
			}
			if status.IsFinalized {
				finalizedblockHash = status.AsFinalized
				break outer
			} else if status.IsDropped {
				return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, fmt.Errorf("❌ Extrinsic dropped")
			} else if status.IsUsurped {
				return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, fmt.Errorf("❌ Extrinsic usurped")
			} else if status.IsRetracted {
				return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, fmt.Errorf("❌ Extrinsic retracted")
			} else if status.IsInvalid {
				return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, fmt.Errorf("❌ Extrinsic invalid")
			}
		case <-timeout:
			return gsrpc_types.Hash{}, gsrpc_types.UCompact{}, fmt.Errorf("⌛️  Timeout of %d seconds reached without getting finalized status for extrinsic", a.finalizationTimeout)
		}
	}

	return finalizedblockHash, o.Nonce, nil
}

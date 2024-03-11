package avail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/vedhavyas/go-subkey"
	"golang.org/x/crypto/sha3"
)

// AvailMessageHeaderFlag indicates that this data is a Blob Pointer
// which will be used to retrieve data from Avail
const AvailMessageHeaderFlag byte = 0x0a

func IsAvailMessageHeaderByte(header byte) bool {
	return (AvailMessageHeaderFlag & header) > 0
}

type BridgdeApiResponse struct {
	BlobRoot           gsrpc_types.Hash   `json:"blobRoot"`
	BlockHash          gsrpc_types.Hash   `json:"blockHash"`
	BridgeRoot         gsrpc_types.Hash   `json:"bridgeRoot"`
	DataRoot           gsrpc_types.Hash   `json:"dataRoot"`
	DataRootCommitment gsrpc_types.Hash   `json:"dataRootCommitment"`
	DataRootIndex      uint64             `json:"dataRootIndex"`
	DataRootProof      []gsrpc_types.Hash `json:"dataRootProof"`
	Leaf               gsrpc_types.Hash   `json:"leaf"`
	LeafIndex          uint64             `json:"leafIndex"`
	LeafProof          []gsrpc_types.Hash `json:"leafProof"`
	RangeHash          gsrpc_types.Hash   `json:"rangeHash"`
}

type AvailDA struct {
	enable      bool
	timeout     time.Duration
	appID       int
	api         *gsrpc.SubstrateAPI
	meta        *gsrpc_types.Metadata
	genesisHash gsrpc_types.Hash
	rv          *gsrpc_types.RuntimeVersion
	keyringPair signature.KeyringPair
	key         gsrpc_types.StorageKey
}

func NewAvailDA(cfg DAConfig) (*AvailDA, error) {

	Seed := cfg.Seed
	AppID := cfg.AppID

	appID := 0
	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}

	// Creating new substrate api
	api, err := gsrpc.NewSubstrateAPI(cfg.ApiURL)
	if err != nil {
		return nil, err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		log.Warn("⚠️ cannot get metadata: error:%v", err)
		return nil, err
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		log.Warn("⚠️ cannot get block hash: error:%v", err)
		return nil, err
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		log.Warn("⚠️ cannot get runtime version: error:%v", err)
		return nil, err
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		log.Warn("⚠️ cannot create LeyPair: error:%v", err)
		return nil, err
	}

	key, err := gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		log.Warn("⚠️ cannot create storage key: error:%v", err)
		return nil, err
	}

	return &AvailDA{
		enable:      cfg.Enable,
		timeout:     cfg.Timeout,
		appID:       appID,
		api:         api,
		meta:        meta,
		genesisHash: genesisHash,
		rv:          rv,
		keyringPair: keyringPair,
		key:         key,
	}, nil
}

func (a *AvailDA) Store(ctx context.Context, message []byte) ([]byte, error) {

	c, err := gsrpc_types.NewCall(a.meta, "DataAvailability.submit_data", gsrpc_types.NewBytes(message))
	if err != nil {
		log.Warn("⚠️ cannot create new call: error:%v", err)
		return nil, err
	}

	// Create the extrinsic
	ext := gsrpc_types.NewExtrinsic(c)

	var accountInfo gsrpc_types.AccountInfo
	ok, err := a.api.RPC.State.GetStorageLatest(a.key, &accountInfo)
	if err != nil || !ok {
		log.Warn("⚠️ cannot get latest storage: error:%v", err)
		return nil, err
	}

	nonce := GetAccountNonce(uint32(accountInfo.Nonce))
	// fmt.Println("Nonce from localDatabase:", nonce, "    ::::::::   from acountInfo:", accountInfo.Nonce)
	o := gsrpc_types.SignatureOptions{
		BlockHash:          a.genesisHash,
		Era:                gsrpc_types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        a.genesisHash,
		Nonce:              gsrpc_types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        a.rv.SpecVersion,
		Tip:                gsrpc_types.NewUCompactFromUInt(0),
		AppID:              gsrpc_types.NewUCompactFromUInt(uint64(a.appID)),
		TransactionVersion: a.rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(a.keyringPair, o)
	if err != nil {
		log.Warn("⚠️ cannot sign: error:%v", err)
		return nil, err
	}

	// Send the extrinsic
	sub, err := a.api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		log.Warn("⚠️ cannot submit extrinsic: error:%v", err)
		return nil, err
	}

	log.Info("✅  Tx batch is submitted to Avail", "length", len(message), "address", a.keyringPair.Address, "appID", a.appID)

	defer sub.Unsubscribe()
	timeout := time.After(time.Duration(a.timeout) * time.Second)
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
				return nil, fmt.Errorf("❌ Extrinsic dropped")
			} else if status.IsUsurped {
				return nil, fmt.Errorf("❌ Extrinsic usurped")
			} else if status.IsRetracted {
				return nil, fmt.Errorf("❌ Extrinsic retracted")
			} else if status.IsInvalid {
				return nil, fmt.Errorf("❌ Extrinsic invalid")
			}
		case <-timeout:
			return nil, fmt.Errorf("⌛️  Timeout of %d seconds reached without getting finalized status for extrinsic", a.timeout)
		}
	}

	// Calculated batch hash for batch commitment
	var batchHash [32]byte
	h := sha3.NewLegacyKeccak256()
	h.Write(message)
	h.Sum(batchHash[:0])

	extrinsicIndex := 1
	// Quering for merkle proof from Bridge Api
	bridgeApiBaseURL := "https://bridge-api.sandbox.avail.tools"
	blockHashPath := "/eth/proof/" + "0xf53613fa06b6b7f9dc5e4cf5f2849affc94e19d8a9e8999207ece01175c988ed" //+ finalizedblockHash.Hex()
	params := url.Values{}
	params.Add("index", fmt.Sprint(extrinsicIndex))

	u, _ := url.ParseRequestURI(bridgeApiBaseURL)
	u.Path = blockHashPath
	u.RawQuery = params.Encode()
	urlStr := fmt.Sprintf("%v", u)

	// TODO: Add time difference between batch submission and querying merkle proof
	resp, err := http.Get(urlStr) //nolint
	if err != nil {
		return nil, fmt.Errorf("bridge Api request not successfull, err=%w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var bridgdeApiResponse BridgdeApiResponse
	err = json.Unmarshal(body, &bridgdeApiResponse)
	if err != nil {
		return nil, err
	}
	var merkleProofInput MerklePoofInput = MerklePoofInput{bridgdeApiResponse.DataRootProof, bridgdeApiResponse.LeafProof, bridgdeApiResponse.RangeHash, bridgdeApiResponse.DataRootIndex, bridgdeApiResponse.BlobRoot, bridgdeApiResponse.BridgeRoot, bridgdeApiResponse.Leaf, bridgdeApiResponse.LeafIndex}

	// Creating BlobPointer to submit over settlement layer
	blobPointer := BlobPointer{BlockHash: finalizedblockHash, Sender: a.keyringPair.Address, Nonce: nonce, DasTreeRootHash: dastree.Hash(message), MerklePoofInput: merkleProofInput}
	log.Info("✅  Sucesfully included in block data to Avail", "BlobPointer:", blobPointer)
	blobPointerData, err := blobPointer.MarshalToBinary()
	if err != nil {
		log.Warn("⚠️ BlobPointer MashalBinary error", "err", err)
		return nil, err
	}

	// buf := new(bytes.Buffer)
	// err = binary.Write(buf, binary.BigEndian, AvailMessageHeaderFlag)
	// if err != nil {
	// 	log.Warn("⚠️ batch type byte serialization failed", "err", err)
	// 	return nil, err
	// }

	// err = binary.Write(buf, binary.BigEndian, blobPointerData)
	// if err != nil {
	// 	log.Warn("⚠️ blob pointer data serialization failed", "err", err)
	// 	return nil, err
	// }

	// serializedBlobPointerData := buf.Bytes()

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
	for _, ext := range avail_blk.Block.Extrinsics {
		// Extracting sender address for extrinsic
		ext_Addr, err := subkey.SS58Address(ext.Signature.Signer.AsID.ToBytes(), 42)
		if err != nil {
			log.Error("❌ unable to get sender address from extrinsic", "err", err)
		}

		if ext_Addr == Address && ext.Signature.Nonce.Int64() == Nonce.Int64() {
			args := ext.Method.Args
			var data []byte
			err = codec.Decode(args, &data)
			if err != nil {
				return []byte{}, fmt.Errorf("❌ unable to decode the extrinsic data by address: %v with nonce: %v", Address, Nonce)
			}
			return data, nil
		}
	}

	log.Info("✅  Succesfully fetched data from Avail")
	return nil, fmt.Errorf("❌ unable to find any extrinsic for this blobPointer:%+v", blobPointer)
}

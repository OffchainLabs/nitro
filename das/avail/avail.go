package avail

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

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

type ProofResponse struct {
	DataProof DataProof `koanf:"dataProof"`
	message   []byte    `koanf:"message"`
}

// HeaderF struct represents response from queryDataProof
type DataProof struct {
	DataRoot       gsrpc_types.Hash   `koanf:"dataRoot"`
	BlobRoot       gsrpc_types.Hash   `koanf:"blobRoot"`
	BridgeRoot     gsrpc_types.Hash   `koanf:"bridgeRoot"`
	Proof          []gsrpc_types.Hash `koanf:"proof"`
	NumberOfLeaves int                `koanf:"numberOfLeaves"`
	LeafIndex      int                `koanf:"leafIndex"`
	Leaf           gsrpc_types.Hash   `koanf:"leaf"`
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

	//Creating new substrate api
	api, err := gsrpc.NewSubstrateAPI(cfg.ApiURL)
	if err != nil {
		return nil, err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		log.Warn("cannot get metadata: error:%v", err)
		return nil, err
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		log.Warn("cannot get block hash: error:%v", err)
		return nil, err
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		log.Warn("cannot get runtime version: error:%v", err)
		return nil, err
	}

	keyringPair, err := signature.KeyringPairFromSecret(Seed, 42)
	if err != nil {
		log.Warn("cannot create LeyPair: error:%v", err)
		return nil, err
	}

	key, err := gsrpc_types.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		log.Warn("cannot create storage key: error:%v", err)
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
		log.Warn("cannot create new call: error:%v", err)
		return nil, err
	}

	// Create the extrinsic
	ext := gsrpc_types.NewExtrinsic(c)

	var accountInfo gsrpc_types.AccountInfo
	ok, err := a.api.RPC.State.GetStorageLatest(a.key, &accountInfo)
	if err != nil || !ok {
		log.Warn("cannot get latest storage: error:%v", err)
		return nil, err
	}

	nonce := GetAccountNonce(uint32(accountInfo.Nonce))
	//fmt.Println("Nonce from localDatabase:", nonce, "    ::::::::   from acountInfo:", accountInfo.Nonce)
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
		log.Warn("cannot sign: error:%v", err)
		return nil, err
	}

	// Send the extrinsic
	sub, err := a.api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		log.Warn("cannot submit extrinsic: error:%v", err)
		return nil, err
	}

	log.Info("✅	Tx batch is submitted to Avail", "length", len(message), "address", a.keyringPair.Address, "appID", a.appID)

	defer sub.Unsubscribe()
	timeout := time.After(time.Duration(a.timeout) * time.Second)
	var finalizedblockHash gsrpc_types.Hash

outer:
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				log.Info("📥	Submit data extrinsic included in block", "blockHash", status.AsInBlock.Hex())
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

	var batchHash [32]byte

	h := sha3.NewLegacyKeccak256()
	h.Write(message)
	h.Sum(batchHash[:0])

	block, err := a.api.RPC.Chain.GetBlock(finalizedblockHash)
	if err != nil {
		log.Warn("cannot get block: error:%v", err)
		return nil, err
	}

	var dataProof DataProof
	for i := 1; i <= len(block.Block.Extrinsics); i++ {
		// query proof
		var data ProofResponse
		err = a.api.Client.Call(&data, "kate_queryDataProofV2", i, finalizedblockHash)
		if err != nil {
			log.Warn("unable to query data proof:%v", err)
			return nil, err
		}

		if data.DataProof.Leaf.Hex() == fmt.Sprintf("%#x", batchHash) {
			dataProof = data.DataProof
			break
		}
	}

	fmt.Printf("Root:%v\n", dataProof.DataRoot.Hex())
	fmt.Printf("Bridge Root:%v\n", dataProof.BridgeRoot.Hex())
	fmt.Printf("Blob Root:%v\n", dataProof.BlobRoot.Hex())

	// print array of proof
	fmt.Printf("Proof:\n")
	for _, p := range dataProof.Proof {
		fmt.Printf("%v\n", p.Hex())
	}

	fmt.Printf("Number of leaves: %v\n", dataProof.NumberOfLeaves)
	fmt.Printf("Leaf index: %v\n", dataProof.LeafIndex)
	fmt.Printf("Leaf: %v\n", dataProof.Leaf.Hex())

	blobPointer := BlobPointer{BlockHash: finalizedblockHash.Hex(), Sender: a.keyringPair.Address, Nonce: o.Nonce.Int64(), DasTreeRootHash: dastree.Hash(message)}

	log.Info("✅	Sucesfully included in block data to Avail", "BlobPointer:", blobPointer)

	blobPointerData, err := blobPointer.MarshalToBinary()
	if err != nil {
		log.Warn("BlobPointer MashalBinary error", "err", err)
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, AvailMessageHeaderFlag)
	if err != nil {
		log.Warn("batch type byte serialization failed", "err", err)
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, blobPointerData)
	if err != nil {
		log.Warn("blob pointer data serialization failed", "err", err)
		return nil, err
	}

	serializedBlobPointerData := buf.Bytes()

	return serializedBlobPointerData, nil

}

func (a *AvailDA) Read(ctx context.Context, blobPointer BlobPointer) ([]byte, error) {
	log.Info("Requesting data from Avail", "BlobPointer", blobPointer)

	//Intitializing variables
	Hash := blobPointer.BlockHash
	Address := blobPointer.Sender
	Nonce := blobPointer.Nonce

	// Converting this string type into gsrpc_types.hash type
	blk_hash, err := gsrpc_types.NewHashFromHexString(Hash)
	if err != nil {
		return nil, fmt.Errorf("unable to convert string hash into types.hash, error:%v", err)
	}

	// Fetching block based on block hash
	avail_blk, err := a.api.RPC.Chain.GetBlock(blk_hash)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot get block for hash:%v and getting error:%v", Hash, err)
	}

	//Extracting the required extrinsic according to the reference
	for _, ext := range avail_blk.Block.Extrinsics {
		//Extracting sender address for extrinsic
		ext_Addr, err := subkey.SS58Address(ext.Signature.Signer.AsID.ToBytes(), 42)
		if err != nil {
			log.Error("unable to get sender address from extrinsic", "err", err)
		}
		if ext_Addr == Address && ext.Signature.Nonce.Int64() == Nonce {
			args := ext.Method.Args
			var data []byte
			err = codec.Decode(args, &data)
			if err != nil {
				return []byte{}, fmt.Errorf("unable to decode the extrinsic data by address: %v with nonce: %v", Address, Nonce)
			}
			return data, nil
		}
	}

	log.Info("Succesfully fetched data from Avail")
	return nil, fmt.Errorf("unable to find any extrinsic for this blobPointer:%+v", blobPointer)
}

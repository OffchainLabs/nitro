package avail

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/vedhavyas/go-subkey"
)

// AvailMessageHeaderFlag indicates that this data is a Blob Pointer
// which will be used to retrieve data from Avail
const AvailMessageHeaderFlag byte = 0x0a

func IsAvailMessageHeaderByte(header byte) bool {
	return (AvailMessageHeaderFlag & header) > 0
}

type AvailDA struct {
	cfg DAConfig
	api *gsrpc.SubstrateAPI
}

func NewAvailDA(cfg DAConfig) (*AvailDA, error) {
	// Creating new substrate api
	api, err := gsrpc.NewSubstrateAPI(cfg.ApiURL)
	if err != nil {
		return nil, err
	}

	return &AvailDA{
		cfg: cfg,
		api: api,
	}, nil
}

func (a *AvailDA) Store(ctx context.Context, message []byte) ([]byte, error) {

	Seed := a.cfg.Seed
	AppID := a.cfg.AppID

	meta, err := a.api.RPC.State.GetMetadataLatest()
	if err != nil {
		log.Warn("cannot get metadata: error:%v", err)
		return nil, err
	}

	appID := 0
	// if app id is greater than 0 then it must be created before submitting data
	if AppID != 0 {
		appID = AppID
	}

	c, err := gsrpc_types.NewCall(meta, "DataAvailability.submit_data", gsrpc_types.NewBytes(message))
	if err != nil {
		log.Warn("cannot create new call: error:%v", err)
		return nil, err
	}

	// Create the extrinsic
	ext := gsrpc_types.NewExtrinsic(c)

	genesisHash, err := a.api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		log.Warn("cannot get block hash: error:%v", err)
		return nil, err
	}

	rv, err := a.api.RPC.State.GetRuntimeVersionLatest()
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

	var accountInfo gsrpc_types.AccountInfo
	ok, err := a.api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		log.Warn("cannot get latest storage: error:%v", err)
		return nil, err
	}

	nonce := GetAccountNonce(uint32(accountInfo.Nonce))
	// fmt.Println("Nonce from localDatabase:", nonce, "    ::::::::   from acountInfo:", accountInfo.Nonce)
	o := gsrpc_types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                gsrpc_types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              gsrpc_types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                gsrpc_types.NewUCompactFromUInt(0),
		AppID:              gsrpc_types.NewUCompactFromUInt(uint64(appID)),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(keyringPair, o)
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

	log.Info("Tx batch is submitted to Avail", "length", len(message), "address", keyringPair.Address, "appID", appID)

	defer sub.Unsubscribe()
	timeout := time.After(100 * time.Second)
	var blobPointer BlobPointer
	for {
		select {
		case status := <-sub.Chan():
			if status.IsFinalized {
				blobPointer = BlobPointer{BlockHash: string(status.AsFinalized.Hex()), Sender: keyringPair.Address, Nonce: o.Nonce.Int64(), DasTreeRootHash: dastree.Hash(message)}
				log.Info("Sucesfully included in block data to Avail", "BlobPointer:", blobPointer)
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
		case <-timeout:
			return nil, errors.New("Timitout before getting finalized status")
		}
	}

	// log.Info("Sucesfully included in block data to Avail", "BlobPointer:", blobPointer)

}

func (a *AvailDA) Read(ctx context.Context, blobPointer BlobPointer) ([]byte, error) {
	log.Info("Requesting data from Avail", "BlobPointer", blobPointer)

	// Intitializing variables
	Hash := blobPointer.BlockHash
	Address := blobPointer.Sender
	Nonce := blobPointer.Nonce

	// Converting this string type into gsrpc_types.hash type
	blk_hash, err := gsrpc_types.NewHashFromHexString(Hash)
	if err != nil {
		return nil, fmt.Errorf("unable to convert string hash into types.hash, error:%w", err)
	}

	// Fetching block based on block hash
	avail_blk, err := a.api.RPC.Chain.GetBlock(blk_hash)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot get block for hash:%v and getting error:%w", Hash, err)
	}

	// Extracting the required extrinsic according to the reference
	for _, ext := range avail_blk.Block.Extrinsics {
		// Extracting sender address for extrinsic
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

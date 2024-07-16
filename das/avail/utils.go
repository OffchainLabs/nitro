package avail

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsrpc_types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/ethereum/go-ethereum/log"
	"github.com/vedhavyas/go-subkey"
)

func GetExtrinsicIndex(api *gsrpc.SubstrateAPI, blockHash gsrpc_types.Hash, address string, nonce gsrpc_types.UCompact) (int, error) {
	// Fetching block based on block hash
	avail_blk, err := api.RPC.Chain.GetBlock(blockHash)
	if err != nil {
		return -1, fmt.Errorf("❌ cannot get block for hash:%v and getting error:%w", blockHash.Hex(), err)
	}

	// Extracting the required extrinsic according to the reference
	for i, ext := range avail_blk.Block.Extrinsics {
		// Extracting sender address for extrinsic
		ext_Addr, err := subkey.SS58Address(ext.Signature.Signer.AsID.ToBytes(), 42)
		if err != nil {
			log.Error("❌ unable to get sender address from extrinsic", "err", err)
		}

		if ext_Addr == address && ext.Signature.Nonce.Int64() == nonce.Int64() {
			return i, nil
		}
	}
	return -1, fmt.Errorf("❌ unable to find any extrinsic in block %v, from address %v with nonce %v", blockHash, address, nonce)
}

func extractExtrinsicData(avail_blk *gsrpc_types.SignedBlock, extrinsicIndex uint32) ([]byte, error) {

	ext := avail_blk.Block.Extrinsics[extrinsicIndex]
	args := ext.Method.Args
	var data []byte
	err := codec.Decode(args, &data)
	if err != nil {
		return []byte{}, fmt.Errorf("❌ unable to decode the extrinsic data for extrinsic: %v", extrinsicIndex)
	}
	return data, nil
}

// ProofResponse struct represents the response from the queryDataProof2 RPC call
type ProofResponse struct {
	DataProof DataProof
	Message   Message // Interface to capture different message types
}

type TxDataRoot struct {
	DataRoot   gsrpc_types.Hash
	BlobRoot   gsrpc_types.Hash
	BridgeRoot gsrpc_types.Hash
}

// DataProof struct represents the data proof response
type DataProof struct {
	Roots          TxDataRoot
	Proof          []gsrpc_types.Hash
	NumberOfLeaves uint32 // Change to uint32 to match Rust u32
	LeafIndex      uint32 // Change to uint32 to match Rust u32
	Leaf           gsrpc_types.Hash
}

// Message interface represents the enum variants
type Message interface {
	isMessage()
}

func QueryBlobProof(api *gsrpc.SubstrateAPI, transactionIndex int, blockHash gsrpc_types.Hash) (BlobProof, error) {
	var res ProofResponse
	err := api.Client.Call(&res, "kate_queryDataProof", transactionIndex, blockHash)
	if err != nil {
		return BlobProof{}, err
	}
	var leafProof [][32]byte
	for _, hash := range res.DataProof.Proof {
		var byte32Array [32]byte
		copy(byte32Array[:], hash[:])
		leafProof = append(leafProof, byte32Array)
	}
	return BlobProof{DataRoot: res.DataProof.Roots.DataRoot, BlobRoot: res.DataProof.Roots.BlobRoot, BridgeRoot: res.DataProof.Roots.BridgeRoot, LeafProof: leafProof, NumberOfLeaves: res.DataProof.NumberOfLeaves, LeafIndex: res.DataProof.LeafIndex, Leaf: res.DataProof.Leaf}, nil
}

type BridgeApiResponse struct {
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

func QueryMerkleProofInput(bridgeApiBaseURL string, blockHash string, extrinsicIndex int, t time.Duration) (MerkleProofInput, error) {
	// Quering for merkle proof from Bridge Api
	blockHashPath := "/eth/proof/" + blockHash
	params := url.Values{}
	params.Add("index", fmt.Sprint(extrinsicIndex))

	u, _ := url.ParseRequestURI(bridgeApiBaseURL)
	u.Path = blockHashPath
	u.RawQuery = params.Encode()
	urlStr := fmt.Sprintf("%v", u)

	bridgeApiResponse, err := queryForBridgeApiRespose(t, urlStr)
	if err != nil {
		return MerkleProofInput{}, fmt.Errorf("failed querying bridgeApiResponse for blockHash:%v and extrinsicIndex:%v, error:%w", blockHash, extrinsicIndex, err)
	}

	merkleProofInput := createMerkleProofInput(bridgeApiResponse)

	return merkleProofInput, nil
}

func queryForBridgeApiRespose(t time.Duration, urlStr string) (BridgeApiResponse, error) {
	var resp *http.Response
	timeout := time.After(t * time.Second)
	for {
		select {
		case <-timeout:
			return BridgeApiResponse{}, fmt.Errorf("⌛️  Timeout of %f min reached without merkleProofInput from bridge-api", t.Minutes())

		default:
			var err error
			resp, err = http.Get(urlStr) //nolint
			if err != nil {
				return BridgeApiResponse{}, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				log.Info("⚠️🥱  MerkleProofInput is not yet available from bridge-api", "status", resp.Status)
				time.Sleep(3 * time.Minute)
				continue
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return BridgeApiResponse{}, err
			}
			var bridgeApiResponse BridgeApiResponse
			err = json.Unmarshal(body, &bridgeApiResponse)
			if err != nil {
				return BridgeApiResponse{}, err
			}
			return bridgeApiResponse, nil
		}
	}
}

func createMerkleProofInput(b BridgeApiResponse) MerkleProofInput {
	var dataRootProof [][32]byte
	for _, hash := range b.DataRootProof {
		var byte32Array [32]byte
		copy(byte32Array[:], hash[:])
		dataRootProof = append(dataRootProof, byte32Array)
	}
	var leafProof [][32]byte
	for _, hash := range b.LeafProof {
		var byte32Array [32]byte
		copy(byte32Array[:], hash[:])
		leafProof = append(leafProof, byte32Array)
	}
	var merkleProofInput MerkleProofInput = MerkleProofInput{dataRootProof, leafProof, b.RangeHash, b.DataRootIndex, b.BlobRoot, b.BridgeRoot, b.Leaf, b.LeafIndex}
	return merkleProofInput
}

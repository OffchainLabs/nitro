// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/prysmaticlabs/prysm/v4/api/client"
)

type BlobClient struct {
	bc *client.Client
	ec *ethclient.Client

	// The genesis time time won't change so only request it once.
	cachedGenesisTime uint64
}

type BlobClientConfig struct {
	BeaconChainUrl string `koanf:"beacon-chain-url"`
}

var DefaultBlobClientConfig = BlobClientConfig{
	BeaconChainUrl: "",
}

func BlobClientAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".url", DefaultBlobClientConfig.BeaconChainUrl, "Beacon Chain url to use for fetching blobs")
}

func NewBlobClient(bc *client.Client, ec *ethclient.Client) *BlobClient {
	return &BlobClient{bc: bc, ec: ec}
}

// Get all the blobs associated with a particular block.
func (b *BlobClient) Get(ctx context.Context, blockHash common.Hash, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	header, err := b.ec.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}

	genesisTime, err := b.genesisTime(ctx)
	if err != nil {
		return nil, err
	}

	// TODO make denominator configurable for devnets with faster block time
	slot := (header.Time - genesisTime) / 12

	return b.blobSidecars(ctx, slot, versionedHashes)
}

type blobResponse struct {
	Data []blobResponseItem `json:"data"`
}
type blobResponseItem struct {
	BlockRoot       string `json:"block_root"`
	Index           int    `json:"index"`
	Slot            uint64 `json:"slot"`
	BlockParentRoot string `json:"block_parent_root"`
	ProposerIndex   uint64 `json:"proposer_index"`
	Blob            string `json:"blob"`
	KzgCommitment   string `json:"kzg_commitment"`
	KzgProof        string `json:"kzg_proof"`
}

func (b *BlobClient) blobSidecars(ctx context.Context, slot uint64, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	body, err := b.bc.Get(ctx, fmt.Sprintf("/eth/v1/beacon/blob_sidecars/%d", slot))
	if err != nil {
		return nil, errors.Wrap(err, "error calling beacon client in blobSidecars")
	}
	br := &blobResponse{}
	err = json.Unmarshal(body, br)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding json response in blobSidecars")
	}

	if len(br.Data) == 0 {
		return nil, fmt.Errorf("no blobs found for slot %d", slot)
	}

	blobs := make([]kzg4844.Blob, len(versionedHashes))
	var totalFound int

	for i := range blobs {
		commitmentBytes, err := hexutil.Decode(br.Data[i].KzgCommitment)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode commitment for slot(%d) at index(%d), commitment(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].KzgCommitment))
		}
		var commitment kzg4844.Commitment
		copy(commitment[:], commitmentBytes)
		versionedHash := vm.KZGToVersionedHash(commitment)

		// The versioned hashes of the blob commitments are produced in the by HASH_OPCODE_BYTE,
		// presumably in the order they were added to the tx. The spec is unclear if the blobs
		// need to be returned in any particular order from the beacon API, so we put them back in
		// the order from the tx.
		var j int
		var found bool
		for j = range versionedHashes {
			if versionedHashes[j] == versionedHash {
				found = true
				totalFound++
				break
			}
		}
		if !found {
			continue
		}

		blob, err := hexutil.Decode(br.Data[i].Blob)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode blob for slot(%d) at index(%d), blob(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].Blob))
		}
		copy(blobs[j][:], blob)

		proofBytes, err := hexutil.Decode(br.Data[i].KzgProof)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode proof for slot(%d) at index(%d), proof(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].KzgProof))
		}
		var proof kzg4844.Proof
		copy(proof[:], proofBytes)

		err = kzg4844.VerifyBlobProof(blobs[j], commitment, proof)
		if err != nil {
			return nil, fmt.Errorf("failed to verify blob proof for blob at slot(%d) at index(%d), blob(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].Blob))
		}
	}

	if totalFound < len(versionedHashes) {
		return nil, fmt.Errorf("not all of the requested blobs (%d/%d) were found at slot (%d), can't reconstruct batch payload", totalFound, len(versionedHashes), slot)
	}

	return blobs, nil
}

type genesisResponse struct {
	GenesisTime uint64 `json:"genesis_time"`
	// don't currently care about other fields, add if needed
}

func (b *BlobClient) genesisTime(ctx context.Context) (uint64, error) {
	if b.cachedGenesisTime > 0 {
		return b.cachedGenesisTime, nil
	}
	body, err := b.bc.Get(ctx, "/eth/v1/beacon/genesis")
	if err != nil {
		return 0, errors.Wrap(err, "error calling beacon client in genesisTime")
	}
	gr := &genesisResponse{}
	dataWrapper := &struct{ Data *genesisResponse }{Data: gr}
	err = json.Unmarshal(body, dataWrapper)
	if err != nil {
		return 0, errors.Wrap(err, "error decoding json response in genesisTime")
	}

	return gr.GenesisTime, nil
}

func RecoverPayloadFromBlob(
	ctx context.Context,
	bc *BlobClient,
	batchBlockHash common.Hash,
	versionedHashBytes []byte,
) ([]byte, error) {
	if bc == nil {
		return nil, errors.New("blob batch payload was encountered but no BlobClient was configured")
	}

	if len(versionedHashBytes)%len(common.Hash{}) != 0 {
		return nil, fmt.Errorf("error in RecoverPayloadFromBlob, remaining batch calldata payload was not only kzg versioned hashes")
	}
	versionedHashes := make([]common.Hash, len(versionedHashBytes)/len(common.Hash{}))

	blobs, err := bc.Get(ctx, batchBlockHash, versionedHashes)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving blobs")
	}

	rlpEncodedPayload := make([]byte, len(blobs)*len(kzg4844.Blob{}))
	for i := range blobs {
		copy(rlpEncodedPayload[i*len(kzg4844.Blob{}):], blobs[i][:])
	}

	var payload []byte
	err = rlp.DecodeBytes(rlpEncodedPayload, &payload)
	if err != nil {
		return nil, errors.Wrap(err, "error rlp decoding recovered blobs into batch payload")
	}

	return payload, nil
}

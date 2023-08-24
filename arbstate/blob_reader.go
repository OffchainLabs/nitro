// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/v4/api/client"
)

type BlobClient struct {
	bc *client.Client
	ec *ethclient.Client

	// The genesis time time won't change so only request it once.
	cachedGenesisTime uint64
}

// Get all the blobs associated with a particular block.
func (b *BlobClient) Get(ctx context.Context, blockHash common.Hash) ([]kzg4844.Blob, error) {
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

	return b.blobSidecars(ctx, slot)
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

func (b *BlobClient) blobSidecars(ctx context.Context, slot uint64) ([]kzg4844.Blob, error) {
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

	blobs := make([]kzg4844.Blob, len(br.Data))
	commitments := make([]kzg4844.Commitment, len(br.Data))
	proofs := make([]kzg4844.Proof, len(br.Data))

	for i := range blobs {
		blob, err := hexutil.Decode(br.Data[i].Blob)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode blob for slot(%d) at index(%d), blob(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].Blob))
		}
		copy(blobs[i][:], blob)

		commitment, err := hexutil.Decode(br.Data[i].KzgCommitment)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode commitment for slot(%d) at index(%d), commitment(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].KzgCommitment))
		}
		copy(commitments[i][:], commitment)

		proof, err := hexutil.Decode(br.Data[i].KzgProof)
		if err != nil {
			return nil, fmt.Errorf("couldn't decode proof for slot(%d) at index(%d), proof(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].KzgProof))
		}
		copy(proofs[i][:], proof)

		err = kzg4844.VerifyBlobProof(blobs[i], commitments[i], proofs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to verify blob proof for blob at slot(%d) at index(%d), blob(%s)", slot, br.Data[i].Index, pretty.FirstFewChars(br.Data[i].Blob))
		}
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
	ec *ethclient.Client,
	batchBlockHash common.Hash,
) ([]byte, error) {

	return nil, nil
}

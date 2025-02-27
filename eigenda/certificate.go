package eigenda

import (
	"fmt"
	"math/big"

	eigenda_common "github.com/Layr-Labs/eigenda/api/grpc/common"
	"github.com/Layr-Labs/eigenda/core"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"

	cv_binding "github.com/Layr-Labs/eigenda/contracts/bindings/EigenDACertVerifier"
	"github.com/ethereum/go-ethereum/common"
)

// EigenDAV1Cert is an internal representation of the encoded cert commitment (i.e, disperser.BlobInfo)
// read from EigenDA proxy. It is used for type compatibility with the Solidity V1 certificate.
// This object is encoded into to txs submitted to the SequencerInbox.
type EigenDAV1Cert struct {
	BlobVerificationProof cv_binding.BlobVerificationProof `json:"blobVerificationProof"`
	BlobHeader            cv_binding.BlobHeader            `json:"blobHeader"`
}

/*
Unlike 4844 there's no need to inject a version byte into the 0th offset of the hash.
Taking the hash of commitment + length is key to ensure no trust assumption on the data length
for one-step proving.
*/
func (e *EigenDAV1Cert) PreimageHash() (*common.Hash, error) {
	bytes, err := e.SerializeCommitment()
	if err != nil {
		return nil, err
	}


	// DataLength is the # of field elements for the blob
	bytes = append(bytes, uint32ToBytes(e.BlobHeader.DataLength)...)
	dataHash := crypto.Keccak256Hash(bytes)


	return &dataHash, nil
}

// SerializeCommitment serializes the kzg commitment points to a byte slice
func (e *EigenDAV1Cert) SerializeCommitment() ([]byte, error) {

	return append(e.BlobHeader.Commitment.X.Bytes(), e.BlobHeader.Commitment.Y.Bytes()...), nil
}

// Load loads the disperser.BlobInfo struct into the EigenDAV1Cert struct
func (b *EigenDAV1Cert) Load(blobInfo *disperser.BlobInfo) {

	x := blobInfo.GetBlobHeader().GetCommitment().GetX()
	y := blobInfo.GetBlobHeader().GetCommitment().GetY()

	b.BlobHeader = cv_binding.BlobHeader{}

	b.BlobHeader.Commitment = cv_binding.BN254G1Point{
		X: new(big.Int).SetBytes(x),
		Y: new(big.Int).SetBytes(y),
	}

	b.BlobHeader.DataLength = blobInfo.GetBlobHeader().GetDataLength()

	for _, quorumBlobParam := range blobInfo.GetBlobHeader().GetBlobQuorumParams() {
		b.BlobHeader.QuorumBlobParams = append(b.BlobHeader.QuorumBlobParams, cv_binding.QuorumBlobParam{
			QuorumNumber:                    uint8(quorumBlobParam.QuorumNumber),
			AdversaryThresholdPercentage:    uint8(quorumBlobParam.AdversaryThresholdPercentage),
			ConfirmationThresholdPercentage: uint8(quorumBlobParam.ConfirmationThresholdPercentage),
			ChunkLength:                     quorumBlobParam.ChunkLength,
		})
	}

	var signatoryRecordHash [32]byte
	copy(signatoryRecordHash[:], blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetSignatoryRecordHash())

	b.BlobVerificationProof.BatchId = blobInfo.GetBlobVerificationProof().GetBatchId()
	b.BlobVerificationProof.BlobIndex = blobInfo.GetBlobVerificationProof().GetBlobIndex()
	b.BlobVerificationProof.BatchMetadata = cv_binding.BatchMetadata{
		BatchHeader:             cv_binding.BatchHeader{},
		SignatoryRecordHash:     signatoryRecordHash,
		ConfirmationBlockNumber: blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetConfirmationBlockNumber(),
	}

	b.BlobVerificationProof.InclusionProof = blobInfo.GetBlobVerificationProof().GetInclusionProof()
	b.BlobVerificationProof.QuorumIndices = blobInfo.GetBlobVerificationProof().GetQuorumIndexes()

	batchRootSlice := blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetBatchRoot()
	var blobHeadersRoot [32]byte
	copy(blobHeadersRoot[:], batchRootSlice)
	b.BlobVerificationProof.BatchMetadata.BatchHeader.BlobHeadersRoot = blobHeadersRoot

	b.BlobVerificationProof.BatchMetadata.BatchHeader.QuorumNumbers = blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetQuorumNumbers()
	b.BlobVerificationProof.BatchMetadata.BatchHeader.SignedStakeForQuorums = blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetQuorumSignedPercentages()
	b.BlobVerificationProof.BatchMetadata.BatchHeader.ReferenceBlockNumber = blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetReferenceBlockNumber()
}
/*
Convert EigenDAV1Cert to DisperserBlobInfo struct for compatibility with proxy server expected type
*/
func (e *EigenDAV1Cert) ToDisperserBlobInfo() (*disperser.BlobInfo, error) {
	// Convert BlobHeader
	var disperserBlobHeader disperser.BlobHeader
	commitment := &eigenda_common.G1Commitment{
		X: e.BlobHeader.Commitment.X.Bytes(),
		Y: e.BlobHeader.Commitment.Y.Bytes(),
	}
	quorumParams := make([]*disperser.BlobQuorumParam, len(e.BlobHeader.QuorumBlobParams))
	for i, qp := range e.BlobHeader.QuorumBlobParams {
		quorumParams[i] = &disperser.BlobQuorumParam{
			QuorumNumber:                    uint32(qp.QuorumNumber),
			AdversaryThresholdPercentage:    uint32(qp.AdversaryThresholdPercentage),
			ConfirmationThresholdPercentage: uint32(qp.ConfirmationThresholdPercentage),
			ChunkLength:                     qp.ChunkLength,
		}
	}
	disperserBlobHeader = disperser.BlobHeader{
		Commitment:       commitment,
		DataLength:       e.BlobHeader.DataLength,
		BlobQuorumParams: quorumParams,
	}

	// Convert BlobVerificationProof
	var disperserBlobVerificationProof disperser.BlobVerificationProof
	var disperserBatchMetadata disperser.BatchMetadata
		metadata := e.BlobVerificationProof.BatchMetadata
		quorumNumbers := metadata.BatchHeader.QuorumNumbers
		quorumSignedPercentages := metadata.BatchHeader.SignedStakeForQuorums

		disperserBatchMetadata = disperser.BatchMetadata{
			BatchHeader: &disperser.BatchHeader{
				BatchRoot:               metadata.BatchHeader.BlobHeadersRoot[:],
				QuorumNumbers:           quorumNumbers,
				QuorumSignedPercentages: quorumSignedPercentages,
				ReferenceBlockNumber:    metadata.BatchHeader.ReferenceBlockNumber,
			},
			BatchHeaderHash:         metadata.SignatoryRecordHash[:],
			Fee:                     []byte{0x00},
			SignatoryRecordHash:     metadata.SignatoryRecordHash[:],
			ConfirmationBlockNumber: metadata.ConfirmationBlockNumber,
		}

		disperserBlobVerificationProof = disperser.BlobVerificationProof{
		BatchId:        e.BlobVerificationProof.BatchId,
		BlobIndex:      e.BlobVerificationProof.BlobIndex,
		BatchMetadata:  &disperserBatchMetadata,
		InclusionProof: e.BlobVerificationProof.InclusionProof,
		QuorumIndexes:  e.BlobVerificationProof.QuorumIndices,
	}

	// set batchHeaderHash - this value is critical for looking the blob against EigenDA disperser.
	// It's lost when translating the BlobInfo --> EigenDAV1Cert and isn't persisted on-chain to
	// reduce calldata sizes.

	bh := disperserBlobVerificationProof.BatchMetadata.BatchHeader

	reducedHeader := core.BatchHeader{
		BatchRoot:            [32]byte(bh.GetBatchRoot()),
		ReferenceBlockNumber: uint(bh.GetReferenceBlockNumber()),
	}

	headerHash, err := reducedHeader.GetBatchHeaderHash()
	if err != nil {
		return nil, fmt.Errorf("generating batch header hash: %w", err)
	}

	disperserBlobVerificationProof.BatchMetadata.BatchHeaderHash = headerHash[:]

	return &disperser.BlobInfo{
		BlobHeader:            &disperserBlobHeader,
		BlobVerificationProof: &disperserBlobVerificationProof,
	}, nil
}

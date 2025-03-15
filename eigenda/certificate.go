package eigenda

import (
	"fmt"
	"math/big"

	eigenda_common "github.com/Layr-Labs/eigenda/api/grpc/common"
	"github.com/Layr-Labs/eigenda/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"

	cv_binding "github.com/Layr-Labs/eigenda/contracts/bindings/EigenDACertVerifier"
	"github.com/ethereum/go-ethereum/common"
)

// EigenDAV1Cert is an internal representation of the encoded cert commitment (i.e, disperser.BlobInfo)
// read from EigenDA proxy. It is used for type compatibility with the Solidity V1 certificate.
// This object is encoded into txs submitted to the SequencerInbox.
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
func (e *EigenDAV1Cert) Load(blobInfo *disperser.BlobInfo) {

	x := blobInfo.GetBlobHeader().GetCommitment().GetX()
	y := blobInfo.GetBlobHeader().GetCommitment().GetY()

	e.BlobHeader = cv_binding.BlobHeader{}

	e.BlobHeader.Commitment = cv_binding.BN254G1Point{
		X: new(big.Int).SetBytes(x),
		Y: new(big.Int).SetBytes(y),
	}

	e.BlobHeader.DataLength = blobInfo.GetBlobHeader().GetDataLength()

	for _, quorumBlobParam := range blobInfo.GetBlobHeader().GetBlobQuorumParams() {
		e.BlobHeader.QuorumBlobParams = append(e.BlobHeader.QuorumBlobParams, cv_binding.QuorumBlobParam{
			QuorumNumber:                    uint8(quorumBlobParam.QuorumNumber),
			AdversaryThresholdPercentage:    uint8(quorumBlobParam.AdversaryThresholdPercentage),
			ConfirmationThresholdPercentage: uint8(quorumBlobParam.ConfirmationThresholdPercentage),
			ChunkLength:                     quorumBlobParam.ChunkLength,
		})
	}

	var signatoryRecordHash [32]byte
	copy(signatoryRecordHash[:], blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetSignatoryRecordHash())

	e.BlobVerificationProof.BatchId = blobInfo.GetBlobVerificationProof().GetBatchId()
	e.BlobVerificationProof.BlobIndex = blobInfo.GetBlobVerificationProof().GetBlobIndex()
	e.BlobVerificationProof.BatchMetadata = cv_binding.BatchMetadata{
		BatchHeader:             cv_binding.BatchHeader{},
		SignatoryRecordHash:     signatoryRecordHash,
		ConfirmationBlockNumber: blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetConfirmationBlockNumber(),
	}

	e.BlobVerificationProof.InclusionProof = blobInfo.GetBlobVerificationProof().GetInclusionProof()
	e.BlobVerificationProof.QuorumIndices = blobInfo.GetBlobVerificationProof().GetQuorumIndexes()

	batchRootSlice := blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetBatchRoot()
	var blobHeadersRoot [32]byte
	copy(blobHeadersRoot[:], batchRootSlice)
	e.BlobVerificationProof.BatchMetadata.BatchHeader.BlobHeadersRoot = blobHeadersRoot

	e.BlobVerificationProof.BatchMetadata.BatchHeader.QuorumNumbers = blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetQuorumNumbers()
	e.BlobVerificationProof.BatchMetadata.BatchHeader.SignedStakeForQuorums = blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetQuorumSignedPercentages()
	e.BlobVerificationProof.BatchMetadata.BatchHeader.ReferenceBlockNumber = blobInfo.GetBlobVerificationProof().GetBatchMetadata().GetBatchHeader().GetReferenceBlockNumber()
}

/*
Convert EigenDAV1Cert to DisperserBlobInfo struct for compatibility with proxy server expected type
*/
func (e *EigenDAV1Cert) ToDisperserBlobInfo() (*disperser.BlobInfo, error) {
	xBytes := e.BlobHeader.Commitment.X.Bytes()
	yBytes := e.BlobHeader.Commitment.Y.Bytes()

	// Remove 0 byte padding (if applicable)
	// Sometimes the big.Int --> bytes transformation would result in a byte array with an
	// extra 0x0 prefixed byte which changes the cert representation returned from /put/
	// on eigenda-proxy since the commitment coordinates returned from the disperser are always
	// 32 bytes each. If the prefixes are kept then secondary storage lookups would fail on the proxy!

	parsedX, err := removeZeroPadding32Bytes(xBytes)
	if err != nil {
		log.Error(`
		failed to remove 0x0 bytes from v1 certificate commitment x field.
		This cert may fail if referenced as lookup key for secondary storage targets on eigenda-proxy.
	`)
		parsedX = xBytes
	}

	parsedY, err := removeZeroPadding32Bytes(yBytes)
	if err != nil {
		log.Error(`
		failed to remove 0x0 bytes from v1 certificate commitment y field.
		This cert may fail if referenced as lookup key for secondary storage targets on eigenda-proxy.
	`)
		parsedY = yBytes
	}

	var disperserBlobHeader disperser.BlobHeader
	commitment := &eigenda_common.G1Commitment{
		X: parsedX,
		Y: parsedY,
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
		BatchHeaderHash: metadata.SignatoryRecordHash[:],
		// assumed to always be 0x00
		// see: https://github.com/Layr-Labs/eigenda/blob/545b7ebc4772e9d85b9863c334abe0512508c0df/disperser/batcher/batcher.go#L319
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

	// set batchHeaderHash - this value is critical for looking up the blob against EigenDA disperser.
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

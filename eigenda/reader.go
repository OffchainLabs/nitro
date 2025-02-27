package eigenda

import (
	"context"
	"encoding/binary"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/arbutil"
)

func NewReaderForEigenDA(reader EigenDAReader) *readerForEigenDA {
	return &readerForEigenDA{readerEigenDA: reader}
}

type readerForEigenDA struct {
	readerEigenDA EigenDAReader
}

func (d *readerForEigenDA) IsValidHeaderByte(headerByte byte) bool {
	return IsEigenDAMessageHeaderByte(headerByte)
}

func (d *readerForEigenDA) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimageRecorder daprovider.PreimageRecorder,
	validateSeqMsg bool,
) ([]byte, error) {
	return RecoverPayloadFromEigenDABatch(ctx, sequencerMsg[sequencerMsgOffset:], d.readerEigenDA, preimageRecorder, "binary")
}

func RecoverPayloadFromEigenDABatch(ctx context.Context,
	sequencerMsg []byte,
	daReader EigenDAReader,
	preimageRecoder daprovider.PreimageRecorder,
	domain string,
) ([]byte, error) {

	eigenDAV1Cert, err := ParseSequencerMsg(sequencerMsg)
	if err != nil {
		log.Error("Failed to parse sequencer message", "err", err)
		return nil, err
	}

	data, err := daReader.QueryBlob(ctx, eigenDAV1Cert, domain)
	if err != nil {
		log.Error("Failed to query data from EigenDA", "err", err)
		return nil, err
	}

	hash, err := eigenDAV1Cert.PreimageHash()
	if err != nil {
		return nil, err
	}

	if preimageRecoder != nil {
		// iFFT the preimage data
		preimage, err := GenericEncodeBlob(data)
		if err != nil {
			return nil, err
		}
		preimageRecoder(*hash, preimage, arbutil.EigenDaPreimageType)
	}
	return data, nil
}

func interfaceToBytesJSON(data interface{}) ([]byte, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// ParseSequencerMsg parses the certificate from the inbox message
func ParseSequencerMsg(abiEncodedCert []byte) (*EigenDAV1Cert, error) {

	spoofedFunc := certDecodeABI.Methods["decodeCert"]

	m := make(map[string]interface{})
	err := spoofedFunc.Inputs.UnpackIntoMap(m, abiEncodedCert)
	if err != nil {
		return nil, err
	}

	b, err := interfaceToBytesJSON(m["cert"])
	if err != nil {
		return nil, err
	}

	// decode to EigenDAV1Cert
	var blobInfo EigenDAV1Cert
	err = json.Unmarshal(b, &blobInfo)

	if err != nil {
		return nil, err
	}

	return &blobInfo, nil

}

func uint32ToBytes(n uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, n)
	return bytes
}

package avail

import (
	"bytes"
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
)

var ErrNoAvailReader = errors.New("Avail batch payload was encountered but no BlobReader was configured")

type AvailDAWriter interface {
	Store(context.Context, []byte) ([]byte, error)
}

type AvailDAReader interface {
	Read(context.Context, BlobPointer) ([]byte, error)
}

type readerForAvailDA struct {
	availDAReader AvailDAReader
}

func NewReaderForAvailDA(availDAReader AvailDAReader) *readerForAvailDA {
	return &readerForAvailDA{availDAReader: availDAReader}
}

func (a *readerForAvailDA) IsValidHeaderByte(headerByte byte) bool {
	return IsAvailMessageHeaderByte(headerByte)
}

func (a *readerForAvailDA) RecoverPayloadFromBatch(ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimageRecorder daprovider.PreimageRecorder,
	validateSeqMsg bool,
) ([]byte, error) {

	buf := bytes.NewBuffer(sequencerMsg[40:])

	header, err := buf.ReadByte()
	if err != nil {
		log.Error("Couldn't deserialize Avail header byte", "err", err)
		return nil, err
	}
	if !IsAvailMessageHeaderByte(header) {
		return nil, errors.New("tried to deserialize a message that doesn't have the Avail header")
	}

	blobPointer := BlobPointer{}
	err = blobPointer.UnmarshalFromBinary(buf.Bytes())
	if err != nil {
		log.Error("Couldn't unmarshal Avail blob pointer", "err", err)
		return nil, err
	}

	log.Info("Attempting to fetch data for", "batchNum", batchNum, "availBlockHash", blobPointer.BlockHash)
	payload, err := a.availDAReader.Read(ctx, blobPointer)
	if err != nil {
		log.Error("Failed to resolve blob pointer from avail", "err", err)
		return nil, err
	}

	log.Info("Succesfully fetched payload from Avail", "batchNum", batchNum, "availBlockHash", blobPointer.BlockHash)

	log.Info("Recording Sha256 preimage for Avail data")

	if preimageRecorder != nil {
		log.Info("Data is being recorded into the orcale", "length", len(payload))
		dastree.RecordHash(preimageRecorder, payload)
	}
	return payload, nil
}

type witerForAvailDA struct {
	availDAWriter AvailDAWriter
}

func NewWriterForAvailDA(availDAWriter AvailDAWriter) *witerForAvailDA {
	return &witerForAvailDA{availDAWriter: availDAWriter}
}

func (a *witerForAvailDA) Store(ctx context.Context,
	message []byte,
	timeout uint64,
	sig []byte,
	disableFallbackStoreDataOnChain bool,
) ([]byte, error) {
	return a.availDAWriter.Store(ctx, message)
}

package melrecording

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

// DAPReader implements recording of preimages when melextraction.ExtractMessages function is called by MEL validator for creation
// of validation entry. Since ExtractMessages function would use daprovider.Reader interface to fetch the sequencer batch via RecoverPayload
// we implement collecting of preimages as well in the same method and record it
type DAPReader struct {
	validatorCtx context.Context
	reader       daprovider.Reader
	preimages    daprovider.PreimagesMap
}

func (r *DAPReader) RecoverPayload(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadResult] {
	promise := r.reader.RecoverPayloadAndPreimages(batchNum, batchBlockHash, sequencerMsg)
	result, err := promise.Await(r.validatorCtx)
	if err != nil {
		return containers.NewReadyPromise(daprovider.PayloadResult{}, err)
	}
	validator.CopyPreimagesInto(r.preimages, result.Preimages)
	return containers.NewReadyPromise(daprovider.PayloadResult{Payload: result.Payload}, nil)
}

func (r *DAPReader) CollectPreimages(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PreimagesResult] {
	return r.reader.CollectPreimages(batchNum, batchBlockHash, sequencerMsg)
}

func (r *DAPReader) RecoverPayloadAndPreimages(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	return r.reader.RecoverPayloadAndPreimages(batchNum, batchBlockHash, sequencerMsg)
}

// DAPReaderSource is used for recording preimages related to sequencer batches stored by da providers, given a
// DapReaderSource it implements GetReader method to return a daprovider.Reader interface that records preimgaes. It takes
// in a context variable (corresponding to creation of validation entry) from the MEL validator
type DAPReaderSource struct {
	validatorCtx context.Context
	dapReaders   arbstate.DapReaderSource
	preimages    daprovider.PreimagesMap
}

func NewDAPReaderSource(validatorCtx context.Context, dapReaders arbstate.DapReaderSource) *DAPReaderSource {
	return &DAPReaderSource{
		validatorCtx: validatorCtx,
		dapReaders:   dapReaders,
		preimages:    make(daprovider.PreimagesMap),
	}
}

func (s *DAPReaderSource) GetReader(headerByte byte) daprovider.Reader {
	reader := s.dapReaders.GetReader(headerByte)
	return &DAPReader{
		validatorCtx: s.validatorCtx,
		reader:       reader,
		preimages:    s.preimages,
	}
}

func (s *DAPReaderSource) Preimages() daprovider.PreimagesMap { return s.preimages }

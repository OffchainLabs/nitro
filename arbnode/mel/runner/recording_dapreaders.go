package melrunner

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/containers"
)

type RecordingDAPReader struct {
	ctx       context.Context
	reader    daprovider.Reader
	preimages daprovider.PreimagesMap
}

func (r *RecordingDAPReader) RecoverPayload(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadResult] {
	promise := r.reader.RecoverPayloadAndPreimages(batchNum, batchBlockHash, sequencerMsg)
	result, err := promise.Await(r.ctx)
	if err != nil {
		return containers.NewReadyPromise(daprovider.PayloadResult{}, err)
	}
	copyPreimagesInto(r.preimages, result.Preimages)
	return containers.NewReadyPromise(daprovider.PayloadResult{Payload: result.Payload}, nil)
}

func (r *RecordingDAPReader) CollectPreimages(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PreimagesResult] {
	return r.reader.CollectPreimages(batchNum, batchBlockHash, sequencerMsg)
}

func (r *RecordingDAPReader) RecoverPayloadAndPreimages(batchNum uint64, batchBlockHash common.Hash, sequencerMsg []byte) containers.PromiseInterface[daprovider.PayloadAndPreimagesResult] {
	return r.reader.RecoverPayloadAndPreimages(batchNum, batchBlockHash, sequencerMsg)
}

type RecordingDAPReaderSource struct {
	ctx        context.Context
	dapReaders arbstate.DapReaderSource
	preimages  daprovider.PreimagesMap
}

func NewRecordingDAPReaderSource(ctx context.Context, dapReaders arbstate.DapReaderSource) *RecordingDAPReaderSource {
	return &RecordingDAPReaderSource{
		ctx:        ctx,
		dapReaders: dapReaders,
		preimages:  make(daprovider.PreimagesMap),
	}
}

func (s *RecordingDAPReaderSource) GetReader(headerByte byte) daprovider.Reader {
	reader := s.dapReaders.GetReader(headerByte)
	return &RecordingDAPReader{
		ctx:       s.ctx,
		reader:    reader,
		preimages: s.preimages,
	}
}

func (s *RecordingDAPReaderSource) Preimages() daprovider.PreimagesMap { return s.preimages }

func copyPreimagesInto(dest, source map[arbutil.PreimageType]map[common.Hash][]byte) {
	for piType, piMap := range source {
		if dest[piType] == nil {
			dest[piType] = make(map[common.Hash][]byte, len(piMap))
		}
		for hash, preimage := range piMap {
			dest[piType][hash] = preimage
		}
	}
}

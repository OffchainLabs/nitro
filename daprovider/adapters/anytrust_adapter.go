package adapters

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
)

// AnyTrustReaderAdapter adapts DAS reader to the unified daprovider.Reader interface
type AnyTrustReaderAdapter struct {
	dasReader     das.DataAvailabilityServiceReader
	keysetFetcher *das.KeysetFetcher
	unifiedReader daprovider.Reader
}

func NewAnyTrustReaderAdapter(dasReader das.DataAvailabilityServiceReader, keysetFetcher *das.KeysetFetcher) *AnyTrustReaderAdapter {
	return &AnyTrustReaderAdapter{
		dasReader:     dasReader,
		keysetFetcher: keysetFetcher,
		unifiedReader: dasutil.NewReaderForDAS(dasReader, keysetFetcher),
	}
}

func (a *AnyTrustReaderAdapter) IsValidHeaderByte(ctx context.Context, headerByte byte) bool {
	return a.unifiedReader.IsValidHeaderByte(ctx, headerByte)
}

func (a *AnyTrustReaderAdapter) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) ([]byte, daprovider.PreimagesMap, error) {
	return a.unifiedReader.RecoverPayloadFromBatch(ctx, batchNum, batchBlockHash, sequencerMsg, preimages, validateSeqMsg)
}

// AnyTrustWriterAdapter adapts DAS writer to the unified daprovider.Writer interface
type AnyTrustWriterAdapter struct {
	dasWriter     das.DataAvailabilityServiceWriter
	unifiedWriter daprovider.Writer
}

func NewAnyTrustWriterAdapter(dasWriter das.DataAvailabilityServiceWriter) *AnyTrustWriterAdapter {
	return &AnyTrustWriterAdapter{
		dasWriter:     dasWriter,
		unifiedWriter: dasutil.NewWriterForDAS(dasWriter),
	}
}

func (a *AnyTrustWriterAdapter) Store(ctx context.Context, message []byte, timeout uint64, disableFallbackStoreDataOnChain bool) ([]byte, error) {
	return a.unifiedWriter.Store(ctx, message, timeout, disableFallbackStoreDataOnChain)
}

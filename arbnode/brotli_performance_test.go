package arbnode

import (
	"bytes"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const (
	BatchSizeLimit     = 20_000_000
	CompressionLevel   = 11
	RecompressionLevel = 11
	NumMessages        = 100
	MessageSize        = 1024 * 128
)

func TestGoLangBrotliPerformance(t *testing.T) {
	bs := createNewBatchSegments(false)
	finalized := fillBatch(t, bs)
	t.Logf("Go brotli compressed size: %d bytes", len(finalized))

	// Verify the output is valid by decompressing it
	verifyDecompression(t, finalized)
}

func TestNativeBrotliPerformance(t *testing.T) {
	bs := createNewBatchSegments(true)
	finalized := fillBatch(t, bs)
	t.Logf("Native brotli compressed size: %d bytes", len(finalized))

	// Verify the output is valid by decompressing it
	verifyDecompression(t, finalized)
}

func verifyDecompression(t *testing.T, compressed []byte) {
	// Skip the brotli header byte
	require.Greater(t, len(compressed), 1, "compressed data too short")

	decompressed, err := arbcompress.Decompress(compressed[1:], arbstate.MaxDecompressedLen)
	require.NoError(t, err, "decompression failed")
	require.Greater(t, len(decompressed), 0, "decompressed data is empty")

	t.Logf("Decompressed size: %d bytes", len(decompressed))
}

func fillBatch(t *testing.T, bs *batchSegments) []byte {
	for i := 0; i < NumMessages; i++ {
		msg := getMessage(i)
		added, err := bs.AddMessage(msg)
		require.NoError(t, err)
		require.True(t, added, "message %d not added to batch", i)
	}
	finalized, err := bs.CloseAndGetBytes()
	require.NoError(t, err)
	return finalized
}

func getMessage(i int) *arbostypes.MessageWithMetadata {
	return &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{},
			L2msg:  testhelpers.RandomSlice(MessageSize),
		},
	}
}

func createNewBatchSegments(useNativeBrotli bool) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, BatchSizeLimit*2))

	var writer brotliWriter
	if useNativeBrotli {
		writer = arbcompress.NewWriterLevel(compressedBuffer, CompressionLevel)
	} else {
		writer = brotli.NewWriterLevel(compressedBuffer, CompressionLevel)
	}

	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   writer,
		rawSegments:        make([][]byte, 0, NumMessages),
		sizeLimit:          BatchSizeLimit,
		compressionLevel:   CompressionLevel,
		recompressionLevel: RecompressionLevel,
		useNativeBrotli:    useNativeBrotli,
	}
}

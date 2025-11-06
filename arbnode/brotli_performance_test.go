package arbnode

import (
	"bytes"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/stretchr/testify/require"
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
	println(len(finalized))
}

func TestNativeBrotliPerformance(t *testing.T) {
	bs := createNewBatchSegments(true)
	finalized := fillBatch(t, bs)
	println(len(finalized))
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

	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   brotli.NewWriterLevel(compressedBuffer, CompressionLevel),
		rawSegments:        make([][]byte, 0, NumMessages),
		sizeLimit:          BatchSizeLimit,
		recompressionLevel: RecompressionLevel,
		useNativeBrotli:    useNativeBrotli,
	}
}

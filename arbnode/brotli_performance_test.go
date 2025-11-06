package arbnode

import (
	"bytes"
	"testing"
	"time"

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

func TestBrotliComparison(t *testing.T) {
	messages := make([]*arbostypes.MessageWithMetadata, NumMessages)
	var totalRawSize int
	for i := 0; i < NumMessages; i++ {
		msg := getMessage()
		messages[i] = msg
		totalRawSize += len(msg.Message.L2msg)
	}
	t.Logf("Setup: Pre-generated %d messages with total raw size: %d bytes (%.2f MB)", NumMessages, totalRawSize, float64(totalRawSize)/(1024*1024))

	runCompressionTest := func(t *testing.T, name string, useNative bool) (time.Duration, []byte) {
		start := time.Now()

		bs := createNewBatchSegments(useNative)

		for i, msg := range messages {
			added, err := bs.AddMessage(msg)
			require.NoError(t, err, "AddMessage failed for %s, message %d", name, i)
			require.True(t, added, "message %d not added to batch for %s", i, name)
		}

		finalized, err := bs.CloseAndGetBytes()
		require.NoError(t, err, "CloseAndGetBytes failed for %s", name)

		duration := time.Since(start)

		t.Logf(
			"| %s | Time: %v | Compressed Size: %d | Ratio: %.2f |",
			name,
			duration,
			len(finalized),
			float64(totalRawSize)/float64(len(finalized)),
		)

		return duration, finalized
	}

	var goLangData []byte
	var nativeData []byte

	t.Run("GoLang_Brotli_Performance", func(t *testing.T) {
		_, goLangData = runCompressionTest(t, "GoLang Brotli", false)
	})

	t.Run("Native_Brotli_Performance", func(t *testing.T) {
		_, nativeData = runCompressionTest(t, "Native Brotli", true)
	})

	require.Equal(t, len(goLangData), len(nativeData), "CRITICAL: Final compressed sizes must be equal, demonstrating identical input and compression algorithm consistency.")

	t.Logf("✅ Consistency Check: Compressed Sizes Match at %d bytes", len(goLangData))
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

func getMessage() *arbostypes.MessageWithMetadata {
	return &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{},
			L2msg:  testhelpers.RandomSlice(MessageSize),
		},
	}
}

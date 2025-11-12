package arbnode

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const BatchSizeLimit = 50_000_000

var configs = []testConfig{
	{
		name:               "100kB/low-then-high",
		compressionLevel:   1,
		recompressionLevel: 11,
		numMessages:        10,
		messageSize:        10_000,
	},
	{
		name:               "100kB/low-then-mid",
		compressionLevel:   1,
		recompressionLevel: 6,
		numMessages:        10,
		messageSize:        10_000,
	},
	{
		name:               "100kB/mid-then-high",
		compressionLevel:   6,
		recompressionLevel: 11,
		numMessages:        10,
		messageSize:        10_000,
	},
	{
		name:               "100kB/high",
		compressionLevel:   11,
		recompressionLevel: 11,
		numMessages:        10,
		messageSize:        10_000,
	},
	{
		name:               "1MB/mid-then-high",
		compressionLevel:   6,
		recompressionLevel: 11,
		numMessages:        10,
		messageSize:        100_000,
	},
}

func TestBrotliCompressionValidity(t *testing.T) {
	msgTypes := []struct {
		typ string
		gen messageGenerator
	}{{"rand", getRandomContent}, {"strct", getStructuredContent}}

	for _, cfg := range configs {
		for _, msgType := range msgTypes {
			cfg.messageGenerator = msgType.gen
			messages, expectedBatch := generateMessages(t, cfg)

			batchVerification := func(t *testing.T, useNativeBrotli bool) {
				compressedBatch := doCompression(t, cfg, messages, useNativeBrotli)
				decompressedBatch, err := arbcompress.Decompress(compressedBatch, BatchSizeLimit)
				require.NoError(t, err)
				require.Equal(t, decompressedBatch, expectedBatch)
			}

			t.Run(fmt.Sprintf("%s/Native", cfg.name), func(b *testing.T) {
				batchVerification(t, true)
			})
			t.Run(fmt.Sprintf("%s/GoLang", cfg.name), func(b *testing.T) {
				batchVerification(t, false)
			})
		}
	}
}

func BenchmarkBrotli(b *testing.B) {
	msgTypes := []struct {
		typ string
		gen messageGenerator
	}{{"rand", getRandomContent}, {"strct", getStructuredContent}}

	allResults := make(map[string]runResult, len(configs)*len(msgTypes))

	for _, cfg := range configs {
		for _, msgType := range msgTypes {
			cfg := cfg
			cfg.name = fmt.Sprintf("%s/%s", cfg.name, msgType.typ)
			cfg.messageGenerator = msgType.gen

			res := &runResult{}

			messages, _ := generateMessages(b, cfg)

			b.Run(fmt.Sprintf("%s/Native", cfg.name), func(b *testing.B) {
				for b.Loop() {
					doCompression(b, cfg, messages, true)
				}
				res.timeNative = b.Elapsed() / time.Duration(b.N)
			})

			b.Run(fmt.Sprintf("%s/GoLang", cfg.name), func(b *testing.B) {
				for b.Loop() {
					doCompression(b, cfg, messages, false)
				}
				res.timeGoLang = b.Elapsed() / time.Duration(b.N)
			})

			allResults[cfg.name] = *res
		}
	}

	b.Logf("------------------------------------------------------------------------------------------------------------------")
	b.Logf("| %-25s | GoLang Time   | Native Time   | Native/Go Ratio |", "Configuration")
	b.Logf("| %-25s |   (per op)    |   (per op)    |  (time per op)  |", "")
	b.Logf("------------------------------------------------------------------------------------------------------------------")

	configNames := make([]string, 0, len(allResults))
	for name := range allResults {
		configNames = append(configNames, name)
	}
	sort.Strings(configNames)

	for _, configName := range configNames {
		res := allResults[configName]
		nativeToGoRatio := float64(res.timeNative) / float64(res.timeGoLang)

		b.Logf("| %-25s | %13v | %13v | %14.2f%% |",
			configName,
			res.timeGoLang,
			res.timeNative,
			nativeToGoRatio*100,
		)
	}
	b.Logf("------------------------------------------------------------------------------------------------------------------")
}

type testConfig struct {
	name               string
	compressionLevel   int
	recompressionLevel int
	numMessages        int
	messageSize        int
	messageGenerator   messageGenerator
}

type messageGenerator func(size int) []byte

type runResult struct {
	timeGoLang time.Duration
	timeNative time.Duration
}

func getRandomContent(size int) []byte {
	return testhelpers.RandomSlice(uint64(size)) // nolint: gosec
}

func getStructuredContent(size int) []byte {
	const baseTxTemplate = `{
        "id": %d,
        "sender": "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA%04d",
        "destination": "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB%04d",
        "timestamp": 1678886400,
        "gasLimit": 50000000,
        "value": "%d",
        "data": "0x%s",
        "nonce": %d
    },`

	const payloadSize = 32
	payloadPart := testhelpers.RandomSlice(payloadSize)
	payloadString := string(payloadPart)

	var data []byte
	for i := 0; len(data) < size; i++ {
		tx := fmt.Sprintf(baseTxTemplate, i, i*2, i+3, i%5, payloadString, i*7)
		data = append(data, []byte(tx)...)
	}
	return data[:size]
}

func generateMessages(t testing.TB, cfg testConfig) ([]*arbostypes.MessageWithMetadata, []byte) {
	messages := make([]*arbostypes.MessageWithMetadata, cfg.numMessages)
	for i := 0; i < cfg.numMessages; i++ {
		messages[i] = &arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{},
				L2msg:  cfg.messageGenerator(cfg.messageSize),
			},
		}
	}

	expectedBatch := make([]byte, 0, BatchSizeLimit)
	for _, msg := range messages {
		segment := make([]byte, 1, len(msg.Message.L2msg)+1)
		segment[0] = arbstate.BatchSegmentKindL2Message
		segment = append(segment, msg.Message.L2msg...)

		encoded, err := rlp.EncodeToBytes(segment)
		require.NoError(t, err)

		expectedBatch = append(expectedBatch, encoded...)
	}

	return messages, expectedBatch
}

func doCompression(t testing.TB, cfg testConfig, messages []*arbostypes.MessageWithMetadata, useNativeBrotli bool) []byte {
	bs := createNewBatchSegments(cfg, useNativeBrotli)
	for _, msg := range messages {
		added, err := bs.AddMessage(msg)
		require.NoError(t, err)
		require.True(t, added)
	}
	compressed, err := bs.CloseAndGetBytes()
	require.NoError(t, err)
	return compressed[1:] // skip header byte marking bytes as brotli-compressed
}

func createNewBatchSegments(cfg testConfig, useNativeBrotli bool) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, 2*BatchSizeLimit))
	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   brotli.NewWriterLevel(compressedBuffer, cfg.compressionLevel),
		rawSegments:        make([][]byte, 0, cfg.numMessages),
		sizeLimit:          BatchSizeLimit,
		compressionLevel:   cfg.compressionLevel,
		recompressionLevel: cfg.recompressionLevel,
		useNativeBrotli:    useNativeBrotli,
	}
}

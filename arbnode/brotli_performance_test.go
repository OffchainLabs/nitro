package arbnode

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/stretchr/testify/require"
)

type messageGenerator func(size int) []byte

type testConfig struct {
	name               string
	compressionLevel   int
	recompressionLevel int
	numMessages        int
	messageSize        int
	messageGenerator   messageGenerator
}

type runResult struct {
	name       string
	timeGoLang time.Duration
	timeNative time.Duration
	ratio      float64
}

func getRandomContent(size int) []byte {
	return testhelpers.RandomSlice(uint64(size))
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

func getMessage(cfg testConfig) *arbostypes.MessageWithMetadata {
	return &arbostypes.MessageWithMetadata{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{},
			L2msg:  cfg.messageGenerator(cfg.messageSize),
		},
	}
}

var configs = []testConfig{
	{
		name:               "Random_Small_Level1",
		compressionLevel:   1,
		recompressionLevel: 1,
		numMessages:        50,
		messageSize:        1024 * 64,
		messageGenerator:   getRandomContent,
	},
	{
		name:               "Random_Large_Level11",
		compressionLevel:   11,
		recompressionLevel: 11,
		numMessages:        100,
		messageSize:        1024 * 128,
		messageGenerator:   getRandomContent,
	},
	{
		name:               "Structured_Small_Level1",
		compressionLevel:   1,
		recompressionLevel: 1,
		numMessages:        50,
		messageSize:        1024 * 64,
		messageGenerator:   getStructuredContent,
	},
	{
		name:               "Structured_Large_Level11",
		compressionLevel:   11,
		recompressionLevel: 11,
		numMessages:        100,
		messageSize:        1024 * 128,
		messageGenerator:   getStructuredContent,
	},
}

const BenchmarkBatchSizeLimit = 50_000_000

func createNewBatchSegments(cfg testConfig, useNativeBrotli bool) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, BenchmarkBatchSizeLimit))
	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   brotli.NewWriterLevel(compressedBuffer, cfg.compressionLevel),
		rawSegments:        make([][]byte, 0, cfg.numMessages),
		sizeLimit:          BenchmarkBatchSizeLimit,
		recompressionLevel: cfg.recompressionLevel,
		useNativeBrotli:    useNativeBrotli,
	}
}

func BenchmarkBrotli(b *testing.B) {
	allResults := make([]runResult, len(configs))

	for cfgIndex, cfg := range configs {
		messages := make([]*arbostypes.MessageWithMetadata, cfg.numMessages)
		for i := 0; i < cfg.numMessages; i++ {
			messages[i] = getMessage(cfg)
		}

		res := &allResults[cfgIndex]
		res.name = cfg.name

		b.Run(fmt.Sprintf("%s/Native", cfg.name), func(b *testing.B) {
			for b.Loop() {
				bs := createNewBatchSegments(cfg, true)
				for _, msg := range messages {
					added, err := bs.AddMessage(msg)
					require.NoError(b, err)
					require.True(b, added)
				}
				_, err := bs.CloseAndGetBytes()
				require.NoError(b, err)
			}
			res.timeNative = b.Elapsed() / time.Duration(b.N)
		})

		b.Run(fmt.Sprintf("%s/GoLang", cfg.name), func(b *testing.B) {
			for b.Loop() {
				bs := createNewBatchSegments(cfg, false)
				for _, msg := range messages {
					added, err := bs.AddMessage(msg)
					require.NoError(b, err)
					require.True(b, added)
				}
				_, err := bs.CloseAndGetBytes()
				require.NoError(b, err)
			}
			res.timeGoLang = b.Elapsed() / time.Duration(b.N)
		})
	}

	b.Logf("------------------------------------------------------------------------------------------------------------------")
	b.Logf("| %-25s | GoLang Time | Native Time | Native/Go Ratio |", "Configuration")
	b.Logf("| %-25s | (per op)    | (per op)    | (Time Native / Time Go) |", "")
	b.Logf("------------------------------------------------------------------------------------------------------------------")

	for _, res := range allResults {
		nativeToGoRatio := float64(res.timeNative) / float64(res.timeGoLang)

		b.Logf("| %-25s | %11v | %11v | %15.2f X |",
			res.name,
			res.timeGoLang,
			res.timeNative,
			nativeToGoRatio,
		)
	}
	b.Logf("------------------------------------------------------------------------------------------------------------------")
}

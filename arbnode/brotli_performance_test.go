package arbnode

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbstate"
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
	timeGoLang time.Duration
	timeNative time.Duration
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

func generateMessages(b *testing.B, cfg testConfig) ([]*arbostypes.MessageWithMetadata, []byte) {
	messages := make([]*arbostypes.MessageWithMetadata, cfg.numMessages)
	for i := 0; i < cfg.numMessages; i++ {
		messages[i] = &arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{},
				L2msg:  cfg.messageGenerator(cfg.messageSize),
			},
		}
	}

	expectedBatch := make([]byte, 0, BenchmarkBatchSizeLimit)
	for _, msg := range messages {
		segment := make([]byte, 1, len(msg.Message.L2msg)+1)
		segment[0] = arbstate.BatchSegmentKindL2Message
		segment = append(segment, msg.Message.L2msg...)

		encoded, err := rlp.EncodeToBytes(segment)
		require.NoError(b, err)

		expectedBatch = append(expectedBatch, encoded...)
	}

	return messages, expectedBatch
}

var configs = []testConfig{
	{
		name:               "small/low",
		compressionLevel:   1,
		recompressionLevel: 1,
		numMessages:        40,
		messageSize:        1024 * 64,
	},
	{
		name:               "large/high",
		compressionLevel:   11,
		recompressionLevel: 11,
		numMessages:        100,
		messageSize:        1024 * 128,
	},
	{
		name:               "large/mid-then-high",
		compressionLevel:   6,
		recompressionLevel: 11,
		numMessages:        100,
		messageSize:        1024 * 128,
	},
	{
		name:               "large/low-then-high",
		compressionLevel:   1,
		recompressionLevel: 11,
		numMessages:        100,
		messageSize:        1024 * 128,
	},
}

const BenchmarkBatchSizeLimit = 50_000_000

func createNewBatchSegments(cfg testConfig, useNativeBrotli bool) *batchSegments {
	compressedBuffer := bytes.NewBuffer(make([]byte, 0, 2*BenchmarkBatchSizeLimit))
	return &batchSegments{
		compressedBuffer:   compressedBuffer,
		compressedWriter:   brotli.NewWriterLevel(compressedBuffer, cfg.compressionLevel),
		rawSegments:        make([][]byte, 0, cfg.numMessages),
		sizeLimit:          BenchmarkBatchSizeLimit,
		recompressionLevel: cfg.recompressionLevel,
		useNativeBrotli:    useNativeBrotli,
	}
}

func benchCompression(b *testing.B, cfg testConfig, messages []*arbostypes.MessageWithMetadata, useNativeBrotli bool) {
	for b.Loop() {
		bs := createNewBatchSegments(cfg, useNativeBrotli)
		for _, msg := range messages {
			added, err := bs.AddMessage(msg)
			require.NoError(b, err)
			require.True(b, added)
		}
		_, err := bs.CloseAndGetBytes()
		require.NoError(b, err)
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
				benchCompression(b, cfg, messages, true)
				res.timeNative = b.Elapsed() / time.Duration(b.N)
			})

			b.Run(fmt.Sprintf("%s/GoLang", cfg.name), func(b *testing.B) {
				benchCompression(b, cfg, messages, false)
				res.timeGoLang = b.Elapsed() / time.Duration(b.N)
			})

			allResults[cfg.name] = *res
		}
	}

	b.Logf("------------------------------------------------------------------------------------------------------------------")
	b.Logf("| %-25s | GoLang Time   | Native Time   | Native/Go Ratio |", "Configuration")
	b.Logf("| %-25s |   (per op)    |   (per op)    |  (time per op)  |", "")
	b.Logf("------------------------------------------------------------------------------------------------------------------")

	for config, res := range allResults {
		nativeToGoRatio := float64(res.timeNative) / float64(res.timeGoLang)

		b.Logf("| %-25s | %13v | %13v | %14.2f%% |",
			config,
			res.timeGoLang,
			res.timeNative,
			nativeToGoRatio*100,
		)
	}
	b.Logf("------------------------------------------------------------------------------------------------------------------")
}

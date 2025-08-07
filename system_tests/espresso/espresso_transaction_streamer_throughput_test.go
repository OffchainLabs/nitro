package espresso_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/system_tests/espresso/chain"
	execution_engine "github.com/offchainlabs/nitro/system_tests/espresso/execution_engine"
	generate "github.com/offchainlabs/nitro/system_tests/espresso/generate"
	transaction_streamer "github.com/offchainlabs/nitro/system_tests/espresso/transaction_streamer"
)

// RunChains sets up the Espresso chain and the TransactionStreamer,
// producing messages at a specified interval. It simulates the sequencer
// sending messages to the TransactionStreamer.
// It also simulates the Espresso chain producing blocks at a specified interval.
func RunChains(ctx context.Context, t *testing.T, mockEspressoChain *chain.MockEspressoChain, streamer *arbnode.TransactionStreamer, behaviors ...generate.GenerationBehavior) {
	messagesInChannel := make(chan generate.Message, 10)
	// Produce Espresso Blocks at a 2 second interval
	go chain.ProduceEspressoBlocksAtInterval(ctx, mockEspressoChain, 2*time.Second)

	// Produce the messages in the channel, so that the TransactionStreamer can
	// process them.
	go generate.ProduceMessages(
		ctx,
		execution_engine.DefaultMessageHasher,
		messagesInChannel,
		behaviors...,
	)

	// Start sending transactions to the TransactionStreamer at a specified
	// interval. This simulates the sequencer sending messages to the
	// TransactionStreamer.
	go (func() {
		_ = generate.SendMessageToWriter(ctx, streamer, messagesInChannel)
	})()
}

// numeric is a type constraint that allows any numeric type, including
// integers and floating-point numbers.
type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// expected is a generic type that holds an expected value and an epsilon
// value for comparison. It provides methods to check if an actual value is
// within the expected range.
type expected[T numeric] struct {
	value   T
	epsilon T
}

// IsWithinEpsilon checks if the actual value is within the range of
// expected value ± epsilon.
func (e expected[T]) IsWithinEpsilon(actual T) bool {
	return actual >= e.value-e.epsilon && actual <= e.value+e.epsilon
}

// IsLessThanOrEqualTo checks if the actual value is less than or equal to
// the expected value plus epsilon.
func (e expected[T]) IsLessThanOrEqualTo(actual T) bool {
	return actual <= e.value+e.epsilon
}

// IsGreaterThanOrEqualTo checks if the actual value is greater than or equal to
// the expected value minus epsilon.
func (e expected[T]) IsGreaterThanOrEqualTo(actual T) bool {
	return actual >= e.value-e.epsilon
}

func (e expected[T]) String() string {
	return fmt.Sprintf("%v (± %v)", e.value, e.epsilon)
}

// scenario defines a test scenario for the TransactionStreamer Espresso
// throughput test.
type scenario struct {
	name string

	// Test Setup
	samples         int
	streamerOptions []transaction_streamer.MockTransactionStreamerEnvironmentOption
	behaviors       []generate.GenerationBehavior

	// Test Acceptance
	duration          expected[time.Duration]
	messageThroughput expected[float64]
	sizeThroughput    expected[float64]
}

// TestEspressoTransactionStreamerToEspressoThroughput is a test that setups
// and times a simplified interaction setup between the TransactionStreamer
// and the Espresso chain.
//
// The purpose of this test is to setup the mock environment, and measure the
// performance throughput of the Environment based on the actual implementation
// of the Espresso communication present within the TransactionStreamer.
func TestEspressoTransactionStreamerToEspressoThroughput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scenarios := []scenario{
		// Molten scenario
		// Generate 500 messages as 4 message/s each sized as 3KiB
		// Idealized Builder
		//
		// Expect the throughput to roughly be 4 messages/s and 12KiB/s
		{
			name:    "MoltenStandardThroughputIdealizedBuilder",
			samples: 500,
			behaviors: []generate.GenerationBehavior{
				generate.GenerateStandardMoltenMessages(500),
			},

			duration: expected[time.Duration]{
				value:   2*time.Minute + 30*time.Second, // 2m30s
				epsilon: 30 * time.Second,               // 30s tolerance
			},
			messageThroughput: expected[float64]{
				value:   4,   // 4 messages/s
				epsilon: 0.1, // 0.1 message/s tolerance
			},
			sizeThroughput: expected[float64]{
				value:   4 * generate.DefaultMoltenMessageSize, // 4 messages/s * 3KiB per message
				epsilon: 512,                                   // 512 bytes tolerance
			},
		},

		// Molten scenario
		// Generate 500 messages as 4 message/s each sized as 3KiB
		// Builder with Max Size of 1MiB
		//
		// Expect the throughput to roughly be 4 messages/s and 12KiB/s
		{
			name:    "MoltenStandardThroughput1MiBBuilder",
			samples: 500,
			streamerOptions: []transaction_streamer.MockTransactionStreamerEnvironmentOption{
				transaction_streamer.AddChainOptions(
					// 1MiB
					chain.WithBuilder(chain.NewMaxSizeRestrictedBuilder(1024 * 1024)),
				),
			},

			behaviors: []generate.GenerationBehavior{
				generate.GenerateStandardMoltenMessages(500),
			},

			duration: expected[time.Duration]{
				value:   6*time.Minute + 30*time.Second, // 6m30s
				epsilon: 30 * time.Second,               // 30s tolerance
			},
			messageThroughput: expected[float64]{
				value:   4,   // 4 messages/s
				epsilon: 0.1, // 0.1 message/s tolerance
			},
			sizeThroughput: expected[float64]{
				value:   4 * generate.DefaultMoltenMessageSize, // 4 messages/s * 3KiB per message
				epsilon: 512,                                   // 512 bytes tolerance
			},
		},

		// Molten Scenario
		// Backlog generate 1_000 messages as quickly as possible each sized as 3KiB
		// Idealized Builder
		//
		// Expect the throughput to allow us to catch up at a significant rate.
		// We expect to be able to process at at least 8x the rate of the
		// standard throughput, so we expect to see 32 messages/s and 96KiB/s
		// throughput.
		//
		// NOTE: current observed throughput depends on hardware.  But on
		// a 2023 Macbook Pro with an M3 Pro we observe a throughput of the
		// following:
		// Took 7.576660458s for 1,000 messages. A Throughput of 131.98 messages/s
		// and 305,442.56 bytes/s.
		{
			name:    "MoltenPreloadBacklogIdealizedBuilder",
			samples: 1_000,
			behaviors: []generate.GenerationBehavior{
				generate.PreloadMoltenMessages(1_000),
			},
			duration: expected[time.Duration]{
				value:   30 * time.Second, // 30s
				epsilon: 1 * time.Second,  // 1s tolerance
			},
			messageThroughput: expected[float64]{
				value:   32,   // 32 messages/s
				epsilon: 0.05, // 0.05 message/s tolerance
			},
			sizeThroughput: expected[float64]{
				value:   32 * generate.DefaultMoltenMessageSize, // 32 messages/s * 3KiB per message
				epsilon: 128,                                    // 128 bytes tolerance
			},
		},

		// Molten Scenario
		// Backlog generate 1_000 messages as quickly as possible each sized as 3KiB
		// Builder with Max Size of 1MiB
		// Expect the throughput to allow us to catch up at a significant rate.
		//
		// We expect to be able to process at at least 8x the rate of the
		// standard throughput, so we expect to see 32 messages/s and 96KiB/s
		// throughput.
		// This is because the Builder is able to handle the backlog of messages
		// without being restricted by the size of the Builder.
		//
		// NOTE: current observed throughput depends on hardware.  But on
		// a 2023 Macbook Pro with an M3 Pro we observe a throughput
		// of the following:
		// Took 4m12.588955167s for 1,000 messages. A throughput of 90.09 messages/s
		// and 276,756.48 bytes/s.
		{
			name:    "MoltenPreloadBacklog1MiBBuilder",
			samples: 1_000,
			streamerOptions: []transaction_streamer.MockTransactionStreamerEnvironmentOption{
				transaction_streamer.AddChainOptions(
					// 1MiB
					chain.WithBuilder(chain.NewMaxSizeRestrictedBuilder(1024 * 1024)),
				),
			},
			behaviors: []generate.GenerationBehavior{
				generate.PreloadMoltenMessages(1_000),
			},
			duration: expected[time.Duration]{
				value:   30 * time.Second, // 30s
				epsilon: 1 * time.Second,  // 1s tolerance
			},
			messageThroughput: expected[float64]{
				value:   32,   // 32 messages/s
				epsilon: 0.05, // 0.05 message/s tolerance
			},
			sizeThroughput: expected[float64]{
				value:   32 * generate.DefaultMoltenMessageSize, // 32 messages/s * 3KiB per message
				epsilon: 128,                                    // 128 bytes tolerance
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup the context for the test
			scenarioCtx, scenarioCancel := context.WithCancel(ctx)
			defer scenarioCancel()

			hasher := execution_engine.DefaultMessageHasher

			// Setup a buffer of GeneratedMessages to be sent to the TransactionStreamer
			// so we can build up a backlog of messages to be processed.
			blocksWithTransactionsCh := make(chan espresso_client.TransactionsInBlock, scenario.samples)

			// Setup the mock environment for the TransactionStreamer
			mockEspressoChain, _, _, streamer, err := transaction_streamer.NewMockTransactionStreamerEnvironment(
				scenarioCtx,
				transaction_streamer.AddEspressoClientOptions(
					func(espressoClient espresso_client.EspressoClient) espresso_client.EspressoClient {
						return chain.NewEspressoClientSimulatedLatency(espressoClient)
					},
					func(espressoClient espresso_client.EspressoClient) espresso_client.EspressoClient {
						return chain.NewSiphonBlocksWithTransactions(espressoClient, blocksWithTransactionsCh)
					},
				),
				transaction_streamer.WithMultipleMockTransactionStreamerEnvironmentOptions(
					scenario.streamerOptions...,
				),
			)

			if have, want := err, error(nil); !errors.Is(have, want) {
				t.Fatalf("encountered error while creating mock transaction streamer environment:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			}

			// Run the scenario with the mock environment
			RunChains(scenarioCtx, t, mockEspressoChain, streamer, scenario.behaviors...)

			start := time.Now()
			// Start the TransactionStreamer, so that processing begins
			if have, want := streamer.Start(scenarioCtx), error(nil); !errors.Is(have, want) {
				t.Fatalf("encountered error while starting TransactionStreamer:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			}

			// Let's grab the messages that are being sent to the TransactionStreamer
			// and are being processed by the Espresso chain.
			receivedMessages := make(map[common.Hash]arbostypes.MessageWithMetadata)

			// Let's consume the transactions from the mock espresso chain, until we get all
			// of the transactions that we sent to the TransactionStreamer.
			for len(receivedMessages) < scenario.samples {
				// Read the next transaction
				blockWithTx := <-blocksWithTransactionsCh

				messages, err := generate.ConvertEspressoTransactionsInBlockToMessages(blockWithTx)
				if have, want := err, error(nil); !errors.Is(have, want) {
					t.Fatalf("encountered error while converting transactions in block to messages:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
				}

				for _, m := range messages {
					receivedMessages[hasher.HashMessageWithMetadata(&m)] = m
				}
			}
			end := time.Now()

			// Stop running the TransactionStreamer
			scenarioCancel()
			streamer.StopWaiter.StopAndWait()

			timingData := chain.Timing(start, end)

			// Compute the total bytes processed
			var totalBytesProcessed uint64
			for _, m := range receivedMessages {
				totalBytesProcessed += uint64(len(m.Message.L2msg))
			}

			if have, want := timingData.Duration, scenario.duration; !want.IsLessThanOrEqualTo(have) {
				t.Errorf("duration is not within expected range:\nhave:\n\t%s\nwant:\n\t<=%s\n",
					have,
					want,
				)
			}

			if have, want := float64(len(receivedMessages))/timingData.Duration.Seconds(), scenario.messageThroughput; !want.IsGreaterThanOrEqualTo(have) {
				t.Errorf("message throughput is not within expected range:\nhave:\n\t%.2f messages/s\nwant:\n\t>= %s messages/s\n",
					have,
					want,
				)
			}

			if have, want := float64(totalBytesProcessed)/timingData.Duration.Seconds(), scenario.sizeThroughput; !want.IsGreaterThanOrEqualTo(have) {
				t.Errorf("size throughput is not within expected range:\nhave:\n\t%.2f bytes/s\nwant:\n\t>= %s bytes/s\n",
					have,
					want,
				)
			}
		})
	}
}

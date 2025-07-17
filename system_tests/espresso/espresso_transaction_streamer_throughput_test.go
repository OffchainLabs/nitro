package espresso_test

import (
	"context"
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

// TestTransactionStreamerEspressoThroughput is a test that setups and times
// a simplified interaction setup between the TransactionStreamer and the
// Espresso chain.
//
// The purpose of this test is to setup the mock environment, and measure the
// performance throughput of the Environment based on the actual implementation
// of the Espresso communication present within the TransactionStreamer.
func TestTransactionStreamerEspressoThroughput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate N Messages, to see how long it takes to process them all.
	const N = 1_000

	hasher := execution_engine.DefaultMessageHasher

	// Setup configurable constants for the test
	const pollingInterval = 1 * time.Second
	const submissionInterval = 5 * time.Second
	const maxTransactionSize = 1_000_000

	runCtx, runCancel := context.WithCancel(ctx)

	// Setup a buffer of GeneratedMessages to be sent to the TransactionStreamer
	// so we can build up a backlog of messages to be processed.
	messagesInChannel := make(chan generate.Message, N)
	blocksWithTransactionsCh := make(chan espresso_client.TransactionsInBlock, N)

	// Setup the Testing Environment
	mockEspressoChain, _, streamer, err := transaction_streamer.NewMockTransactionStreamerEnvironment(
		runCtx,
		transaction_streamer.AddEspressoClientOptions(func(espressoClient espresso_client.EspressoClient) espresso_client.EspressoClient {
			return chain.NewSiphonBlocksWithTransactions(espressoClient, blocksWithTransactionsCh)
		}),
		transaction_streamer.WithEspressoTransactionStreamerOptions(
			arbnode.WithMaxTransactionSize(maxTransactionSize),
			arbnode.WithTxnsPollingInterval(pollingInterval),
		),
	)

	if have, want := err, error(nil); have != want {
		t.Fatalf("encountered error while creating mock transaction streamer environment:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
	}

	// Produce Espresso Blocks at a 2 second interval
	go chain.ProduceEspressoBlocksAtInterval(runCtx, mockEspressoChain, 2*time.Second)

	// Produce the messages in the channel, so that the TransactionStreamer can
	// process them.
	go generate.GenerateNMessages(
		runCtx,
		generate.NewSimpleGenerator(),
		messagesInChannel,
		N,
	)

	// Start sending transactions to the TransactionStreamer at a specified
	// interval. This simulates the sequencer sending messages to the
	// TransactionStreamer.
	go generate.WriteMessagesToSequencerAtInterval(runCtx, streamer, messagesInChannel, 250*time.Millisecond)

	start := time.Now()
	// Start the TransactionStreamer, so that processing begins
	if have, want := streamer.Start(runCtx), error(nil); have != want {
		t.Fatalf("encountered error while starting TransactionStreamer:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
	}

	// Let's grab the messages that are being sent to the TransactionStreamer
	// and are being processed by the Espresso chain.
	receivedMessages := make(map[common.Hash]arbostypes.MessageWithMetadata)

	// Let's consume the transactions from the mock espresso chain, until we get all
	// of the transactions that we sent to the TransactionStreamer.
	for len(receivedMessages) < N {
		// Read the next transaction
		blockWithTx := <-blocksWithTransactionsCh

		messages, err := generate.ConvertEspressoTransactionsInBlockToMessages(blockWithTx)
		if have, want := err, error(nil); have != want {
			t.Fatalf("encountered error while converting transactions in block to messages:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
		}

		for _, m := range messages {
			receivedMessages[hasher.HashMessageWithMetadata(&m)] = m
		}
	}
	end := time.Now()

	// Stop running the TransactionStreamer
	runCancel()
	streamer.StopWaiter.StopAndWait()

	timingData := chain.Timing(start, end)

	fmt.Printf("Polling Interval %s, Submission Interval %s => Took %s, Throughput: %.2f messages/s\n",
		pollingInterval,
		submissionInterval,
		timingData.Duration,
		float64(N)/timingData.Duration.Seconds(),
	)
}

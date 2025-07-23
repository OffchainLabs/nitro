package execution_engine_test

import (
	"context"
	"crypto"
	"fmt"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	execution_engine "github.com/offchainlabs/nitro/system_tests/espresso/execution_engine"
)

// ExampleNewMockExecutionEngineForTransactionStreamer demonstrates how to
// build and utilize the Mock Execution Engine for Transaction Streamer.
//
// This example illustrates the features, and anticipated behavior of the
// Mock Execution Engine for Transaction Streamer.
func ExampleNewMockExecutionEngineForTransactionStreamer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	executionEngine := execution_engine.NewMockExecutionEngineForTransactionStreamer()

	{
		fmt.Print("\n")

		// The Engine starts at index 0 by default.
		promise := executionEngine.HeadMessageIndex()

		result, err := promise.Await(ctx)
		if err != nil {
			panic(fmt.Errorf("expected to get head message index, but got error: %w", err))
		}

		fmt.Printf("head message index: %d\n", result)
	}

	{
		fmt.Print("\n")

		// If you try to ask for a message that hasn't been digested yet, it
		// will return an error, indicating the failure.
		promise := executionEngine.ResultAtMessageIndex(1)

		_, err := promise.Await(ctx)
		if err == nil {
			panic(fmt.Errorf("expected to not find message index 1, but got no error"))
		}

		fmt.Printf("received error asking for message index 1: %s\n", err)
	}

	{
		fmt.Print("\n")

		// The Engine will advance the message index automatically when
		// DigestMessage is invoked.
		promise := executionEngine.DigestMessage(
			1,
			&arbostypes.MessageWithMetadata{
				Message: &arbostypes.L1IncomingMessage{
					Header: &arbostypes.L1IncomingMessageHeader{},
				},
			},
			nil,
		)

		result, err := promise.Await(ctx)
		if err != nil {
			panic(fmt.Errorf("expected to digest message, but got error: %w", err))
		}

		fmt.Printf("block hash: %s\nsend root: %s\n", result.BlockHash.Hex(), result.SendRoot.Hex())
	}

	{
		fmt.Print("\n")

		// The Engine will reflect the new head message index after digesting a
		// message.
		promise := executionEngine.HeadMessageIndex()

		result, err := promise.Await(ctx)
		if err != nil {
			panic(err)
		}

		fmt.Printf("head message index: %d\n", result)
	}

	{
		fmt.Print("\n")

		// The Digested message should be retrievable now by its index.
		promise := executionEngine.ResultAtMessageIndex(1)

		result, err := promise.Await(ctx)
		if err != nil {
			panic(fmt.Errorf("expected to find message index 1, but got error: %w", err))
		}

		fmt.Printf("block hash: %s\nsend root: %s\n", result.BlockHash.Hex(), result.SendRoot.Hex())
	}

	// Output:
	// head message index: 0
	//
	// received error asking for message index 1: no message result found for index 1
	//
	// block hash: 0xbc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a
	// send root: 0x0000000000000000000000000000000000000000000000000000000000000000
	//
	// head message index: 1
	//
	// block hash: 0xbc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a
	// send root: 0x0000000000000000000000000000000000000000000000000000000000000000
}

// ExampleWithHasher demonstrates how to use a custom hasher with the
// NewMockExecutionEngineForTransactionStreamer function.
func ExampleWithHasher() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a mock execution engine with a custom hasher.
	executionEngine := execution_engine.NewMockExecutionEngineForTransactionStreamer(
		execution_engine.WithHasher(execution_engine.NewStdLibHasher(crypto.SHA256)),
	)

	// Digest a message using the custom hasher.
	promise := executionEngine.DigestMessage(
		1,
		&arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{},
			},
		},
		nil,
	)

	result, err := promise.Await(ctx)
	if err != nil {
		panic(fmt.Errorf("expected to digest message, but got error: %w", err))
	}

	fmt.Printf("block hash: %s\nsend root: %s\n", result.BlockHash.Hex(), result.SendRoot.Hex())

	// Output:
	// block hash: 0x6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d
	// send root: 0x0000000000000000000000000000000000000000000000000000000000000000
}

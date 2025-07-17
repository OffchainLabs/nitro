package chain_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	tagged_base64 "github.com/EspressoSystems/espresso-network/sdks/go/tagged-base64"
	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"

	"github.com/offchainlabs/nitro/system_tests/espresso/chain"
)

// ExampleNewMockEspressoChain demonstrates how to create and utilize the
// MockEspressoChain.
func ExampleNewMockEspressoChain() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a new mock Espresso chain
	mockChain := chain.NewMockEspressoChain()

	// The Mock Chain is simplified representation of an Espresso Chain
	// that allows the consumer control over when the chain advances and
	// produces the next block.
	//
	// This approach allows for the flexibility of controlling specific
	// progression patterns that might result in edge cases that can
	// be reproduced.

	{
		fmt.Println()
		// The chain starts at height 0, and will only Advance, when
		// the Advance method is called.

		height, err := mockChain.FetchLatestBlockHeight(ctx)
		if err != nil {
			panic(fmt.Errorf("error fetching latest block height: %w\n", err))
		}

		fmt.Printf("Starting Block Height: %d\n", height)

		// Advance the chain to the next block.
		mockChain.Advance()

		height, err = mockChain.FetchLatestBlockHeight(ctx)
		if err != nil {
			panic(fmt.Errorf("error fetching latest block height after advance: %w\n", err))
		}
		fmt.Printf("After Advance Block Height: %d\n", height)
	}

	{
		fmt.Println()
		// If we submit a transaction, it will not immediately result in a block
		// but instead will be pending until the next Advance call.
		// Once the next Advance call occurs, the pending transactions will
		// be included in the next block.

		txHash, err := mockChain.SubmitTransaction(ctx, espresso_common.Transaction{
			Namespace: 1234,
			Payload:   []byte("example transaction payload"),
		})

		if err != nil {
			panic(fmt.Errorf("error submitting transaction: %w\n", err))
		}

		fmt.Printf("Submitted transaction with hash: %s\n", txHash)

		// If we try to grab the block for the transaction now, it will not be
		// found

		_, err = mockChain.FetchTransactionByHash(ctx, txHash)
		if err == nil {
			// This is an error
			panic(fmt.Errorf("expected error fetching transaction by hash, but received no error"))
		}

		fmt.Printf("Lookup for transaction by hash returned error: %s\n", err)

		// Advance the chain to produce the next block
		mockChain.Advance()

		// Now we can fetch the transaction by hash, and it should be found
		txData, err := mockChain.FetchTransactionByHash(ctx, txHash)
		if err != nil {
			panic(fmt.Errorf("error fetching transaction by hash after advance: %w\n", err))
		}

		fmt.Printf("Fetched transaction by hash: %s\n", txData.Hash)

		// This will also allow us to fetch all of the transactions within the
		// block

		txsInBlock, err := mockChain.FetchTransactionsInBlock(ctx, 1, 1234)
		if err != nil {
			panic(fmt.Errorf("error fetching transactions in block: %w\n", err))
		}

		for i, tx := range txsInBlock.Transactions {
			fmt.Printf("Transaction in block [%d]: \"%s\"\n", i, tx)
		}

		// As an alternative, we can also fetch the explorer block information.
		explorerBlock, err := mockChain.FetchExplorerTransactionByHash(ctx, txHash)
		if err != nil {
			panic(fmt.Errorf("error fetching explorer transaction by hash: %w\n", err))
		}

		fmt.Printf("Explorer Block Height: %d\n", explorerBlock.TransactionsDetails.ExplorerDetails.BlockHeight)
		fmt.Printf("Explorer Block Hash: %s\n", explorerBlock.TransactionsDetails.ExplorerDetails.Hash.String())
	}

	// Output:
	// Starting Block Height: 0
	// After Advance Block Height: 1
	//
	// Submitted transaction with hash: MOCK-TXN~Q8U3UD4JrPhZfR24S5u7ulmWIj48P6AlHxxeQF-xUmfI
	// Lookup for transaction by hash returned error: transaction not found for hash: MOCK-TXN~Q8U3UD4JrPhZfR24S5u7ulmWIj48P6AlHxxeQF-xUmfI
	// Fetched transaction by hash: MOCK-TXN~Q8U3UD4JrPhZfR24S5u7ulmWIj48P6AlHxxeQF-xUmfI
	// Transaction in block [0]: "example transaction payload"
	// Explorer Block Height: 1
	// Explorer Block Hash: MOCK-TXN~Q8U3UD4JrPhZfR24S5u7ulmWIj48P6AlHxxeQF-xUmfI
}

// ExampleProduceEspressoBlocksAtInterval demonstrates how to produce blocks
// at a regular interval using the MockEspressoChain, to more accurately
// reflect the real Espresso Chain.
func ExampleProduceEspressoBlocksAtInterval() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a new mock Espresso chain
	mockChain := chain.NewMockEspressoChain()

	{
		produceCtx, produceCancel := context.WithCancel(ctx)

		// We tell the mock chain to produce blocks at a 2 second interval,
		// to simulate a real Espresso chain that produces blocks at a regular
		// interval.
		go chain.ProduceEspressoBlocksAtInterval(produceCtx, mockChain, 2*time.Second)

		// Allow some time for blocks to be produced
		time.Sleep(5 * time.Second)

		// Stop the production of blocks
		produceCancel()
	}

	// Fetch the latest block height to verify that blocks were produced
	height, err := mockChain.FetchLatestBlockHeight(ctx)
	if err != nil {
		panic(fmt.Errorf("error fetching latest block height: %w\n", err))
	}

	fmt.Printf("Latest Block Height after producing blocks: %d\n", height)

	// Output:
	// Latest Block Height after producing blocks: 2
}

// ExampleErrorBlockNotFoundForHeight demonstrates how to create an error
// for a block not found at a specific height, which can occur if the block
// has not been produced yet or has been pruned.
func ExampleErrorBlockNotFoundForHeight() {
	err := chain.ErrorBlockNotFoundForHeight{
		Height: 1,
	}
	fmt.Printf("Error: %s\n", err.Error())

	// Output: Error: block not found at height: 1
}

// ExampleErrorFailedToComputeBlockHash demonstrates how to create an error
// for a failure in computing the block hash, which can occur if the block
// data is invalid or corrupted.
func ExampleErrorFailedToComputeBlockHash() {
	err := chain.ErrorFailedToComputeBlockHash{
		Cause: errors.New("example error"),
	}
	fmt.Printf("Error: %s\n", err.Error())

	// Output: Error: failed to compute block hash: example error
}

// ExampleErrorInvalidHash demonstrates how to create an error
// for an invalid hash, which can be nil or malformed.
func ExampleErrorInvalidHash() {
	{
		err := chain.ErrorInvalidHash{}
		fmt.Printf("Error nil hash: %s\n", err.Error())
	}

	{
		tag, err := tagged_base64.New("EXAMPLE", nil)
		if err != nil {
			panic(fmt.Sprintf("failed to create tagged base64: %v\n", err))
		}

		err = chain.ErrorInvalidHash{
			Hash: tag,
		}

		fmt.Printf("Error with hash: %s\n", err.Error())
	}

	// Output: Error nil hash: invalid hash: nil
	// Error with hash: invalid hash: EXAMPLE~2A
}

// ExampleErrorTransactionNotFoundForHash demonstrates how to create an error
// for a transaction not found by its hash.
func ExampleErrorTransactionNotFoundForHash() {
	tag, err := tagged_base64.New("EXAMPLE", nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create tagged base64: %v\n", err))
	}

	err = chain.ErrorTransactionNotFoundForHash{
		Hash: *tag,
	}

	fmt.Printf("Error: %s\n", err.Error())

	// Output: Error: transaction not found for hash: EXAMPLE~2A
}

// ExampleErrorSubmitTransaction demonstrates how to create an error
// for a failed transaction submission.
func ExampleErrorSubmitTransaction() {
	err := chain.ErrorSubmitTransaction{
		Cause: errors.New("failed to submit transaction"),
	}
	fmt.Printf("Error: %s\n", err.Error())

	// Output: Error: failed to submit transaction: failed to submit transaction
}

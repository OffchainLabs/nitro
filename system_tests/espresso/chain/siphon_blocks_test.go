package chain_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"

	"github.com/offchainlabs/nitro/system_tests/espresso/chain"
)

// generatePayloadOfSize generates a transaction payload with random data
// of the specified size.
func generatePayloadOfSize(size int) espresso_common.Transaction {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}

	return espresso_common.Transaction{
		Payload: bytes,
	}
}

// TestSiphonBlocksWithTransactions tests the SiphonBlocksWithTransactions
// wrapper around the Espresso client, ensuring that it correctly fetches
// transactions in blocks and sends them to a channel.
func TestSiphonBlocksWithTransactions(t *testing.T) {
	ch := make(chan espresso_client.TransactionsInBlock, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockClient := chain.NewMockEspressoChain()
	siphonClient := chain.NewSiphonBlocksWithTransactions(mockClient, ch)

	tx := generatePayloadOfSize(5_000)

	txnHash, err := mockClient.SubmitTransaction(ctx, generatePayloadOfSize(5_000))
	if have, want := err, (error)(nil); errors.Is(have, want) {
		t.Fatalf("expected no error submitting transaction 1:\nhave:\n\t%v\nwant:\n\t%v", have, want)
	}

	mockClient.Advance()

	transactionDetails, err := siphonClient.FetchTransactionByHash(ctx, txnHash)
	if have, want := err, (error)(nil); errors.Is(have, want) {
		t.Fatalf("expected no error fetching transaction by hash:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
	}

	// Fetch transactions in block
	_, err = siphonClient.FetchTransactionsInBlock(ctx, transactionDetails.BlockHeight, tx.Namespace)
	if have, want := err, (error)(nil); errors.Is(have, want) {
		t.Fatalf("expected no error fetching transactions in block:\nhave:\n\t%v\nwant:\n\t%v", have, want)
	}

	// Check if the transaction details were sent to the channel
	select {
	default:
		t.Fatal("expected to receive transaction details from channel, but got none")

	case result := <-ch:
		if have, want := len(result.Transactions), 1; have != want {
			t.Fatalf("expected the number of transactions in the block to match the expectation:\nhave:\n\t%v\nwant:\n\t%v", have, want)
		}

		txnZero := result.Transactions[0]

		if have, want := txnZero, tx.Payload; bytes.Equal(have, want) {
			t.Fatalf("expected transaction payload to match:\nhave:\n\t%x\nwant:\n\t%x", have, want)
		}
	}
}

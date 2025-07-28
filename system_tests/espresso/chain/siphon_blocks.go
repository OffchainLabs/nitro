package chain

import (
	"context"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
)

// SiphonBlocksWithTransactions is a wrapper around an Espresso client that
// fetches transactions in blocks and sends them to a channel. This is useful
// for siphoning blocks with their transactions in a streaming manner, allowing
// for real-time inspection of transaction data as they are fetched
// successfully.
type SiphonBlocksWithTransactions struct {
	espresso_client.EspressoClient
	ch chan<- espresso_client.TransactionsInBlock
}

var _ espresso_client.EspressoClient = &SiphonBlocksWithTransactions{}

// NewSiphonBlocksWithTransactions creates a new instance of
// SiphonBlocksWithTransactions with the specified Espresso client and channel.
func NewSiphonBlocksWithTransactions(
	client espresso_client.EspressoClient,
	ch chan<- espresso_client.TransactionsInBlock,
) *SiphonBlocksWithTransactions {
	return &SiphonBlocksWithTransactions{
		EspressoClient: client,
		ch:             ch,
	}
}

// FetchTransactionsInBlock implements espresso_client.EspressoClient
func (c *SiphonBlocksWithTransactions) FetchTransactionsInBlock(ctx context.Context, blockHeight uint64, namespace uint64) (espresso_client.TransactionsInBlock, error) {
	result, err := c.EspressoClient.FetchTransactionsInBlock(ctx, blockHeight, namespace)
	if err != nil {
		return result, err
	}

	// Send the result to the channel
	c.ch <- result
	return result, err
}

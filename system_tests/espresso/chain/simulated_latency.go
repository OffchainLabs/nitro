package chain

import (
	"context"
	"time"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	espresso_types "github.com/EspressoSystems/espresso-network/sdks/go/types"
	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"
)

// EspressoClientSimulatedLatency is a wrapper around an Espresso chain client
// that introduces a delay before executing any method calls. This can be
// useful for simulating network latency or for testing purposes.
type EspressoClientSimulatedLatency struct {
	espresso_client.EspressoClient
	transactionsInBlockDelay time.Duration
	transactionsByHashDelay  time.Duration
	submitTransactionDelay   time.Duration
	explorerTransactionDelay time.Duration
}

// SimulatedLatencyConfig holds the configuration for creating the
// EspressoClientSimulatedLatency. It includes the client and the delays for
// the various method calls.
type SimulatedLatencyConfig struct {
	EspressoClient           espresso_client.EspressoClient
	TransactionsInBlockDelay time.Duration
	TransactionsByHashDelay  time.Duration
	SubmitTransactionDelay   time.Duration
	ExplorerTransactionDelay time.Duration
}

// SimulatedLatencyOption is a function that modifies the
// SimulatedLatencyConfig. This allows for flexible configuration of the
// simulated latency without hardcoding values in the client implementation.
type SimulatedLatencyOption func(*SimulatedLatencyConfig)

// Ensure that SimulatedLatencyEspressoChain implements EspressoClient
var _ espresso_client.EspressoClient = &EspressoClientSimulatedLatency{}

// WithTransactionsInBlockDelay sets the delay for the FetchTransactionsInBlock
// method.
func WithTransactionsInBlockDelay(delay time.Duration) SimulatedLatencyOption {
	return func(config *SimulatedLatencyConfig) {
		config.TransactionsInBlockDelay = delay
	}
}

// WithTransactionsByHashDelay sets the delay for the FetchTransactionByHash
// method.
func WithTransactionsByHashDelay(delay time.Duration) SimulatedLatencyOption {
	return func(config *SimulatedLatencyConfig) {
		config.TransactionsByHashDelay = delay
	}
}

// WithSubmitTransactionDelay sets the delay for the SubmitTransaction method.
func WithSubmitTransactionDelay(delay time.Duration) SimulatedLatencyOption {
	return func(config *SimulatedLatencyConfig) {
		config.SubmitTransactionDelay = delay
	}
}

// WithExplorerTransactionDelay sets the delay for the
// FetchExplorerTransactionByHash method.
func WithExplorerTransactionDelay(delay time.Duration) SimulatedLatencyOption {
	return func(config *SimulatedLatencyConfig) {
		config.ExplorerTransactionDelay = delay
	}
}

// applySimulatedLatencyOptions applies the provided options to the
// SimulatedLatencyConfig. This allows for multiple options to be applied in a
// single call, making it easier to configure the client.
func applySimulatedLatencyOptions(config *SimulatedLatencyConfig, options ...SimulatedLatencyOption) {
	for _, option := range options {
		option(config)
	}
}

// WithAllDelaysSetTo is a SimulatedLatencyOption that sets all delays to the
// same value.
func WithAllDelaysSetTo(delay time.Duration) SimulatedLatencyOption {
	return func(config *SimulatedLatencyConfig) {
		applySimulatedLatencyOptions(config,
			WithTransactionsInBlockDelay(delay),
			WithTransactionsByHashDelay(delay),
			WithSubmitTransactionDelay(delay),
			WithExplorerTransactionDelay(delay),
		)
	}
}

// NewEspressoClientSimulatedLatency creates a new instance of EspressoChainDelayed
// with the specified chain client and delay duration. The delay will be applied
// to all method calls made to the chain client.
func NewEspressoClientSimulatedLatency(client espresso_client.EspressoClient, options ...SimulatedLatencyOption) *EspressoClientSimulatedLatency {
	const defaultDelay = 250 * time.Millisecond
	config := SimulatedLatencyConfig{
		EspressoClient:           client,
		TransactionsInBlockDelay: defaultDelay,
		TransactionsByHashDelay:  defaultDelay,
		SubmitTransactionDelay:   defaultDelay,
		ExplorerTransactionDelay: defaultDelay,
	}

	applySimulatedLatencyOptions(&config, options...)

	return &EspressoClientSimulatedLatency{
		EspressoClient:           config.EspressoClient,
		transactionsInBlockDelay: config.TransactionsInBlockDelay,
		transactionsByHashDelay:  config.TransactionsByHashDelay,
		submitTransactionDelay:   config.SubmitTransactionDelay,
		explorerTransactionDelay: config.ExplorerTransactionDelay,
	}
}

// This is a compile time check to ensure that MockEspressoChain implements
// EspressoClient.
var _ espresso_client.EspressoClient = &MockEspressoChain{}

// FetchTransactionsInBlock implements espresso_client.EspressoClient
func (c *EspressoClientSimulatedLatency) FetchTransactionsInBlock(ctx context.Context, blockHeight uint64, namespace uint64) (espresso_client.TransactionsInBlock, error) {
	time.Sleep(c.transactionsInBlockDelay)
	return c.EspressoClient.FetchTransactionsInBlock(ctx, blockHeight, namespace)
}

// FetchTransactionByHash implements espresso_client.EspressoClient
func (c *EspressoClientSimulatedLatency) FetchTransactionByHash(ctx context.Context, hash *espresso_types.TaggedBase64) (espresso_types.TransactionQueryData, error) {
	time.Sleep(c.transactionsByHashDelay)
	return c.EspressoClient.FetchTransactionByHash(ctx, hash)
}

// SubmitTransaction implements espresso_client.EspressoClient
func (c *EspressoClientSimulatedLatency) SubmitTransaction(ctx context.Context, tx espresso_common.Transaction) (*espresso_common.TaggedBase64, error) {
	time.Sleep(c.submitTransactionDelay)
	return c.EspressoClient.SubmitTransaction(ctx, tx)
}

// FetchExplorerTransactionByHash implements espresso_client.EspressoClient
func (c *EspressoClientSimulatedLatency) FetchExplorerTransactionByHash(ctx context.Context, hash *espresso_types.TaggedBase64) (espresso_types.ExplorerTransactionQueryData, error) {
	time.Sleep(c.explorerTransactionDelay)
	return c.EspressoClient.FetchExplorerTransactionByHash(ctx, hash)
}

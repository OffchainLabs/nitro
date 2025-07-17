package transaction_streamer

import (
	"context"
	"fmt"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/execution"
	chain "github.com/offchainlabs/nitro/system_tests/espresso/chain"
	execution_engine "github.com/offchainlabs/nitro/system_tests/espresso/execution_engine"
	"github.com/offchainlabs/nitro/system_tests/espresso/key_manager"
	"github.com/offchainlabs/nitro/system_tests/espresso/light_client"
)

// MockTransactionStreamerEnvironmentConfig is a configuration struct for
// creating a mock Transaction Streamer environment for testing purposes.
//
// It allows for custom configuration and setup of the testing environment
// with different configurable options.
type MockTransactionStreamerEnvironmentConfig struct {
	// Transaction Streamer Environment Configuration options
	Database                         ethdb.Database
	ChainConfig                      *params.ChainConfig
	Exec                             execution.ExecutionClient
	BroadcastServer                  *broadcaster.Broadcaster
	FatalErrChan                     chan<- error
	TransactionStreamerConfigFetcher arbnode.TransactionStreamerConfigFetcher
	SnapSyncConfig                   *arbnode.SnapSyncConfig

	// Espresso Environment Configuration options
	EspressoClientOptions []TransactionStreamerEspressoClientOption

	// Espresso Transaction Streamer Configuration
	EspressoTransactionStreamerOptions []arbnode.TransactionStreamerEspressoOption
}

// MockTransactionStreamerEnvironmentConfigDefault represents an option that
// allows for the modification of the MockTransactionStreamerEnvironmentConfig.
//
// This allows the caller to modify the configuration of the mock for specific
// testing scenarios.
type MockTransactionStreamerEnvironmentOption func(*MockTransactionStreamerEnvironmentConfig)

// TransactionStreamerEspressoClient is a function that modifies the
// TransactionStreamerEspressoClient in the
// MockTransactionStreamerEnvironmentConfig.
//
// This allows for the modification of the Espresso Client that is used in the
// TransactionStreamer environment.
//
// This is primarily utilized to add layers of functionality to the
// TransactionStreamerEspressoClient.
type TransactionStreamerEspressoClientOption func(input espresso_client.EspressoClient) espresso_client.EspressoClient

// ErrorFailedToCreateTransactionStreamer is an error type that indicates that
// the TransactionStreamer could not be created successfully.
type ErrorFailedToCreateTransactionStreamer struct {
	Cause error
}

// Error implements error
func (e ErrorFailedToCreateTransactionStreamer) Error() string {
	return fmt.Sprintf("failed to create TransactionStreamer: %v", e.Cause)
}

// ErrorFailedToCreateEspressoSubmitter is an error type that indicates that
// the EspressoSubmitter could not be created successfully.
type ErrorFailedToCreateEspressoSubmitter struct {
	Cause error
}

// Error implements error
func (e ErrorFailedToCreateEspressoSubmitter) Error() string {
	return fmt.Sprintf("failed to create EspressoSubmitter: %v", e.Cause)
}

// WithADatabase is a function that sets the Database in the
// MockTransactionStreamerEnvironmentConfig.
func WithDatabase(database ethdb.Database) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.Database = database
	}
}

// WithChainConfig is a function that sets the ChainConfig in the
// MockTransactionStreamerEnvironmentConfig.
func WithChainConfig(chainConfig *params.ChainConfig) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.ChainConfig = chainConfig
	}
}

// WithExecutionSequencer is a function that sets the ExecutionSequencer in the
// MockTransactionStreamerEnvironmentConfig.
func WithExecutionSequencer(exec execution.ExecutionClient) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.Exec = exec
	}
}

// WithBroadcastServer is a function that sets the BroadcastServer in the
// MockTransactionStreamerEnvironmentConfig.
func WithBroadcastServer(broadcastServer *broadcaster.Broadcaster) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.BroadcastServer = broadcastServer
	}
}

// WithFatalErrChan is a function that sets the FatalErrChan in the
// MockTransactionStreamerEnvironmentConfig.
func WithFatalErrChan(fatalErrChan chan<- error) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.FatalErrChan = fatalErrChan
	}
}

// WithTransactionStreamerConfigFetcher is a function that sets the
// TransactionStreamerConfigFetcher in the
// MockTransactionStreamerEnvironmentConfig.
func WithTransactionStreamerConfigFetcher(fetcher arbnode.TransactionStreamerConfigFetcher) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.TransactionStreamerConfigFetcher = fetcher
	}
}

// WithSnapSyncConfig is a function that sets the SnapSyncConfig in the
// MockTransactionStreamerEnvironmentConfig.
func WithSnapSyncConfig(snapSyncConfig *arbnode.SnapSyncConfig) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.SnapSyncConfig = snapSyncConfig
	}
}

// AddEspressoClientOptions is a function that adds options to the Espresso
// Client in the MockTransactionStreamerEnvironmentConfig.
func AddEspressoClientOptions(options ...TransactionStreamerEspressoClientOption) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.EspressoClientOptions = append(config.EspressoClientOptions, options...)
	}
}

// WithEspressoTransactionStreamerOptions is a function that adds options to the
// Espresso Transaction Streamer in the MockTransactionStreamerEnvironmentConfig.
func WithEspressoTransactionStreamerOptions(options ...arbnode.TransactionStreamerEspressoOption) MockTransactionStreamerEnvironmentOption {
	return func(config *MockTransactionStreamerEnvironmentConfig) {
		config.EspressoTransactionStreamerOptions = append(config.EspressoTransactionStreamerOptions, options...)
	}
}

// NewMockTransactionStreamerEnvironment creates a mock Transaction Streamer
// environment for testing purposes.
//
// The function operates in such a way that it allows for the passed options
// to modify the configuration of the mock environment (as needed), yet
// should still provides a default configuration that is suitable for
// most testing scenarios.
//
// Such a setup / approach allows for the consumer of the mock to only
// modify what is necessary for their specific test case.
//
// NOTE: This does not start the TransactionStreamer, start submitting messages
// to the TransactionStreamer, or start producing blocks in the mock Espresso
// Chain. Those actions are left to the caller of this function, so they are
// able to control the timing, and configuration of those actions.
func NewMockTransactionStreamerEnvironment(ctx context.Context, options ...MockTransactionStreamerEnvironmentOption) (*chain.MockEspressoChain, espresso_client.EspressoClient, *arbnode.TransactionStreamer, error) {
	// Setup the Default Configuration for the Mock Transaction Streamer Environment
	config := &MockTransactionStreamerEnvironmentConfig{
		Database:     rawdb.NewMemoryDatabase(),
		ChainConfig:  params.TestChainConfig,
		Exec:         execution_engine.NewMockExecutionEngineForTransactionStreamer(),
		FatalErrChan: make(chan error),
		TransactionStreamerConfigFetcher: func() *arbnode.TransactionStreamerConfig {
			return &arbnode.DefaultTransactionStreamerConfig
		},
	}

	// Apply the provided options to the configuration
	for _, option := range options {
		option(config)
	}

	// Create a mock Espresso Chain
	espressoChain := chain.NewMockEspressoChain()

	// Create and configure the Espresso Client
	var espressoClient espresso_client.EspressoClient = espressoChain
	for _, option := range config.EspressoClientOptions {
		espressoClient = option(espressoClient)
	}

	// Create the initial Streamer configuration
	streamer, err := arbnode.NewTransactionStreamer(
		ctx,
		config.Database,
		config.ChainConfig,
		config.Exec,
		config.BroadcastServer,
		config.FatalErrChan,
		config.TransactionStreamerConfigFetcher,
		config.SnapSyncConfig,
	)

	if err != nil {
		return nil, nil, nil, ErrorFailedToCreateTransactionStreamer{Cause: err}
	}

	// Configure the Transaction Streamer with the Espresso Fields and options
	arbnode.ConfigureEspressoFields(
		streamer,
		// We setup the initial values before the other options are applied,
		// so that the other options can modify them as needed.
		arbnode.WithEspressoClient(espressoClient),
		arbnode.WithLightClientReader(light_client.NewMockAlwaysLiveLightClientReader()),
		arbnode.WithKeyManager(key_manager.NewMockEspressoKeyManager()),

		// Add the passed in options to the Espresso Transaction Streamer
		arbnode.WithMultipleEspressoOptions(
			config.EspressoTransactionStreamerOptions...,
		),
	)

	return espressoChain, espressoClient, streamer, nil
}

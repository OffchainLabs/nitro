package arbnode

// This file contains a few helper functions to help interoperate with the
// TransactionStreamer and the Espresso code.
//
// These functions are used for testing and configuration purposes, and are
// not intended to general common usage.

import (
	"math/big"
	"time"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	espresso_light_client "github.com/EspressoSystems/espresso-network/sdks/go/light-client"
)

// TransactionStreamerEspressoConfig is a configuration struct for the
// TransactionStreamer that contains fields specific to our Espresso integration
// efforts.
type TransactionStreamerEspressoConfig struct {
	EspressoClient                        espresso_client.EspressoClient
	LightClientReader                     espresso_light_client.LightClientReaderInterface
	KeyManager                            EspressoKeyManagerInterface
	TxnsPollingInterval                   time.Duration
	TxnsSendingInterval                   time.Duration
	TxnsResubmissionInterval              time.Duration
	ResubmitEspressoTxDeadline            time.Duration
	UseEscapeHatch                        bool
	EscapeHatchEnabled                    bool
	MaxTransactionSize                    int64
	MaxBlockLagBeforeEscapeHatch          uint64
	InitialFinalizedSequencerMessageCount *big.Int
}

// TransactionStreamerEspressoOption is a functional option type for configuring
// the TransactionStreamerEspressoConfig.
type TransactionStreamerEspressoOption func(*TransactionStreamerEspressoConfig)

// WithEspressoClient is a functional option to set the Espresso client in the
// TransactionStreamerEspressoConfig.
func WithEspressoClient(espressoClient espresso_client.EspressoClient) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.EspressoClient = espressoClient
	}
}

// WithLightClientReader is a functional option to set the LightClientReader
// in the TransactionStreamerEspressoConfig.
func WithLightClientReader(lightClientReader espresso_light_client.LightClientReaderInterface) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.LightClientReader = lightClientReader
	}
}

// WithKeyManager is a functional option to set the EspressoKeyManagerInterface
// in the TransactionStreamerEspressoConfig.
func WithKeyManager(keyManager EspressoKeyManagerInterface) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.KeyManager = keyManager
	}
}

// WithTxnsPollingInterval is a functional option to set the transaction polling
// interval in the TransactionStreamerEspressoConfig.
func WithTxnsPollingInterval(interval time.Duration) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.TxnsPollingInterval = interval
	}
}

// WithTxnSendingInterval is a functional option to set the transaction sending
// interval in the TransactionStreamerEspressoConfig.
func WithTxnSendingInterval(interval time.Duration) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.TxnsSendingInterval = interval
	}
}

// WithTxnResubmissionInterval is a functional option to set the transaction
// resubmission interval in the TransactionStreamerEspressoConfig.
func WithTxnResubmissionInterval(interval time.Duration) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.TxnsResubmissionInterval = interval
	}
}

// WithResubmitEspressoTxDeadline is a functional option to set the deadline for
// resubmitting Espresso transactions in the TransactionStreamerEspressoConfig.
func WithResubmitEspressoTxDeadline(deadline time.Duration) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.ResubmitEspressoTxDeadline = deadline
	}
}

// WithUseEscapeHatch is a functional option to enable or disable the use of the
// escape hatch in the TransactionStreamerEspressoConfig.
func WithUseEscapeHatch(enable bool) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.UseEscapeHatch = enable
	}
}

// WithMaxTransactionSize is a functional option to set the maximum transaction
// size in the TransactionStreamerEspressoConfig.
func WithEscapeHatchEnabled(enable bool) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.EscapeHatchEnabled = enable
	}
}

// WithMaxBlockLagBeforeEscapeHatch is a functional option to set the maximum
// block lag before the escape hatch is triggered in the
// TransactionStreamerEspressoConfig.
func WithMaxTransactionSize(size int64) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.MaxTransactionSize = size
	}
}

// WithMaxBlockLagBeforeEscapeHatch is a functional option to set the maximum
// block lag before the escape hatch is triggered in the
// TransactionStreamerEspressoConfig.
func WithMaxBlockLagBeforeEscapeHatch(maxBlockLag uint64) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.MaxBlockLagBeforeEscapeHatch = maxBlockLag
	}
}

// WithInitialFinalizedSequencerMessageCount is a functional option to set the
// initial finalized sequencer message count in the TransactionStreamerEspressoConfig.
func WithInitialFinalizedSequencerMessageCount(count *big.Int) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		config.InitialFinalizedSequencerMessageCount = count
	}
}

// WithMultipleEspressoOptions is a functional option that allows for multiple
// TransactionStreamerEspressoOptions to be applied at once. This is useful for
// configuring the TransactionStreamer with multiple options in a single call.
func WithMultipleEspressoOptions(options ...TransactionStreamerEspressoOption) TransactionStreamerEspressoOption {
	return func(config *TransactionStreamerEspressoConfig) {
		applyEspressoOptions(config, options...)
	}
}

// applyEspressoOptions is a helper function that applies the provided options
// to the TransactionStreamerEspressoConfig.
func applyEspressoOptions(
	config *TransactionStreamerEspressoConfig,
	options ...TransactionStreamerEspressoOption,
) {
	for _, option := range options {
		option(config)
	}
}

// ConfigureEspressoFields is a function that configures the Espresso fields
// in the TransactionStreamer with the provided options.
//
// Since many of the Espresso fields are optional, this function allows for
// flexible modification of the TransactionStreamer configuration
// without hardcoding values in the TransactionStreamer implementation.
//
// This is useful, and even necessary, for testing purposes, as it allows
// for easy configuration of the TransactionStreamer with various Espresso
// fields that would otherwise be impossible to set without
// modifying the TransactionStreamer implementation itself.
func ConfigureEspressoFields(
	streamer *TransactionStreamer,
	options ...TransactionStreamerEspressoOption,
) {
	config := TransactionStreamerEspressoConfig{
		InitialFinalizedSequencerMessageCount: big.NewInt(0),
		TxnsPollingInterval:                   DefaultBatchPosterConfig.EspressoTxnsPollingInterval,
		TxnsSendingInterval:                   DefaultBatchPosterConfig.EspressoTxnsSendingInterval,
		TxnsResubmissionInterval:              DefaultBatchPosterConfig.EspressoTxnsResubmissionInterval,
		MaxTransactionSize:                    DefaultBatchPosterConfig.EspressoTxSizeLimit,
		ResubmitEspressoTxDeadline:            DefaultBatchPosterConfig.ResubmitEspressoTxDeadline,
		UseEscapeHatch:                        DefaultBatchPosterConfig.UseEscapeHatch,
	}

	applyEspressoOptions(&config, options...)

	streamer.espressoClient = config.EspressoClient
	streamer.lightClientReader = config.LightClientReader
	streamer.EspressoKeyManager = config.KeyManager
	streamer.espressoTxnsPollingInterval = config.TxnsPollingInterval
	streamer.espressoTxnsSendingInterval = config.TxnsSendingInterval
	streamer.espressoTxnsResubmissionInterval = config.TxnsResubmissionInterval
	streamer.resubmitEspressoTxDeadline = config.ResubmitEspressoTxDeadline
	streamer.UseEscapeHatch = config.UseEscapeHatch
	streamer.EscapeHatchEnabled = config.EscapeHatchEnabled
	streamer.espressoMaxTransactionSize = config.MaxTransactionSize
	streamer.maxBlockLagBeforeEscapeHatch = config.MaxBlockLagBeforeEscapeHatch
	streamer.InitialFinalizedSequencerMessageCount = config.InitialFinalizedSequencerMessageCount
}

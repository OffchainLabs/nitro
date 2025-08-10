package submitter

import (
	"fmt"
	"math/big"
	"time"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	espresso_light_client "github.com/EspressoSystems/espresso-network/sdks/go/light-client"

	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	espresso_key_manager "github.com/offchainlabs/nitro/espresso/key-manager"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MessageGetter interface {
	GetMessage(seqNum arbutil.MessageIndex) (*arbostypes.MessageWithMetadata, error)
}

type EspressoSubmitter interface {
	Start(sw *stopwaiter.StopWaiter) error
	NotifyNewPendingMessages(pos arbutil.MessageIndex, messages []arbostypes.MessageWithMetadataAndBlockInfo) error
	GetKeyManager() espresso_key_manager.EspressoKeyManagerInterface
	RegisterSigner() error
}

// EspressoSubmitterConfig holds the configuration options for implementations
// of the [EspressoSubmitter] interface.
type EspressoSubmitterConfig struct {
	// Simple Configuration values. These will be expected to have default
	// values set for them, but can be overridden by the user.

	ChainID                               uint64
	EspressoTxnsPollingInterval           time.Duration
	EspressoTxnSendingInterval            time.Duration
	EspressoTxnsResubmissionInterval      time.Duration
	EspressoMaxTransactionSize            int64
	ResubmitEspressoTxDeadline            time.Duration
	InitialFinalizedSequencerMessageCount *big.Int

	// These are attestation values that will signify information to load
	// for attestation initialization

	UserDataAttestationFile string
	QuoteFile               string

	// These are the interfaces that will be used to interact with
	// Espresso, and the underlying chain information.

	EspressoClient    espresso_client.EspressoClient
	LightClientReader espresso_light_client.LightClientReaderInterface
	KeyManager        espresso_key_manager.EspressoKeyManagerInterface
	MessageGetter     MessageGetter
	Db                ethdb.Database
}

// DefaultEspressoSubmitterConfig provides a default configuration for the
// EspressoSubmitter.
//
// It includes default values for the following fields:
// - EspressoTxnsPollingInterval
// - EspressoTxnSendingInterval
// - MaxBlockLagBeforeEscapeHatch
// - EspressoMaxTransactionSize
// - ResubmitEspressoTxDeadline
// - InitialFinalizedSequencerMessageCount
//
// NOTE: The following fields are not set by default and must be provided:
// - EspressoClient
// - LightClientReader
// - MessageGetter
var DefaultEspressoSubmitterConfig = EspressoSubmitterConfig{
	EspressoTxnsPollingInterval:           time.Second,
	EspressoTxnSendingInterval:            time.Second,
	EspressoMaxTransactionSize:            200_000,
	ResubmitEspressoTxDeadline:            16 * time.Second,
	InitialFinalizedSequencerMessageCount: big.NewInt(0),
}

// EspressoSubmitterConfigOption is a function type that takes a pointer to
// [EspressoSubmitterConfig] and modifies it.
//
// This allows for flexible configuration of the EspressoSubmitter by
// passing in various options to a variadic function based on the
// [Functional Options] pattern.
//
// [Functional Options]: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type EspressoSubmitterConfigOption func(*EspressoSubmitterConfig)

// applyEspressoSubmitterConfigOptions is a helper function that applies
// a list of [EspressoSubmitterConfigOption] functions to the provided
// [EspressoSubmitterConfig].
func applyEspressoSubmitterConfigOptions(
	config *EspressoSubmitterConfig,
	options ...EspressoSubmitterConfigOption,
) {
	for _, option := range options {
		option(config)
	}
}

// WithMultipleOptions is a variadic function that takes multiple
// [EspressoSubmitterConfigOption] functions and applies them to the provided
// [EspressoSubmitterConfig].
func WithMultipleOptions(options ...EspressoSubmitterConfigOption) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		applyEspressoSubmitterConfigOptions(config, options...)
	}
}

// WithChainID is an [EspressoSubmitterConfigOption] that sets the ChainID in
// the [EspressoSubmitterConfig].
func WithChainID(chainID uint64) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.ChainID = chainID
	}
}

// WithMessageGetter is an [EspressoSubmitterConfigOption] that sets the
// [MessageGetter] in the
func WithMessageGetter(getter MessageGetter) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.MessageGetter = getter
	}
}

// WithDatabase is an [EspressoSubmitterConfigOption] that sets the
// [ethdb.Database]  in the [EspressoSubmitterConfig].
func WithDatabase(db ethdb.Database) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.Db = db
	}
}

// WithAttestationFiles is an [EspressoSubmitterConfigOption] that sets the
// user data attestation file and quote file in the [EspressoSubmitterConfig].
func WithAttestationFiles(userDataAttestationFile, quoteFile string) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.UserDataAttestationFile = userDataAttestationFile
		config.QuoteFile = quoteFile
	}
}

// WithEspressoClient is an [EspressoSubmitterConfigOption] that sets the
// [espresso_client.EspressoClient] in the [EspressoSubmitterConfig].
func WithEspressoClient(client espresso_client.EspressoClient) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.EspressoClient = client
	}
}

// WithLightClientReader is an [EspressoSubmitterConfigOption] that sets the
// [espresso_light_client.LightClientReaderInterface] in the
// [EspressoSubmitterConfig].
func WithLightClientReader(reader espresso_light_client.LightClientReaderInterface) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.LightClientReader = reader
	}
}

// WithKeyManager is an [EspressoSubmitterConfigOption] that sets the
// [espresso_key_manager.EspressoKeyManagerInterface] in the
// [EspressoSubmitterConfig].
func WithKeyManager(keyManager espresso_key_manager.EspressoKeyManagerInterface) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.KeyManager = keyManager
	}
}

// WithMaxTransactionSize is an [EspressoSubmitterConfigOption] that sets the
// maximum transaction size in the [EspressoSubmitterConfig].
func WithMaxTransactionSize(size int64) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.EspressoMaxTransactionSize = size
	}
}

// WithTxnsSendingInterval is an [EspressoSubmitterConfigOption] that sets the
// transaction sending interval in the [EspressoSubmitterConfig].
func WithTxnsSendingInterval(interval time.Duration) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.EspressoTxnSendingInterval = interval
	}
}

// WithTxnsPollingInterval is an [EspressoSubmitterConfigOption] that sets the
// transaction polling interval in the [EspressoSubmitterConfig].
func WithTxnsPollingInterval(interval time.Duration) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.EspressoTxnsPollingInterval = interval
	}
}

// WithTxnsResubmissionInterval is an [EspressoSubmitterConfigOption] that sets
// the transaction resubmission interval in the [EspressoSubmitterConfig].
func WithTxnsResubmissionInterval(interval time.Duration) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.EspressoTxnsResubmissionInterval = interval
	}
}

// WithUseEscapeHatch is an [EspressoSubmitterConfigOption] that sets whether
// to use the escape hatch in the [EspressoSubmitterConfig].
func WithInitialFinalizedSequencerMessageCount(count *big.Int) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.InitialFinalizedSequencerMessageCount = count
	}
}

// WithResubmitEspressoTxDeadline is an [EspressoSubmitterConfigOption] that
// sets the deadline for resubmitting Espresso transactions in the
// [EspressoSubmitterConfig].
func WithResubmitEspressoTxDeadline(deadline time.Duration) EspressoSubmitterConfigOption {
	return func(config *EspressoSubmitterConfig) {
		config.ResubmitEspressoTxDeadline = deadline
	}
}

// ValidateEspressoSubmitterConfig checks if the provided
// [EspressoSubmitterConfig] is valid.
//
// It returns an error if any of the following required fields are set to their
// zero value:
// - EspressoClient
// - LightClientReader
// - MessageGetter
// - Db
// - KeyManager
// - ChainID
// - EspressoMaxTransactionSize
// - EspressoTxnsPollingInterval
// - EspressoTxnSendingInterval
func ValidateEspressoSubmitterConfig(config EspressoSubmitterConfig) error {
	if config.EspressoClient == nil {
		return fmt.Errorf("espresso client is not set")
	}

	if config.LightClientReader == nil {
		return fmt.Errorf("light client reader is not set")
	}

	if config.MessageGetter == nil {
		return fmt.Errorf("message getter is not set")
	}

	if config.Db == nil {
		return fmt.Errorf("database is not set")
	}

	if config.KeyManager == nil {
		return fmt.Errorf("espresso key manager is not set")
	}

	if config.ChainID == 0 {
		return fmt.Errorf("chain ID is not set")
	}

	if config.EspressoMaxTransactionSize <= 0 {
		return fmt.Errorf("espresso max transaction size must be greater than 0")
	}

	if config.EspressoTxnsPollingInterval <= 0 {
		return fmt.Errorf("espresso transactions polling interval must be greater than 0")
	}

	if config.EspressoTxnSendingInterval <= 0 {
		return fmt.Errorf("espresso transactions submission interval must be greater than 0")
	}

	return nil
}

// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package dapserver contains temporary DAS migration code
// TODO: This file is temporary and will be removed once DA provider initialization
// is moved out of arbnode/node.go on the custom-da branch
package dapserver

import (
	"context"
	"net/http"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

// DASServerConfig is the configuration for a DAS server
// TODO: This is temporary and duplicates dasserver.ServerConfig
// It will be removed when DAS initialization moves to the factory pattern
type DASServerConfig struct {
	Addr               string                              `koanf:"addr"`
	Port               uint64                              `koanf:"port"`
	JWTSecret          string                              `koanf:"jwtsecret"`
	EnableDAWriter     bool                                `koanf:"enable-da-writer"`
	DataAvailability   das.DataAvailabilityConfig          `koanf:"data-availability"`
	ServerTimeouts     genericconf.HTTPServerTimeoutConfig `koanf:"server-timeouts"`
	RPCServerBodyLimit int                                 `koanf:"rpc-server-body-limit"`
}

// DefaultDASServerConfig provides default values for DAS server configuration
// TODO: This is temporary and will be removed with the migration
var DefaultDASServerConfig = DASServerConfig{
	Addr:               "localhost",
	Port:               9880,
	JWTSecret:          "",
	EnableDAWriter:     false,
	ServerTimeouts:     genericconf.HTTPServerTimeoutConfigDefault,
	RPCServerBodyLimit: genericconf.HTTPServerBodyLimitDefault,
	DataAvailability:   das.DefaultDataAvailabilityConfig,
}

// ServerConfigAddDASOptions adds DAS-specific command-line options
// TODO: This is temporary and will be removed when DAS config moves elsewhere
func ServerConfigAddDASOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".addr", DefaultDASServerConfig.Addr, "JSON rpc server listening interface")
	f.Uint64(prefix+".port", DefaultDASServerConfig.Port, "JSON rpc server listening port")
	f.String(prefix+".jwtsecret", DefaultDASServerConfig.JWTSecret, "path to file with jwtsecret for validation")
	f.Bool(prefix+".enable-da-writer", DefaultDASServerConfig.EnableDAWriter, "implies if the das server supports daprovider's writer interface")
	f.Int(prefix+".rpc-server-body-limit", DefaultDASServerConfig.RPCServerBodyLimit, "HTTP-RPC server maximum request body size in bytes; the default (0) uses geth's 5MB limit")
	das.DataAvailabilityConfigAddNodeOptions(prefix+".data-availability", f)
	genericconf.HTTPServerTimeoutConfigAddOptions(prefix+".server-timeouts", f)
}

// NewServerForDAS creates a new DA provider server configured for DAS/AnyTrust
// TODO: This is temporary. On the custom-da branch, this initialization logic
// moves to the factory pattern and this function will be removed.
//
// Returns:
// - *http.Server: The HTTP server instance
// - func(): Cleanup function to stop the DAS lifecycle manager
// - error: Any error that occurred during initialization
func NewServerForDAS(
	ctx context.Context,
	config *DASServerConfig,
	dataSigner signature.DataSignerFunc,
	l1Client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	sequencerInboxAddr common.Address,
) (*http.Server, func(), error) {
	// Initialize DAS components
	var err error
	var daWriter dasutil.DASWriter
	var daReader dasutil.DASReader
	var dasKeysetFetcher *das.KeysetFetcher
	var dasLifecycleManager *das.LifecycleManager

	if config.EnableDAWriter {
		// Create both reader and writer for sequencer nodes
		daWriter, daReader, dasKeysetFetcher, dasLifecycleManager, err = das.CreateDAReaderAndWriter(
			ctx, &config.DataAvailability, dataSigner, l1Client, sequencerInboxAddr,
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Create only reader for non-sequencer nodes
		daReader, dasKeysetFetcher, dasLifecycleManager, err = das.CreateDAReader(
			ctx, &config.DataAvailability, l1Reader, &sequencerInboxAddr,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	// Apply DAS-specific wrappers
	daReader = das.NewReaderTimeoutWrapper(daReader, config.DataAvailability.RequestTimeout)
	if config.DataAvailability.PanicOnError {
		if daWriter != nil {
			daWriter = das.NewWriterPanicWrapper(daWriter)
		}
		daReader = das.NewReaderPanicWrapper(daReader)
	}

	// Convert to daprovider interfaces
	var writer daprovider.Writer
	if daWriter != nil {
		writer = dasutil.NewWriterForDAS(daWriter, config.DataAvailability.MaxBatchSize)
	}
	reader := dasutil.NewReaderForDAS(daReader, dasKeysetFetcher, daprovider.KeysetValidate)

	// Translate DAS config to generic server config
	serverConfig := ServerConfig{
		Addr:               config.Addr,
		Port:               config.Port,
		JWTSecret:          config.JWTSecret,
		EnableDAWriter:     config.EnableDAWriter,
		ServerTimeouts:     config.ServerTimeouts,
		RPCServerBodyLimit: config.RPCServerBodyLimit,
	}

	// Create the generic DA provider server with DAS components
	// Support both DAS without tree flag (0x80) and with tree flag (0x88)
	server, err := NewServerWithDAPProvider(
		ctx,
		&serverConfig,
		reader,
		writer,
		nil, // DAS doesn't use a validator
		[]byte{
			daprovider.DASMessageHeaderFlag,
			daprovider.DASMessageHeaderFlag | daprovider.TreeDASMessageHeaderFlag,
		},
		data_streaming.PayloadCommitmentVerifier(),
	)
	if err != nil {
		// Clean up lifecycle manager if server creation fails
		if dasLifecycleManager != nil {
			dasLifecycleManager.StopAndWaitUntil(2 * time.Second)
		}
		return nil, nil, err
	}

	// Return server and cleanup function for the lifecycle manager
	cleanupFn := func() {
		if dasLifecycleManager != nil {
			log.Info("Stopping DAS lifecycle manager")
			dasLifecycleManager.StopAndWaitUntil(2 * time.Second)
		}
	}

	return server, cleanupFn, nil
}

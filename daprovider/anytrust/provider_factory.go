// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/daprovider"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

// FactoryMode selects which AnyTrust DA components the factory will construct.
type FactoryMode int

const (
	// ModeReader builds a reader only, against rest-aggregator. No writer is built.
	ModeReader FactoryMode = iota
	// ModeWriter builds both a reader and a writer. Requires rpc-aggregator and rest-aggregator configured.
	ModeWriter
	// ModeRetiring suppresses the writer entirely. The reader is the rest-aggregator if enabled, or a
	// halt-on-any-AnyTrust-batch reader if rest-aggregator is disabled. Used for chains being retired off AnyTrust.
	ModeRetiring
)

func (m FactoryMode) String() string {
	switch m {
	case ModeReader:
		return "reader"
	case ModeWriter:
		return "writer"
	case ModeRetiring:
		return "retiring"
	default:
		return fmt.Sprintf("FactoryMode(%d)", int(m))
	}
}

// lint:require-exhaustive-initialization
type Factory struct {
	config       *Config
	dataSigner   signature.DataSignerFunc
	l1Client     *ethclient.Client
	l1Reader     *headerreader.HeaderReader
	seqInboxAddr common.Address
	mode         FactoryMode
}

// SupportedHeaderBytes are the header bytes supported by AnyTrust DA.
var SupportedHeaderBytes = []byte{
	daprovider.AnyTrustMessageHeaderFlag,
	daprovider.AnyTrustMessageHeaderFlag | daprovider.AnyTrustTreeMessageHeaderFlag,
}

// NewFactory validates the supplied config against mode and returns the factory.
// A returned *Factory is guaranteed validated; callers do not need to call ValidateConfig.
func NewFactory(
	config *Config,
	dataSigner signature.DataSignerFunc,
	l1Client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddr common.Address,
	mode FactoryMode,
) (*Factory, error) {
	f := &Factory{
		config:       config,
		dataSigner:   dataSigner,
		l1Client:     l1Client,
		l1Reader:     l1Reader,
		seqInboxAddr: seqInboxAddr,
		mode:         mode,
	}
	if err := f.ValidateConfig(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Factory) GetSupportedHeaderBytes() []byte {
	return []byte{
		daprovider.AnyTrustMessageHeaderFlag,
		daprovider.AnyTrustMessageHeaderFlag | daprovider.AnyTrustTreeMessageHeaderFlag,
	}
}

func (f *Factory) ValidateConfig() error {
	if !f.config.Enable {
		return errors.New("anytrust data availability must be enabled")
	}
	switch f.mode {
	case ModeRetiring:
		if f.config.RPCAggregator.Enable {
			return errors.New("always-fallback-to-parent-chain-da is incompatible with rpc-aggregator.enable=true; disable rpc-aggregator (writer-only) in the same config edit")
		}
		return nil
	case ModeWriter:
		if !f.config.RPCAggregator.Enable || !f.config.RestAggregator.Enable {
			return errors.New("rpc-aggregator.enable and rest-aggregator.enable must be set when running writer mode")
		}
		return nil
	case ModeReader:
		if f.config.RPCAggregator.Enable {
			return errors.New("rpc-aggregator is only for writer mode")
		}
		if !f.config.RestAggregator.Enable {
			return errors.New("rest-aggregator.enable must be set for reader mode")
		}
		return nil
	default:
		return fmt.Errorf("unknown factory mode: %d", f.mode)
	}
}

func (f *Factory) CreateReader(ctx context.Context) (daprovider.Reader, func(), error) {
	if f.mode == ModeRetiring && !f.config.RestAggregator.Enable {
		return &daprovider.DangerousAlwaysFallbackReader{}, nil, nil
	}

	var daReader anytrustutil.Reader
	var keysetFetcher *KeysetFetcher
	var lifecycleManager *LifecycleManager
	var err error

	if f.mode == ModeWriter {
		_, daReader, keysetFetcher, lifecycleManager, err = CreateDAReaderAndWriter(
			ctx, f.config, f.dataSigner, f.l1Client, f.seqInboxAddr)
	} else {
		daReader, keysetFetcher, lifecycleManager, err = CreateDAReader(
			ctx, f.config, f.l1Reader, &f.seqInboxAddr)
	}

	if err != nil {
		return nil, nil, err
	}

	daReader = NewReaderTimeoutWrapper(daReader, f.config.RequestTimeout)
	if f.config.PanicOnError {
		daReader = NewReaderPanicWrapper(daReader)
	}

	reader := anytrustutil.NewReader(daReader, keysetFetcher, daprovider.KeysetValidate)
	cleanupFn := func() {
		if lifecycleManager != nil {
			lifecycleManager.StopAndWaitUntil(0)
		}
	}
	return reader, cleanupFn, nil
}

func (f *Factory) CreateWriter(ctx context.Context) (daprovider.Writer, func(), error) {
	if f.mode != ModeWriter {
		return nil, nil, nil
	}

	daWriter, _, _, lifecycleManager, err := CreateDAReaderAndWriter(
		ctx, f.config, f.dataSigner, f.l1Client, f.seqInboxAddr)
	if err != nil {
		return nil, nil, err
	}

	if f.config.PanicOnError {
		daWriter = NewWriterPanicWrapper(daWriter)
	}

	writer := anytrustutil.NewWriter(daWriter, f.config.MaxBatchSize)
	cleanupFn := func() {
		if lifecycleManager != nil {
			lifecycleManager.StopAndWaitUntil(0)
		}
	}
	return writer, cleanupFn, nil
}

func (f *Factory) CreateValidator(ctx context.Context) (daprovider.Validator, func(), error) {
	return nil, nil, nil
}

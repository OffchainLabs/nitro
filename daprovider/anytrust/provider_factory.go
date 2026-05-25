// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/daprovider"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

// lint:require-exhaustive-initialization
type Factory struct {
	config         *Config
	dataSigner     signature.DataSignerFunc
	l1Client       *ethclient.Client
	l1Reader       *headerreader.HeaderReader
	seqInboxAddr   common.Address
	enableWriter   bool
	alwaysFallback bool
}

// SupportedHeaderBytes are the header bytes supported by AnyTrust DA.
var SupportedHeaderBytes = []byte{
	daprovider.AnyTrustMessageHeaderFlag,
	daprovider.AnyTrustMessageHeaderFlag | daprovider.AnyTrustTreeMessageHeaderFlag,
}

// NewFactory creates a new AnyTrust DA provider factory.
// alwaysFallback is --node.dangerous.always-fallback-to-parent-chain-da; see ValidateConfig and CreateReader.
func NewFactory(
	config *Config,
	dataSigner signature.DataSignerFunc,
	l1Client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddr common.Address,
	enableWriter bool,
	alwaysFallback bool,
) *Factory {
	return &Factory{
		config:         config,
		dataSigner:     dataSigner,
		l1Client:       l1Client,
		l1Reader:       l1Reader,
		seqInboxAddr:   seqInboxAddr,
		enableWriter:   enableWriter,
		alwaysFallback: alwaysFallback,
	}
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

	if f.alwaysFallback {
		if f.enableWriter {
			return errors.New("always-fallback-to-parent-chain-da requires enableWriter=false; the dangerous flag disallows posting new AnyTrust batches")
		}
		if f.config.RPCAggregator.Enable {
			log.Warn("always-fallback-to-parent-chain-da is set but rpc-aggregator is still enabled; ignoring rpc-aggregator (writer-only)")
		}
		return nil
	}

	if f.enableWriter {
		if !f.config.RPCAggregator.Enable || !f.config.RestAggregator.Enable {
			return errors.New("rpc-aggregator.enable and rest-aggregator.enable must be set when running writer mode")
		}
	} else {
		if f.config.RPCAggregator.Enable {
			return errors.New("rpc-aggregator is only for writer mode")
		}
		if !f.config.RestAggregator.Enable {
			return errors.New("rest-aggregator.enable must be set for reader mode")
		}
	}

	return nil
}

func (f *Factory) CreateReader(ctx context.Context) (daprovider.Reader, func(), error) {
	if f.alwaysFallback && !f.config.RestAggregator.Enable {
		return &daprovider.DangerousAlwaysFallbackReader{}, nil, nil
	}

	cfg := f.config
	if f.alwaysFallback && cfg.RPCAggregator.Enable {
		// rpc-aggregator is the writer path; with alwaysFallback the writer is suppressed.
		// Mask it so the reader-only path in CreateDAReader doesn't reject it.
		log.Warn("alwaysFallback is set; ignoring rpc-aggregator (writer-path) config")
		cfgCopy := *cfg
		cfgCopy.RPCAggregator.Enable = false
		cfg = &cfgCopy
	}

	var daReader anytrustutil.Reader
	var keysetFetcher *KeysetFetcher
	var lifecycleManager *LifecycleManager
	var err error

	if f.enableWriter {
		_, daReader, keysetFetcher, lifecycleManager, err = CreateDAReaderAndWriter(
			ctx, cfg, f.dataSigner, f.l1Client, f.seqInboxAddr)
	} else {
		daReader, keysetFetcher, lifecycleManager, err = CreateDAReader(
			ctx, cfg, f.l1Reader, &f.seqInboxAddr)
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
	if !f.enableWriter {
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

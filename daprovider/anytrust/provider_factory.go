// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/daprovider"
	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

// lint:require-exhaustive-initialization
type Factory struct {
	config       *Config
	dataSigner   signature.DataSignerFunc
	l1Client     *ethclient.Client
	l1Reader     *headerreader.HeaderReader
	seqInboxAddr common.Address
	enableWriter bool
}

// SupportedHeaderBytes are the header bytes supported by AnyTrust DA.
var SupportedHeaderBytes = []byte{
	daprovider.DASMessageHeaderFlag,
	daprovider.DASMessageHeaderFlag | daprovider.TreeDASMessageHeaderFlag,
}

// NewFactory creates a new AnyTrust DA provider factory.
func NewFactory(
	config *Config,
	dataSigner signature.DataSignerFunc,
	l1Client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddr common.Address,
	enableWriter bool,
) *Factory {
	return &Factory{
		config:       config,
		dataSigner:   dataSigner,
		l1Client:     l1Client,
		l1Reader:     l1Reader,
		seqInboxAddr: seqInboxAddr,
		enableWriter: enableWriter,
	}
}

func (f *Factory) GetSupportedHeaderBytes() []byte {
	return []byte{
		daprovider.DASMessageHeaderFlag,
		daprovider.DASMessageHeaderFlag | daprovider.TreeDASMessageHeaderFlag,
	}
}

func (f *Factory) ValidateConfig() error {
	if !f.config.Enable {
		return errors.New("anytrust data availability must be enabled")
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
	var daReader anytrustutil.Reader
	var keysetFetcher *KeysetFetcher
	var lifecycleManager *LifecycleManager
	var err error

	if f.enableWriter {
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

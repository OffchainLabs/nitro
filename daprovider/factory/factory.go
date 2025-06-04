// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package factory

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
)

type DAProviderMode string

const (
	ModeAnyTrust    DAProviderMode = "anytrust"
	ModeReferenceDA DAProviderMode = "referenceda"
)

type DAProviderFactory interface {
	CreateReader(ctx context.Context) (daprovider.Reader, func(), error)
	CreateWriter(ctx context.Context) (daprovider.Writer, func(), error)
	CreateValidator(ctx context.Context) (daprovider.Validator, func(), error)
	ValidateConfig() error
}

type AnyTrustFactory struct {
	config       *das.DataAvailabilityConfig
	dataSigner   signature.DataSignerFunc
	l1Client     *ethclient.Client
	l1Reader     *headerreader.HeaderReader
	seqInboxAddr common.Address
	enableWriter bool
}

type ReferenceDAFactory struct {
	config       *referenceda.Config
	enableWriter bool
}

func NewDAProviderFactory(
	mode DAProviderMode,
	anytrust *das.DataAvailabilityConfig,
	referencedaCfg *referenceda.Config,
	dataSigner signature.DataSignerFunc,
	l1Client *ethclient.Client,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddr common.Address,
	enableWriter bool,
) (DAProviderFactory, error) {
	switch mode {
	case ModeAnyTrust:
		return &AnyTrustFactory{
			config:       anytrust,
			dataSigner:   dataSigner,
			l1Client:     l1Client,
			l1Reader:     l1Reader,
			seqInboxAddr: seqInboxAddr,
			enableWriter: enableWriter,
		}, nil
	case ModeReferenceDA:
		factory := &ReferenceDAFactory{
			config:       referencedaCfg,
			enableWriter: enableWriter,
		}
		return factory, nil
	default:
		return nil, fmt.Errorf("unsupported DA provider mode: %s", mode)
	}
}

// AnyTrust Factory Implementation
func (f *AnyTrustFactory) ValidateConfig() error {
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

func (f *AnyTrustFactory) CreateReader(ctx context.Context) (daprovider.Reader, func(), error) {
	if f.enableWriter {
		_, daReader, keysetFetcher, lifecycleManager, err := das.CreateDAReaderAndWriter(
			ctx, f.config, f.dataSigner, f.l1Client, f.seqInboxAddr)
		if err != nil {
			return nil, nil, err
		}

		daReader = das.NewReaderTimeoutWrapper(daReader, f.config.RequestTimeout)
		if f.config.PanicOnError {
			daReader = das.NewReaderPanicWrapper(daReader)
		}

		reader := dasutil.NewReaderForDAS(daReader, keysetFetcher)
		cleanupFn := func() {
			if lifecycleManager != nil {
				lifecycleManager.StopAndWaitUntil(0)
			}
		}
		return reader, cleanupFn, nil
	} else {
		daReader, keysetFetcher, lifecycleManager, err := das.CreateDAReader(
			ctx, f.config, f.l1Reader, &f.seqInboxAddr)
		if err != nil {
			return nil, nil, err
		}

		daReader = das.NewReaderTimeoutWrapper(daReader, f.config.RequestTimeout)
		if f.config.PanicOnError {
			daReader = das.NewReaderPanicWrapper(daReader)
		}

		reader := dasutil.NewReaderForDAS(daReader, keysetFetcher)
		cleanupFn := func() {
			if lifecycleManager != nil {
				lifecycleManager.StopAndWaitUntil(0)
			}
		}
		return reader, cleanupFn, nil
	}
}

func (f *AnyTrustFactory) CreateWriter(ctx context.Context) (daprovider.Writer, func(), error) {
	if !f.enableWriter {
		return nil, nil, nil
	}

	daWriter, _, _, lifecycleManager, err := das.CreateDAReaderAndWriter(
		ctx, f.config, f.dataSigner, f.l1Client, f.seqInboxAddr)
	if err != nil {
		return nil, nil, err
	}

	if f.config.PanicOnError {
		daWriter = das.NewWriterPanicWrapper(daWriter)
	}

	writer := dasutil.NewWriterForDAS(daWriter)
	cleanupFn := func() {
		if lifecycleManager != nil {
			lifecycleManager.StopAndWaitUntil(0)
		}
	}
	return writer, cleanupFn, nil
}

func (f *AnyTrustFactory) CreateValidator(ctx context.Context) (daprovider.Validator, func(), error) {
	// AnyTrust doesn't use the Validator interface
	return nil, nil, nil
}

// ReferenceDA Factory Implementation
func (f *ReferenceDAFactory) ValidateConfig() error {
	if !f.config.Enable {
		return errors.New("referenceda must be enabled")
	}
	return nil
}

func (f *ReferenceDAFactory) initializeComponents() error {
	return nil
}

func (f *ReferenceDAFactory) CreateReader(ctx context.Context) (daprovider.Reader, func(), error) {
	reader := referenceda.NewReader()
	return reader, nil, nil
}

func (f *ReferenceDAFactory) CreateWriter(ctx context.Context) (daprovider.Writer, func(), error) {
	if !f.enableWriter {
		return nil, nil, nil
	}

	writer := referenceda.NewWriter()
	return writer, nil, nil
}

func (f *ReferenceDAFactory) CreateValidator(ctx context.Context) (daprovider.Validator, func(), error) {
	return referenceda.NewValidator(), nil, nil
}

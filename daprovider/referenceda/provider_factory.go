// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/signature"
)

// lint:require-exhaustive-initialization
type Factory struct {
	config       *Config
	enableWriter bool
	dataSigner   signature.DataSignerFunc
	l1Client     *ethclient.Client
}

// NewFactory creates a new ReferenceDA provider factory.
func NewFactory(
	config *Config,
	dataSigner signature.DataSignerFunc,
	l1Client *ethclient.Client,
	enableWriter bool,
) *Factory {
	return &Factory{
		config:       config,
		enableWriter: enableWriter,
		dataSigner:   dataSigner,
		l1Client:     l1Client,
	}
}

func (f *Factory) ValidateConfig() error {
	if !f.config.Enable {
		return errors.New("referenceda must be enabled")
	}
	return nil
}

func (f *Factory) CreateReader(ctx context.Context) (daprovider.Reader, func(), error) {
	if f.config.ValidatorContract == "" {
		return nil, nil, errors.New("validator-contract address not configured for reference DA reader")
	}
	validatorAddr := common.HexToAddress(f.config.ValidatorContract)
	storage := GetInMemoryStorage()
	reader := NewReader(storage, f.l1Client, validatorAddr)
	return reader, nil, nil
}

func (f *Factory) CreateWriter(ctx context.Context) (daprovider.Writer, func(), error) {
	if !f.enableWriter {
		return nil, nil, nil
	}

	if f.dataSigner == nil {
		var signer signature.DataSignerFunc
		if f.config.SigningKey.PrivateKey != "" {
			privKey, err := crypto.HexToECDSA(f.config.SigningKey.PrivateKey)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid private key: %w", err)
			}
			signer = signature.DataSignerFromPrivateKey(privKey)
		} else if f.config.SigningKey.KeyFile != "" {
			keyData, err := os.ReadFile(f.config.SigningKey.KeyFile)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read key file: %w", err)
			}
			privKey, err := crypto.HexToECDSA(strings.TrimSpace(string(keyData)))
			if err != nil {
				return nil, nil, fmt.Errorf("invalid private key in file: %w", err)
			}
			signer = signature.DataSignerFromPrivateKey(privKey)
		} else {
			return nil, nil, errors.New("no signing key configured for reference DA writer")
		}
		f.dataSigner = signer
	}

	writer := NewWriter(f.dataSigner, f.config.MaxBatchSize)
	return writer, nil, nil
}

func (f *Factory) CreateValidator(ctx context.Context) (daprovider.Validator, func(), error) {
	if f.config.ValidatorContract == "" {
		return nil, nil, errors.New("validator-contract address not configured for reference DA validator")
	}
	validatorAddr := common.HexToAddress(f.config.ValidatorContract)
	return NewValidator(f.l1Client, validatorAddr), nil, nil
}

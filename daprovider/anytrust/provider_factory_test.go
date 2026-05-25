// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/daprovider"
)

func newTestFactoryConfig() *Config {
	cfg := DefaultConfigForNode
	cfg.Enable = true
	return &cfg
}

func newTestFactory(t *testing.T, cfg *Config, mode FactoryMode) (*Factory, error) {
	t.Helper()
	return NewFactory(
		cfg,
		nil,
		nil,
		nil,
		common.Address{},
		mode,
	)
}

func TestFactory_NewFactory_Retiring_AcceptsMissingRestAggregator(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = false
	cfg.RPCAggregator.Enable = false
	if _, err := newTestFactory(t, cfg, ModeRetiring); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFactory_NewFactory_Retiring_RejectsLingeringRPCAggregator(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = false
	cfg.RPCAggregator.Enable = true
	_, err := newTestFactory(t, cfg, ModeRetiring)
	if err == nil || !strings.Contains(err.Error(), "rpc-aggregator.enable=true") {
		t.Fatalf("expected rejection of rpc-aggregator in retiring mode, got %v", err)
	}
}

func TestFactory_NewFactory_Reader_RejectsLingeringRPCAggregator(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = true
	cfg.RPCAggregator.Enable = true
	_, err := newTestFactory(t, cfg, ModeReader)
	if err == nil || !strings.Contains(err.Error(), "rpc-aggregator is only for writer mode") {
		t.Fatalf("expected reader-mode rejection of rpc-aggregator, got %v", err)
	}
}

func TestFactory_CreateReader_Retiring_NoCommitteeReader_ReturnsDangerousReader(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = false
	f, err := newTestFactory(t, cfg, ModeRetiring)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	reader, cleanup, err := f.CreateReader(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cleanup != nil {
		t.Fatal("expected nil cleanup function for dangerous fallback reader")
	}
	if _, ok := reader.(*daprovider.DangerousAlwaysFallbackReader); !ok {
		t.Fatalf("expected *daprovider.DangerousAlwaysFallbackReader, got %T", reader)
	}
}

func TestFactory_CreateWriter_Retiring_ReturnsNil(t *testing.T) {
	cfg := newTestFactoryConfig()
	f, err := newTestFactory(t, cfg, ModeRetiring)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	writer, cleanup, err := f.CreateWriter(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if writer != nil {
		t.Fatalf("expected nil writer in retiring mode, got %T", writer)
	}
	if cleanup != nil {
		t.Fatal("expected nil cleanup")
	}
}

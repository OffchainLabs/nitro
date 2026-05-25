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

func newTestFactory(t *testing.T, cfg *Config, enableWriter bool, alwaysFallback bool) *Factory {
	t.Helper()
	return NewFactory(
		cfg,
		nil, // dataSigner — unused for ValidateConfig and the dangerous-fallback CreateReader path
		nil, // l1Client
		nil, // l1Reader
		common.Address{},
		enableWriter,
		alwaysFallback,
	)
}

func TestFactory_ValidateConfig_AlwaysFallback_RejectsWriterMode(t *testing.T) {
	cfg := newTestFactoryConfig()
	f := newTestFactory(t, cfg, true, true)
	err := f.ValidateConfig()
	if err == nil {
		t.Fatal("expected error when alwaysFallback=true and enableWriter=true")
	}
	if !strings.Contains(err.Error(), "always-fallback-to-parent-chain-da requires enableWriter=false") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestFactory_ValidateConfig_AlwaysFallback_ReaderOnly_AcceptsMissingRestAggregator(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = false
	cfg.RPCAggregator.Enable = false
	f := newTestFactory(t, cfg, false, true)
	if err := f.ValidateConfig(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFactory_ValidateConfig_AlwaysFallback_ReaderOnly_AcceptsLingeringRPCAggregator(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = false
	cfg.RPCAggregator.Enable = true
	f := newTestFactory(t, cfg, false, true)
	if err := f.ValidateConfig(); err != nil {
		t.Fatalf("expected nil despite lingering rpc-aggregator config, got %v", err)
	}
}

func TestFactory_ValidateConfig_NormalReaderOnly_RejectsLingeringRPCAggregator(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = true
	cfg.RPCAggregator.Enable = true
	f := newTestFactory(t, cfg, false, false)
	err := f.ValidateConfig()
	if err == nil || !strings.Contains(err.Error(), "rpc-aggregator is only for writer mode") {
		t.Fatalf("expected reader-mode rejection of rpc-aggregator, got %v", err)
	}
}

func TestFactory_CreateReader_AlwaysFallback_NoCommitteeReader_ReturnsDangerousReader(t *testing.T) {
	cfg := newTestFactoryConfig()
	cfg.RestAggregator.Enable = false
	f := newTestFactory(t, cfg, false, true)
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

func TestFactory_CreateWriter_AlwaysFallback_ReturnsNil(t *testing.T) {
	cfg := newTestFactoryConfig()
	f := newTestFactory(t, cfg, false, true)
	writer, cleanup, err := f.CreateWriter(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if writer != nil {
		t.Fatalf("expected nil writer when enableWriter=false, got %T", writer)
	}
	if cleanup != nil {
		t.Fatal("expected nil cleanup")
	}
}

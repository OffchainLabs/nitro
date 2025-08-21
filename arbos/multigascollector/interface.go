// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package multigascollector

import (
	"context"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

// TransactionMultiGas represents gas data for a single transaction
type TransactionMultiGas struct {
	TxHash   []byte
	TxIndex  uint32
	MultiGas multigas.MultiGas
}

// BlockInfo represents information about a block
type BlockInfo struct {
	BlockNumber    uint64
	BlockHash      []byte
	BlockTimestamp uint64
}

// CollectorConfig holds configuration parameters for a MultiGas collector.
// If OutputDir is empty, collection is disabled.
type CollectorConfig struct {
	OutputDir      string `koanf:"output-dir"`
	BatchSize      int    `koanf:"batch-size"`
	ClearOutputDir bool   `koanf:"clear-output-dir"`
}

// DefaultCollectorConfig provides defaults for the collector.
var DefaultCollectorConfig = CollectorConfig{
	OutputDir:      "",
	BatchSize:      2000,
	ClearOutputDir: true,
}

// Collector defines the interface for collecting transaction- and block-level
// multi-dimensional gas usage data.
type Collector interface {
	// Start begins background processing. Must be called exactly once per instance.
	Start(ctx context.Context)

	// StopAndWait cancels processing, flushes any buffered data, and blocks
	// until the collector has fully shut down.
	StopAndWait()

	// PrepareToCollectBlock signals the beginning of a new block.
	// Any unfinalised transactions buffered prior to this call are discarded.
	PrepareToCollectBlock()

	// CollectTransactionMultiGas records multi-gas data for a single transaction
	// within the current block.
	CollectTransactionMultiGas(tx TransactionMultiGas)

	// FinaliseBlock completes the current block with its metadata, attaches any
	// buffered transactions, and appends it to the block buffer. Finalised blocks
	// are persisted once the batch is complete.
	FinaliseBlock(info BlockInfo)
}

// CollectorFactory defines a factory function that instantiates a Collector
// based on the provided configuration.
type CollectorFactory func(cfg CollectorConfig) (Collector, error)

// Copyright 2022, Offchain Labs, Inc.
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
	// Start begins background processing. Should be called once per instance.
	Start(ctx context.Context)

	// StopAndWait cancels processing, flushes any buffered data, and blocks
	// until the collector has fully shut down.
	StopAndWait()

	// StartBlock signals the beginning of a new block.
	StartBlock(blockNum uint64)

	// AddTransaction records multi-gas data for a transaction within the current block.
	AddTransaction(tx TransactionMultiGas)

	// FinaliseBlock finalises the current block with metadata and flushes
	// any buffered transaction data into it.
	FinaliseBlock(info BlockInfo)
}

// CollectorFactory defines a factory function that instantiates a Collector
// based on the provided configuration.
type CollectorFactory func(cfg CollectorConfig) (Collector, error)

// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/multigascollector"
	"github.com/offchainlabs/nitro/arbos/multigascollector/proto"
)

// TestMultigasCollector_System spins up an L2 node with the multigas collector enabled,
// sends a couple of transactions, and validates the on-disk protobuf batches.
func TestMultigasCollectorFromNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outDir := t.TempDir()

	// Build a node with collector enabled
	builder := NewNodeBuilder(ctx).
		DefaultConfig(t, false).
		WithMultigasCollector(multigascollector.CollectorConfig{
			OutputDir:      outDir,
			BatchSize:      5, // small to force multiple batches for 20 txs
			ClearOutputDir: true,
		})
	cleanup := builder.Build(t)

	// Generate a L2 user and send 20 transacrtions
	builder.L2Info.GenerateAccount("Alice")
	want := make(map[common.Hash]struct{})
	for i := 0; i < 20; i++ {
		// unique value to avoid duplicate txs
		value := big.NewInt(1e12 + int64(i))

		tx := builder.L2Info.PrepareTx(
			"Owner", "Alice",
			builder.L2Info.TransferGas,
			value,
			nil,
		)
		require.NoError(t, builder.L2.Client.SendTransaction(ctx, tx))
		_, err := builder.L2.EnsureTxSucceeded(tx)
		require.NoError(t, err)

		want[tx.Hash()] = struct{}{}
	}

	// Stop the node; collector.StopAndWait() is called under cleanup()
	cleanup()

	// Read all batches
	files, err := filepath.Glob(filepath.Join(outDir, "multigas_batch_*.pb"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 3, "expected at least 3 multigas batch files, got %d", len(files))

	// sort by filename to get natural block order
	sort.Strings(files)

	var blocks []*proto.BlockMultiGasData
	var lastBlockNumber uint64
	for _, f := range files {
		// Parse <start> and <end> from filename "multigas_batch_%010d_%010d.pb"
		base := filepath.Base(f)
		var start, end uint64
		_, err := fmt.Sscanf(base, "multigas_batch_%010d_%010d.pb", &start, &end)
		require.NoErrorf(t, err, "failed to parse batch filename %q", base)

		raw, err := os.ReadFile(f)
		require.NoError(t, err, "reading %s", f)
		var batch proto.BlockMultiGasBatch
		require.NoError(t, protobuf.Unmarshal(raw, &batch), "unmarshal %s", f)
		blocks = append(blocks, batch.Data...)

		// Check block numbers are strictly increasing inside the batch
		for i := 1; i < len(batch.Data); i++ {
			prev := batch.Data[i-1].BlockNumber
			cur := batch.Data[i].BlockNumber
			if !(cur > prev) {
				t.Fatalf("block numbers not strictly increasing in %s: %d -> %d at idx %d",
					base, prev, cur, i)
			}
		}

		// Check block range matches the file description
		if end > start {
			require.Equal(t, start, batch.Data[0].BlockNumber, "start block number mismatch in %s", f)
			require.Equal(t, end, batch.Data[len(batch.Data)-1].BlockNumber, "end block number mismatch in %s", f)
		}

		// Check block order across batches (files)
		if lastBlockNumber+1 != start {
			t.Fatal(fmt.Sprintf("block numbers misordered, want %d < %d", lastBlockNumber, start))
		}
		lastBlockNumber = end
	}

	// Scan for our tx hashes
	found := 0
	for _, b := range blocks {
		for _, ptx := range b.Transactions {
			h := common.BytesToHash(ptx.TxHash)
			if _, ok := want[h]; !ok {
				continue
			}
			require.NotNil(t, ptx.GetMultiGas(), "missing multigas for tx %s", h)
			found++
		}
	}

	require.Equal(t, len(want), found, "not all 20 sent txs were found in multigas batches")
}

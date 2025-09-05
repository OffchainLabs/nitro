// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	protobuf "google.golang.org/protobuf/proto"

	"github.com/offchainlabs/nitro/arbos/multigascollector/proto"
)

// readCollectorBatches scans <dir> for "multigas_batch_*.pb" files, verifies
// in-batch monotonicity and cross-batch continuity, and returns all
// BlockMultiGasData entries flattened.
// requiredFiles semantics:
//
//	-1 => require at least one batch file
//	 0 => require zero files and return an empty slice
//	>0 => require exact number of files
func readCollectorBatches(t *testing.T, dir string, requiredFiles int) []*proto.BlockMultiGasData {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "multigas_batch_*.pb"))
	require.NoError(t, err)

	switch {
	case requiredFiles == -1:
		require.NotEmptyf(t, files, "no multigas batch files found in %s", dir)
	case requiredFiles == 0:
		require.Emptyf(t, files, "expected zero multigas batch files in %s, got %d", dir, len(files))
		return nil
	case requiredFiles > 0:
		require.Equalf(t, requiredFiles, len(files),
			"expected %d multigas batch files in %s, got %d", requiredFiles, dir, len(files))
	}

	//  sort by filename to get natural block order
	sort.Strings(files)

	var blocks []*proto.BlockMultiGasData
	var lastEnd uint64

	for i, f := range files {
		// Parse <start> and <end> from filename "multigas_batch_%010d_%010d.pb"
		base := filepath.Base(f)
		var start, end uint64
		_, err := fmt.Sscanf(base, "multigas_batch_%010d_%010d.pb", &start, &end)
		require.NoErrorf(t, err, "parse filename %q", base)

		raw, err := os.ReadFile(f)
		require.NoError(t, err, "read %s", f)

		var batch proto.BlockMultiGasBatch
		require.NoError(t, protobuf.Unmarshal(raw, &batch), "unmarshal %s", f)

		// Check block numbers are strictly increasing inside the batch
		for j := 1; j < len(batch.Data); j++ {
			require.Greater(t, batch.Data[j].BlockNumber, batch.Data[j-1].BlockNumber,
				"non-increasing block number in %s", base)
		}

		// Check filename range matches content & cross-batch continuity
		if len(batch.Data) > 0 {
			if end > start {
				require.Equal(t, start, batch.Data[0].BlockNumber, "start mismatch in %s", base)
				require.Equal(t, end, batch.Data[len(batch.Data)-1].BlockNumber, "end mismatch in %s", base)
			}
			if i > 0 {
				require.Equal(t, lastEnd+1, batch.Data[0].BlockNumber,
					"block numbers misordered across batches: want %d, got %d",
					lastEnd+1, batch.Data[0].BlockNumber)
			}
			lastEnd = batch.Data[len(batch.Data)-1].BlockNumber
		}
		blocks = append(blocks, batch.Data...)
	}
	return blocks
}

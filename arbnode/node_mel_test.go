// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbnode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/db/schema"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
)

func putRLPValue(t *testing.T, db interface{ Put([]byte, []byte) error }, key []byte, val uint64) {
	t.Helper()
	data, err := rlp.EncodeToBytes(val)
	require.NoError(t, err)
	require.NoError(t, db.Put(key, data))
}

func TestValidateAndInitializeDBForMEL_StateAlreadyExists(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	melDB, err := melrunner.NewDatabase(db)
	require.NoError(t, err)
	require.NoError(t, melDB.SaveState(&mel.State{
		ParentChainBlockNumber: 10,
		ParentChainBlockHash:   common.HexToHash("0x1234"),
	}))

	_, err = validateAndInitializeDBForMEL(context.Background(), nil, nil, db, false)
	require.NoError(t, err)
}

func TestValidateAndInitializeDBForMEL_LegacyKeysWithExistingState(t *testing.T) {
	t.Parallel()
	// If MEL state already exists, legacy keys are irrelevant — should succeed.
	db := rawdb.NewMemoryDatabase()
	melDB, err := melrunner.NewDatabase(db)
	require.NoError(t, err)
	require.NoError(t, melDB.SaveState(&mel.State{
		ParentChainBlockNumber: 10,
		ParentChainBlockHash:   common.HexToHash("0x1234"),
	}))
	putRLPValue(t, db, schema.SequencerBatchCountKey, 5)
	putRLPValue(t, db, schema.DelayedMessageCountKey, 3)

	_, err = validateAndInitializeDBForMEL(context.Background(), nil, nil, db, false)
	require.NoError(t, err)
}

func TestValidateAndInitializeDBForMEL_NonZeroMessageCount(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	putRLPValue(t, db, schema.MessageCountKey, 5)

	_, err := validateAndInitializeDBForMEL(context.Background(), nil, nil, db, false)
	require.ErrorContains(t, err, "stale msgs")
}

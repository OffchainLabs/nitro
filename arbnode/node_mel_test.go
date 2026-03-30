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
	melDB := melrunner.NewDatabase(db)
	require.NoError(t, melDB.SaveState(&mel.State{
		ParentChainBlockNumber: 10,
		ParentChainBlockHash:   common.HexToHash("0x1234"),
	}))

	err := validateAndInitializeDBForMEL(context.Background(), nil, nil, db)
	require.NoError(t, err)
}

func TestValidateAndInitializeDBForMEL_StaleSequencerBatchCountKey(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	putRLPValue(t, db, schema.MessageCountKey, 0)
	require.NoError(t, db.Put(schema.SequencerBatchCountKey, []byte("stale")))

	err := validateAndInitializeDBForMEL(context.Background(), nil, nil, db)
	require.ErrorContains(t, err, "stale SequencerBatchCountKey from inbox reader")
}

func TestValidateAndInitializeDBForMEL_StaleDelayedMessageCountKey(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	putRLPValue(t, db, schema.MessageCountKey, 0)
	require.NoError(t, db.Put(schema.DelayedMessageCountKey, []byte("stale")))

	err := validateAndInitializeDBForMEL(context.Background(), nil, nil, db)
	require.ErrorContains(t, err, "stale DelayedMessageCountKey from inbox reader")
}

func TestValidateAndInitializeDBForMEL_NonZeroMessageCount(t *testing.T) {
	t.Parallel()
	db := rawdb.NewMemoryDatabase()
	putRLPValue(t, db, schema.MessageCountKey, 5)

	err := validateAndInitializeDBForMEL(context.Background(), nil, nil, db)
	require.ErrorContains(t, err, "stale msgs")
}

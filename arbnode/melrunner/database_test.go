package melrunner

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/arbnode/mel"
)

func TestMelDatabase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create database
	arbDb := rawdb.NewMemoryDatabase()
	melDb := NewDatabase(arbDb)

	headMelState := &mel.State{
		ParentChainBlockNumber: 2,
		ParentChainBlockHash:   common.MaxHash,
	}
	require.NoError(t, melDb.SaveState(ctx, headMelState))

	headMelStateBlockNum, err := melDb.GetHeadMelStateBlockNum()
	require.NoError(t, err)
	require.True(t, headMelStateBlockNum == headMelState.ParentChainBlockNumber)

	var melState *mel.State
	checkMelState := func() {
		require.NoError(t, err)
		if !reflect.DeepEqual(melState, headMelState) {
			t.Fatal("unexpected melState retrieved via GetState using parentChainBlockHash")
		}
	}
	melState, err = melDb.State(ctx, headMelState.ParentChainBlockNumber)
	checkMelState()

}

func TestMelDatabaseReadAndWriteDelayedMessages(t *testing.T) {
	t.Skip("to be implemented")
}

func TestMelDelayedMessagesAccumulation(t *testing.T) {
	t.Skip("to be implemented")
}

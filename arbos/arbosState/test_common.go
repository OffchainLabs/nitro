package arbosState

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"testing"
)

// Create a memory-backed ArbOS state
func OpenArbosStateForTesting(t *testing.T) *ArbosState {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		t.Fatal("failed to init empty statedb")
	}
	return OpenArbosState(statedb)
}

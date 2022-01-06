package arbosState

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/util/testhelpers"
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

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

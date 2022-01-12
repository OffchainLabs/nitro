package arbosState

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/offchainlabs/arbstate/arbos/burn"
	"github.com/offchainlabs/arbstate/util/testhelpers"
)

// Create a memory-backed ArbOS state
func OpenArbosStateForTesting(t *testing.T) *ArbosState {
	raw := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(raw)
	statedb, err := state.New(common.Hash{}, db, nil)
	Require(t, err, "failed to init empty statedb")
	state, err := OpenArbosState(statedb, &burn.SystemBurner{})
	Require(t, err, "failed to open the ArbOS state")
	return state
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

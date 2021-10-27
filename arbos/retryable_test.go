package arbos

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestOpenNonexistentRetryable(t *testing.T) {
	state := OpenArbosStateForTest(t)
	id := common.BigToHash(big.NewInt(978645611142))
	retryable := state.RetryableState().OpenRetryable(id, state.LastTimestampSeen())
	if retryable != nil {
		t.Fatal()
	}
}

func TestOpenExpiredRetryable(t *testing.T) {
	state := OpenArbosStateForTest(t)
	originalTimestamp := state.LastTimestampSeen()
	newTimestamp := originalTimestamp + 42
	state.SetLastTimestampSeen(newTimestamp)

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := originalTimestamp // in the past
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	calldata := []byte{42}
	_ = state.RetryableState().CreateRetryable(state.LastTimestampSeen(), id, timeout, from, to, callvalue, calldata)

	reread := state.RetryableState().OpenRetryable(id, state.LastTimestampSeen())
	if reread != nil {
		t.Fatal()
	}
}

func TestRetryableCreate(t *testing.T) {
	state := OpenArbosStateForTest(t)
	id := common.BigToHash(big.NewInt(978645611142))
	timeout := state.LastTimestampSeen() + 10000000
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	calldata := make([]byte, 42)
	for i := range calldata {
		calldata[i] = byte(i + 3)
	}
	rstate := state.RetryableState()
	retryable := rstate.CreateRetryable(state.LastTimestampSeen(), id, timeout, from, to, callvalue, calldata)

	reread := rstate.OpenRetryable(id, state.LastTimestampSeen())
	if reread == nil {
		t.Fatal()
	}
	if !reread.Equals(retryable) {
		t.Fatal()
	}
}

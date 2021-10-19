package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestOpenNonexistentRetryable(t *testing.T) {
	state := OpenArbosStateForTest()
	id := common.BigToHash(big.NewInt(978645611142))
	retryable := OpenRetryable(state, id)
	if retryable != nil {
		t.Fatal()
	}
}

func TestOpenExpiredRetryable(t *testing.T) {
	state := OpenArbosStateForTest()
	originalTimestamp := state.LastTimestampSeen()
	newTimestamp := originalTimestamp + 42
	state.SetLastTimestampSeen(newTimestamp)

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := originalTimestamp // in the past
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	calldata := []byte{42}
	_ = CreateRetryable(state, id, timeout, from, to, callvalue, calldata)

	reread := OpenRetryable(state, id)
	if reread != nil {
		t.Fatal()
	}
}

func TestRetryableCreate(t *testing.T) {
	state := OpenArbosStateForTest()
	id := common.BigToHash(big.NewInt(978645611142))
	timeout := state.LastTimestampSeen() + 10000000
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	calldata := []byte{42}
	retryable := CreateRetryable(state, id, timeout, from, to, callvalue, calldata)

	reread := OpenRetryable(state, id)
	if reread == nil {
		t.Fatal()
	}
	if reread.id != retryable.id {
		t.Fatal()
	}
	if reread.timeout != retryable.timeout {
		t.Fatal()
	}
	if reread.from != retryable.from {
		t.Fatal()
	}
	if reread.to != retryable.to {
		t.Fatal()
	}
	if reread.callvalue.Cmp(retryable.callvalue) != 0 {
		t.Fatal()
	}
	if !bytes.Equal(reread.calldata, retryable.calldata) {
		t.Fatal()
	}
}

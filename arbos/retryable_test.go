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
	newTimestamp := new(big.Int).Add(originalTimestamp, big.NewInt(42))
	state.SetLastTimestampSeen(newTimestamp)

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := originalTimestamp.Uint64() // in the past
	from := common.BytesToAddress([]byte{ 3, 4, 5 })
	to := common.BytesToAddress([]byte{ 6, 7, 8, 9 })
	callvalue := big.NewInt(0)
	calldata := []byte{ 42 }
	_ = CreateRetryable(state, id, timeout, from, to, callvalue, calldata)

	reread := OpenRetryable(state, id)
	if reread != nil {
		t.Fatal()
	}
}


func TestRetryableCreate(t *testing.T) {
	state := OpenArbosStateForTest()
	id := common.BigToHash(big.NewInt(978645611142))
	timeout := state.LastTimestampSeen().Uint64()+10000000
	from := common.BytesToAddress([]byte{ 3, 4, 5 })
	to := common.BytesToAddress([]byte{ 6, 7, 8, 9 })
	callvalue := big.NewInt(0)
	calldata := []byte{ 42 }
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

func TestPlanOneRedeem(t *testing.T) {
	state := OpenArbosStateForTest()
	id := common.BigToHash(big.NewInt(978645611142))
	timeout := state.LastTimestampSeen().Uint64()+10000000
	from := common.BytesToAddress([]byte{3, 4, 5})
	to := common.BytesToAddress([]byte{6, 7, 8, 9})
	callvalue := big.NewInt(0)
	calldata := []byte{42}
	_ = CreateRetryable(state, id, timeout, from, to, callvalue, calldata)

	refundAddr := common.Address{ 3, 4 }
	gasFundsWei := big.NewInt(486687768)
	redeem := NewPlannedRedeem(state, id, refundAddr, gasFundsWei)

	readBack := PeekNextPlannedRedeem(state)
	if readBack == nil {
		t.Fatal()
	}
	if redeem.retryableId != readBack.retryableId {
		t.Fatal()
	}
	if redeem.gasRefundAddr != readBack.gasRefundAddr {
		t.Fatal()
	}
	if redeem.gasFundsWei.Cmp(readBack.gasFundsWei) != 0 {
		t.Fatal()
	}

	// read back again, verify we got the same thing
	readBack = PeekNextPlannedRedeem(state)
	if readBack == nil {
		t.Fatal()
	}
	if redeem.retryableId != readBack.retryableId {
		t.Fatal()
	}
	if redeem.gasRefundAddr != readBack.gasRefundAddr {
		t.Fatal()
	}
	if redeem.gasFundsWei.Cmp(readBack.gasFundsWei) != 0 {
		t.Fatal()
	}

	// discard the redeem without deleting the retryable, then make sure retryable is still there
	DiscardNextPlannedRedeem(state, false)
	if OpenRetryable(state, id) == nil {
		t.Fatal()
	}

	// make a new redeem, discard it with delete, then make sure the retryable is gone
	_ = NewPlannedRedeem(state, id, refundAddr, gasFundsWei)
	DiscardNextPlannedRedeem(state, true)
	if OpenRetryable(state, id) != nil {
		t.Fatal()
	}
}
// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package storage

import (
	"bytes"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestTimeEncoding(t *testing.T) {
	now := RlpTime(time.Now())
	enc, err := rlp.EncodeToBytes(now)
	if err != nil {
		t.Fatal("failed to encode time", err)
	}
	var dec RlpTime
	err = rlp.DecodeBytes(enc, &dec)
	if err != nil {
		t.Fatal("failed to decode time", err)
	}
	if !time.Time(dec).Equal(time.Time(now)) {
		t.Fatalf("time %v encoded then decoded to %v", now, dec)
	}
}

func TestOldTimeEncoding(t *testing.T) {
	type OldQueuedTransaction struct {
		FullTx          *types.Transaction
		Data            types.DynamicFeeTx
		Meta            []byte
		Sent            bool
		Created         time.Time
		NextReplacement time.Time
	}

	oldTx := OldQueuedTransaction{
		FullTx: types.NewTx(&types.DynamicFeeTx{}),
		Meta:   []byte{0},
	}

	enc, err := rlp.EncodeToBytes(oldTx)
	if err != nil {
		t.Fatal("failed to encode old queued tx", err)
	}
	var dec QueuedTransaction
	err = rlp.DecodeBytes(enc, &dec)
	if err != nil {
		t.Fatal("failed to decode old queued tx", err)
	}

	if !bytes.Equal(oldTx.Meta, dec.Meta) {
		t.Fatalf("meta %v encoded then decoded to %v", oldTx.Meta, dec.Meta)
	}
}

func TestWeirdTimeEncoding(t *testing.T) {
	type OldQueuedTransaction struct {
		FullTx          *types.Transaction
		Data            types.DynamicFeeTx
		Meta            []byte
		Sent            bool
		Created         *RlpTime `rlp:"optional"`
		NextReplacement *RlpTime `rlp:"optional"`
	}

	now := time.Now()
	oldTx := OldQueuedTransaction{
		FullTx:          types.NewTx(&types.DynamicFeeTx{}),
		Meta:            []byte{0},
		Created:         (*RlpTime)(&now),
		NextReplacement: (*RlpTime)(&now),
	}

	enc, err := rlp.EncodeToBytes(oldTx)
	if err != nil {
		t.Fatal("failed to encode old queued tx", err)
	}
	var dec QueuedTransaction
	err = rlp.DecodeBytes(enc, &dec)
	if err != nil {
		t.Fatal("failed to decode old queued tx", err)
	}

	if !bytes.Equal(oldTx.Meta, dec.Meta) {
		t.Fatalf("meta %v encoded then decoded to %v", oldTx.Meta, dec.Meta)
	}
	if !(time.Time)(*oldTx.Created).Equal(dec.Created) {
		t.Fatalf("created %v encoded then decoded to %v", oldTx.Created, dec.Created)
	}
}

func TestNewQueuedTransactionEncoding(t *testing.T) {
	oldTx := &QueuedTransaction{
		FullTx:  types.NewTx(&types.DynamicFeeTx{}),
		Meta:    []byte{0},
		Created: time.Now(),
	}

	enc, err := rlp.EncodeToBytes(oldTx)
	if err != nil {
		t.Fatal("failed to encode old queued tx", err)
	}
	var dec QueuedTransaction
	err = rlp.DecodeBytes(enc, &dec)
	if err != nil {
		t.Fatal("failed to decode old queued tx", err)
	}

	if !bytes.Equal(oldTx.Meta, dec.Meta) {
		t.Fatalf("meta %v encoded then decoded to %v", oldTx.Meta, dec.Meta)
	}
	if !oldTx.Created.Equal(dec.Created) {
		t.Fatalf("created %v encoded then decoded to %v", oldTx.Created, dec.Created)
	}
}

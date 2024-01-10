// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package storage

import (
	"io"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
)

// time.Time doesn't encode as anything in RLP. This fixes that.
// It encodes a timestamp as a uint64 unix timestamp in seconds,
// so any subsecond precision is lost.
type RlpTime time.Time

type rlpTimeEncoding struct {
	Seconds uint64
	Nanos   uint64
}

func (b *RlpTime) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == rlp.List && size == 0 {
		// This is an old time.Time without any data
		return s.Decode(&time.Time{})
	}
	var enc rlpTimeEncoding
	err = s.Decode(&enc)
	if err != nil {
		return err
	}
	*b = RlpTime(time.Unix(int64(enc.Seconds), int64(enc.Nanos)))
	return nil
}

func (b RlpTime) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rlpTimeEncoding{
		Seconds: uint64(time.Time(b).Unix()),
		Nanos:   uint64(time.Time(b).Nanosecond()),
	})
}

func (b RlpTime) String() string {
	return time.Time(b).String()
}

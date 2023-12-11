// Copyright 2021-2023, Offchain Labs, Inc.
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

func (b *RlpTime) DecodeRLP(s *rlp.Stream) error {
	var nanos uint64
	err := s.Decode(&nanos)
	if err != nil {
		return err
	}
	*b = RlpTime(time.Unix(int64(nanos), 0))
	return nil
}

func (b RlpTime) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, uint64(time.Time(b).Unix()))
}

func (b RlpTime) String() string {
	return time.Time(b).String()
}

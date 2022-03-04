//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"context"
	"encoding/binary"
	"reflect"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type DataAvailabilityServiceWriter interface {
	Store(ctx context.Context, message []byte) (*arbstate.DataAvailabilityCertificate, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityServiceReader
	DataAvailabilityServiceWriter
}

type DataAvailabilityMode uint64

const (
	OnchainDataAvailability DataAvailabilityMode = iota
	LocalDataAvailability
)

type DataAvailabilityConfig struct {
	LocalDiskDataDir string
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{}

func serializeSignableFields(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 32+8+8)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)
	return buf
}

func Serialize(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 1+reflect.TypeOf(arbstate.DataAvailabilityCertificate{}).Size())

	buf = append(buf, arbstate.DASMessageHeaderFlag)

	buf = append(buf, serializeSignableFields(c)...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}

//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"context"
	"encoding/binary"
	"errors"
	"reflect"

	"github.com/offchainlabs/arbstate/blsSignatures"
)

type DataAvailabilityServiceReader interface {
	Retrieve(ctx context.Context, hash []byte) ([]byte, error)
}

const DASMessageHeaderFlag byte = 0x80

func IsDASMessageHeaderByte(header byte) bool {
	return (DASMessageHeaderFlag & header) > 0
}

type DataAvailabilityCertificate struct {
	DataHash    [32]byte
	Timeout     uint64
	SignersMask uint64
	Sig         blsSignatures.Signature
}

func DeserializeDASCertFrom(buf []byte) (c *DataAvailabilityCertificate, bytesRead int, err error) {
	if buf[0] != DASMessageHeaderFlag {
		panic("Didn't check DAS certificate header before deserializing")
	}
	bytesRead += 1

	c = &DataAvailabilityCertificate{}
	certSize := 1 + reflect.TypeOf(*c).Size()
	if uintptr(len(buf)) < certSize {
		return nil, 0, errors.New("Can't deserialize DAS cert from smaller buffer")
	}

	bytesRead += copy(c.DataHash[:], buf[bytesRead:bytesRead+32])

	c.Timeout = binary.BigEndian.Uint64(buf[bytesRead : bytesRead+8])
	bytesRead += 8

	c.SignersMask = binary.BigEndian.Uint64(buf[bytesRead : bytesRead+8])
	bytesRead += 8

	c.Sig, err = blsSignatures.SignatureFromBytes(buf[bytesRead : bytesRead+96])
	if err != nil {
		return nil, 0, err
	}
	bytesRead += 96
	return c, bytesRead, nil
}

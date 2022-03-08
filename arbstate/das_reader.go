//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/offchainlabs/nitro/blsSignatures"
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
	c = &DataAvailabilityCertificate{}
	if uintptr(len(buf)) < 1+32+8+8+96 {
		return nil, 0, errors.New("Can't deserialize DAS cert from smaller buffer")
	}
	if !IsDASMessageHeaderByte(buf[0]) {
		return nil, 0, errors.New("Tried to deserialize a message that doesn't have the DAS header.")
	}
	bytesRead += 1

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

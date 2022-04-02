//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/offchainlabs/nitro/blsSignatures"
)

type DataAvailabilityServiceReader interface {
	Retrieve(ctx context.Context, cert []byte) ([]byte, error)
}

// Indicates that this data is a certificate for the data availability service,
// which will retrieve the full batch data.
const DASMessageHeaderFlag byte = 0x80

// Indicates that this message was authenticated by L1. Currently unused.
const L1AuthenticatedMessageHeaderFlag byte = 0x40

// Indicates that this message is zeroheavy-encoded.
const ZeroheavyMessageHeaderFlag byte = 0x20

func IsDASMessageHeaderByte(header byte) bool {
	return (DASMessageHeaderFlag & header) > 0
}

func IsZeroheavyEncodedHeaderByte(header byte) bool {
	return (ZeroheavyMessageHeaderFlag & header) > 0
}

type DataAvailabilityCertificate struct {
	DataHash    [32]byte
	Timeout     uint64
	SignersMask uint64
	Sig         blsSignatures.Signature
}

func DeserializeDASCertFrom(buf []byte) (c *DataAvailabilityCertificate, bytesRead int, err error) {
	c = &DataAvailabilityCertificate{}
	expectedCertSize := uintptr(1 + 32 + 8 + 8 + 96)
	if uintptr(len(buf)) < expectedCertSize {
		return nil, 0, fmt.Errorf("Can't deserialize DAS cert from smaller buffer (was %dB but should be %d)", uintptr(len(buf)), expectedCertSize)
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

// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbstate

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

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

func DeserializeDASCertFrom(rd io.Reader) (c *DataAvailabilityCertificate, err error) {
	r := bufio.NewReader(rd)
	c = &DataAvailabilityCertificate{}
	expectedCertSize := 1 + 32 + 8 + 8 + 96
	if r.Size() < expectedCertSize {
		return nil, fmt.Errorf("Can't deserialize DAS cert from smaller buffer (was %dB but should be %d)", r.Size(), expectedCertSize)
	}

	header, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if !IsDASMessageHeaderByte(header) {
		return nil, errors.New("Tried to deserialize a message that doesn't have the DAS header.")
	}

	_, err = r.Read(c.DataHash[:])
	if err != nil {
		return nil, err
	}

	var timeoutBuf [8]byte
	_, err = r.Read(timeoutBuf[:])
	if err != nil {
		return nil, err
	}
	c.Timeout = binary.BigEndian.Uint64(timeoutBuf[:])

	var signersMaskBuf [8]byte
	_, err = r.Read(signersMaskBuf[:])
	if err != nil {
		return nil, err
	}
	c.SignersMask = binary.BigEndian.Uint64(signersMaskBuf[:])

	var blsSignaturesBuf [96]byte
	_, err = r.Read(blsSignaturesBuf[:])
	if err != nil {
		return nil, err
	}
	c.Sig, err = blsSignatures.SignatureFromBytes(blsSignaturesBuf[:])
	if err != nil {
		return nil, err
	}

	return c, nil
}

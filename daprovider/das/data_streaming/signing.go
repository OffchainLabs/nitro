package data_streaming

import (
	"context"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/signature"
)

type PayloadSigner interface {
	SignPayload(bytes []byte, extras ...uint64) ([]byte, error)
}

type PayloadVerifier interface {
	VerifyPayload(ctx context.Context, signature []byte, bytes []byte, extras ...uint64) error
}

type DefaultPayloadSigner struct {
	inner signature.DataSignerFunc
}

func (s *DefaultPayloadSigner) SignPayload(bytes []byte, extras ...uint64) ([]byte, error) {
	return s.inner(crypto.Keccak256(flattenDataForSigning(bytes, extras...)))
}

type DefaultPayloadVerifier struct {
	inner *signature.Verifier
}

func (v *DefaultPayloadVerifier) VerifyPayload(ctx context.Context, signature []byte, bytes []byte, extras ...uint64) error {
	expectedPayload := flattenDataForSigning(bytes, extras...)
	return v.inner.VerifyData(ctx, signature, expectedPayload)
}

func flattenDataForSigning(bytes []byte, extras ...uint64) []byte {
	var bufferForExtras []byte
	for _, field := range extras {
		bufferForExtras = binary.BigEndian.AppendUint64(bufferForExtras, field)
	}
	return arbmath.ConcatByteSlices(bytes, bufferForExtras)
}

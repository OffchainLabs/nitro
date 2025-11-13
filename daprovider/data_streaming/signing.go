package data_streaming

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/signature"
)

// lint:require-exhaustive-initialization
type PayloadSigner struct {
	signPayload func(bytes []byte, extras ...uint64) ([]byte, error)
}

func DefaultPayloadSigner(signer signature.DataSignerFunc) *PayloadSigner {
	return CustomPayloadSigner(func(bytes []byte, extras ...uint64) ([]byte, error) {
		return signer(crypto.Keccak256(flattenDataForSigning(bytes, extras...)))
	})
}

func CustomPayloadSigner(signingFunc func([]byte, ...uint64) ([]byte, error)) *PayloadSigner {
	return &PayloadSigner{
		signPayload: signingFunc,
	}
}

func PayloadCommiter() *PayloadSigner {
	return CustomPayloadSigner(func(bytes []byte, extras ...uint64) ([]byte, error) {
		return crypto.Keccak256(flattenDataForSigning(bytes, extras...)), nil
	})
}

// lint:require-exhaustive-initialization
type PayloadVerifier struct {
	verifyPayload func(ctx context.Context, signature []byte, bytes []byte, extras ...uint64) error
}

func DefaultPayloadVerifier(verifier *signature.Verifier) *PayloadVerifier {
	return CustomPayloadVerifier(func(ctx context.Context, signature []byte, bytes []byte, extras ...uint64) error {
		expectedPayload := flattenDataForSigning(bytes, extras...)
		return verifier.VerifyData(ctx, signature, expectedPayload)
	})
}

func CustomPayloadVerifier(verifyingFunc func(ctx context.Context, signature []byte, bytes []byte, extras ...uint64) error) *PayloadVerifier {
	return &PayloadVerifier{
		verifyPayload: verifyingFunc,
	}
}

func PayloadCommitmentVerifier() *PayloadVerifier {
	return CustomPayloadVerifier(func(ctx context.Context, signature []byte, data []byte, extras ...uint64) error {
		expectedCommitment := crypto.Keccak256(flattenDataForSigning(data, extras...))
		if bytes.Equal(signature, expectedCommitment) {
			return nil
		} else {
			return errors.New("signature does not match")
		}
	})
}

func NoopPayloadVerifier() *PayloadVerifier {
	return CustomPayloadVerifier(func(ctx context.Context, signature []byte, data []byte, extras ...uint64) error {
		return nil
	})
}

func flattenDataForSigning(bytes []byte, extras ...uint64) []byte {
	var bufferForExtras []byte
	for _, field := range extras {
		bufferForExtras = binary.BigEndian.AppendUint64(bufferForExtras, field)
	}
	return arbmath.ConcatByteSlices(bytes, bufferForExtras)
}

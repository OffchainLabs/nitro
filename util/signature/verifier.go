// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package signature

import (
	"context"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
)

type Verifier struct {
	requireSignature bool
	authorizedMap    map[common.Address]struct{}
	bpValidator      contracts.BatchPosterVerifierInterface
}

func NewVerifier(requireSignature bool, authorizedAddresses []common.Address, bpValidator contracts.BatchPosterVerifierInterface) *Verifier {
	authorizedMap := make(map[common.Address]struct{}, len(authorizedAddresses))
	for _, addr := range authorizedAddresses {
		authorizedMap[addr] = struct{}{}
	}
	return &Verifier{
		requireSignature: requireSignature,
		authorizedMap:    authorizedMap,
		bpValidator:      bpValidator,
	}
}

func (v *Verifier) VerifyHash(ctx context.Context, signature []byte, hash common.Hash) (bool, error) {
	return v.verifyClosure(ctx, signature, func() common.Hash { return hash })
}

func (v *Verifier) VerifyData(ctx context.Context, signature []byte, data ...[]byte) (bool, error) {
	return v.verifyClosure(ctx, signature, func() common.Hash { return crypto.Keccak256Hash(data...) })
}

var ErrMissingFeedSignature = errors.New("missing required feed signature")

func (v *Verifier) verifyClosure(ctx context.Context, signature []byte, getHash func() common.Hash) (bool, error) {
	if len(signature) == 0 {
		if !v.requireSignature {
			// Signature missing and not required
			return true, nil
		}

		return false, ErrMissingFeedSignature
	}

	var hash = getHash()

	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return false, errors.Wrap(err, "unable to recover sequencer feed signing key")
	}

	addr := crypto.PubkeyToAddress(*sigPublicKey)
	if _, exists := v.authorizedMap[addr]; exists {
		return true, nil
	}

	if v.bpValidator == nil {
		return false, nil
	}

	return v.bpValidator.IsBatchPoster(ctx, addr)
}

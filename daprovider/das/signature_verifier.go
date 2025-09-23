// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/contracts"
)

// SignatureVerifier.Store will try to verify that the passed-in data's signature
// is from the batch poster, or from an injectable verification method.
type SignatureVerifier struct {
	addrVerifier *contracts.AddressVerifier

	// Extra batch poster verifier, for local installations to have their
	// own way of testing Stores.
	extraBpVerifier func(message []byte, sig []byte, extraFields ...uint64) bool
}

func NewSignatureVerifierWithSeqInboxCaller(
	seqInboxCaller *bridgegen.SequencerInboxCaller,
	extraSignatureCheckingPublicKey string,
) (*SignatureVerifier, error) {
	var addrVerifier *contracts.AddressVerifier
	if seqInboxCaller != nil {
		addrVerifier = contracts.NewAddressVerifier(seqInboxCaller)
	}

	var extraBpVerifier func(message []byte, sig []byte, extraFeilds ...uint64) bool
	if extraSignatureCheckingPublicKey != "" {
		var pubkey []byte
		var err error
		if extraSignatureCheckingPublicKey[:2] == "0x" {
			pubkey, err = hex.DecodeString(extraSignatureCheckingPublicKey[2:])
			if err != nil {
				return nil, err
			}
		} else {
			pubkeyEncoded, err := os.ReadFile(extraSignatureCheckingPublicKey)
			if err != nil {
				return nil, err
			}
			pubkey, err = hex.DecodeString(string(pubkeyEncoded))
			if err != nil {
				return nil, err
			}
		}
		extraBpVerifier = func(message []byte, sig []byte, extraFields ...uint64) bool {
			if len(sig) >= 64 {
				return crypto.VerifySignature(pubkey, dasStoreHash(message, extraFields...), sig[:64])
			}
			return false
		}
	}

	return &SignatureVerifier{
		addrVerifier:    addrVerifier,
		extraBpVerifier: extraBpVerifier,
	}, nil

}

func (v *SignatureVerifier) verify(
	ctx context.Context, message []byte, sig []byte, extraFields ...uint64) error {
	if v.extraBpVerifier == nil && v.addrVerifier == nil {
		return errors.New("no signature verification method configured")
	}

	var verified bool
	if v.extraBpVerifier != nil {
		verified = v.extraBpVerifier(message, sig, extraFields...)
	}

	if !verified && v.addrVerifier != nil {
		actualSigner, err := DasRecoverSigner(message, sig, extraFields...)
		if err != nil {
			return err
		}
		verified, err = v.addrVerifier.IsBatchPosterOrSequencer(ctx, actualSigner)
		if err != nil {
			return err
		}
	}
	if !verified {
		return errors.New("request not properly signed")
	}
	return nil
}

func (v *SignatureVerifier) String() string {
	hasAddrVerifier := v.addrVerifier != nil
	hasExtraBpVerifier := v.extraBpVerifier != nil
	return fmt.Sprintf("SignatureVerifier{hasAddrVerifier:%v,hasExtraBpVerifier:%v}", hasAddrVerifier, hasExtraBpVerifier)
}

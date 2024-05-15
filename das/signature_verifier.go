// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/pretty"
)

// SignatureVerifier.Store will try to verify that the passed-in data's signature
// is from the batch poster, or from an injectable verification method.
type SignatureVerifier struct {
	inner DataAvailabilityServiceWriter

	addrVerifier *contracts.AddressVerifier

	// Extra batch poster verifier, for local installations to have their
	// own way of testing Stores.
	extraBpVerifier func(message []byte, timeout uint64, sig []byte) bool
}

func NewSignatureVerifier(ctx context.Context, config DataAvailabilityConfig, inner DataAvailabilityServiceWriter) (*SignatureVerifier, error) {
	if config.ParentChainNodeURL == "none" {
		return NewSignatureVerifierWithSeqInboxCaller(nil, inner, config.ExtraSignatureCheckingPublicKey)
	}
	l1client, err := GetL1Client(ctx, config.ParentChainConnectionAttempts, config.ParentChainNodeURL)
	if err != nil {
		return nil, err
	}
	seqInboxAddress, err := OptionalAddressFromString(config.SequencerInboxAddress)
	if err != nil {
		return nil, err
	}
	if seqInboxAddress == nil {
		return NewSignatureVerifierWithSeqInboxCaller(nil, inner, config.ExtraSignatureCheckingPublicKey)
	}

	seqInboxCaller, err := bridgegen.NewSequencerInboxCaller(*seqInboxAddress, l1client)
	if err != nil {
		return nil, err
	}
	return NewSignatureVerifierWithSeqInboxCaller(seqInboxCaller, inner, config.ExtraSignatureCheckingPublicKey)

}

func NewSignatureVerifierWithSeqInboxCaller(
	seqInboxCaller *bridgegen.SequencerInboxCaller,
	inner DataAvailabilityServiceWriter,
	extraSignatureCheckingPublicKey string,
) (*SignatureVerifier, error) {
	var addrVerifier *contracts.AddressVerifier
	if seqInboxCaller != nil {
		addrVerifier = contracts.NewAddressVerifier(seqInboxCaller)
	}

	var extraBpVerifier func(message []byte, timeout uint64, sig []byte) bool
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
		extraBpVerifier = func(message []byte, timeout uint64, sig []byte) bool {
			if len(sig) >= 64 {
				return crypto.VerifySignature(pubkey, dasStoreHash(message, timeout), sig[:64])
			}
			return false
		}
	}

	return &SignatureVerifier{
		inner:           inner,
		addrVerifier:    addrVerifier,
		extraBpVerifier: extraBpVerifier,
	}, nil

}

func (v *SignatureVerifier) Store(
	ctx context.Context, message []byte, timeout uint64, sig []byte,
) (c *daprovider.DataAvailabilityCertificate, err error) {
	log.Trace("das.SignatureVerifier.Store", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", v)
	var verified bool
	if v.extraBpVerifier != nil {
		verified = v.extraBpVerifier(message, timeout, sig)
	}

	if !verified && v.addrVerifier != nil {
		actualSigner, err := DasRecoverSigner(message, timeout, sig)
		if err != nil {
			return nil, err
		}
		isBatchPosterOrSequencer, err := v.addrVerifier.IsBatchPosterOrSequencer(ctx, actualSigner)
		if err != nil {
			return nil, err
		}
		if !isBatchPosterOrSequencer {
			return nil, errors.New("store request not properly signed")
		}
	}

	return v.inner.Store(ctx, message, timeout, sig)
}

func (v *SignatureVerifier) String() string {
	hasAddrVerifier := v.addrVerifier != nil
	hasExtraBpVerifier := v.extraBpVerifier != nil
	return fmt.Sprintf("SignatureVerifier{hasAddrVerifier:%v,hasExtraBpVerifier:%v}", hasAddrVerifier, hasExtraBpVerifier)
}

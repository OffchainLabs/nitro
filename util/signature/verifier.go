// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package signature

import (
	"context"
	"errors"
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/contracts"
)

type Verifier struct {
	config        *VerifierConfig
	authorizedMap map[common.Address]struct{}
	bpValidator   contracts.BatchPosterVerifierInterface
}

type VerifierConfig struct {
	AllowedAddresses []string                `koanf:"allowed-addresses"`
	AcceptSequencer  bool                    `koanf:"accept-sequencer"`
	Dangerous        DangerousVerifierConfig `koanf:"dangerous"`
}

type DangerousVerifierConfig struct {
	AcceptMissing bool `koanf:"accept-missing"`
}

var ErrSignatureNotVerified = errors.New("signature not verified")
var ErrMissingSignature = fmt.Errorf("%w: signature not found", ErrSignatureNotVerified)
var ErrSignerNotApproved = fmt.Errorf("%w: signer not approved", ErrSignatureNotVerified)

func FeedVerifierConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.StringArray(prefix+".allowed-addresses", DefultFeedVerifierConfig.AllowedAddresses, "a list of allowed addresses")
	f.Bool(prefix+".accept-sequencer", DefultFeedVerifierConfig.AcceptSequencer, "accept verified message from sequencer")
	DangerousFeedVerifierConfigAddOptions(prefix+".dangerous", f)
}

func DangerousFeedVerifierConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".accept-missing", DefultFeedVerifierConfig.Dangerous.AcceptMissing, "accept empty as valid signature")
}

var DefultFeedVerifierConfig = VerifierConfig{
	AllowedAddresses: []string{},
	AcceptSequencer:  true,
	Dangerous: DangerousVerifierConfig{
		AcceptMissing: true,
	},
}

var TestingFeedVerifierConfig = VerifierConfig{
	AllowedAddresses: []string{},
	AcceptSequencer:  false,
	Dangerous: DangerousVerifierConfig{
		AcceptMissing: false,
	},
}

func NewVerifier(config *VerifierConfig, bpValidator contracts.BatchPosterVerifierInterface) (*Verifier, error) {
	authorizedMap := make(map[common.Address]struct{}, len(config.AllowedAddresses))
	for _, addrString := range config.AllowedAddresses {
		addr := common.HexToAddress(addrString)
		authorizedMap[addr] = struct{}{}
	}
	if bpValidator == nil && !config.Dangerous.AcceptMissing && config.AcceptSequencer {
		return nil, errors.New("cannot read batch poster addresses")
	}
	return &Verifier{
		config:        config,
		authorizedMap: authorizedMap,
		bpValidator:   bpValidator,
	}, nil
}

func (v *Verifier) VerifyHash(ctx context.Context, signature []byte, hash common.Hash) error {
	return v.verifyClosure(ctx, signature, hash)
}

func (v *Verifier) VerifyData(ctx context.Context, signature []byte, data ...[]byte) error {
	return v.verifyClosure(ctx, signature, crypto.Keccak256Hash(data...))
}

func (v *Verifier) verifyClosure(ctx context.Context, sig []byte, hash common.Hash) error {
	if len(sig) == 0 {
		if v.config.Dangerous.AcceptMissing {
			// Signature missing and not required
			return nil
		}
		return ErrMissingSignature
	}

	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		// nolint:nilerr
		return ErrSignatureNotVerified
	}

	addr := crypto.PubkeyToAddress(*sigPublicKey)

	if _, exists := v.authorizedMap[addr]; exists {
		return nil
	}

	if v.config.Dangerous.AcceptMissing && v.bpValidator == nil {
		return nil
	}

	if !v.config.AcceptSequencer || v.bpValidator == nil {
		return ErrSignerNotApproved
	}

	batchPoster, err := v.bpValidator.IsBatchPoster(ctx, addr)
	if err != nil {
		return err
	}

	if !batchPoster {
		return ErrSignerNotApproved
	}

	return nil
}

// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package signature

import (
	"context"
	"errors"

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
	AllowedAddresses   []string                `koanf:"allowed-addresses"`
	AcceptBatchPosters bool                    `koanf:"accept-batch-posters"`
	Dangerous          DangerousVerifierConfig `koanf:"dangerous"`
}

type DangerousVerifierConfig struct {
	AcceptMissing bool `koanf:"accept-missing"`
}

func FeedVerifierConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.StringArray(prefix+".allowed-addresses", DefultFeedVerifierConfig.AllowedAddresses, "a list of allowed addresses")
	f.Bool(prefix+".accept-batch-posters", DefultFeedVerifierConfig.AcceptBatchPosters, "accept verified message from batch posters")
	DangerousFeedVerifierConfigAddOptions(prefix+".dangerous", f)
}

func DangerousFeedVerifierConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".accept-missing", DefultFeedVerifierConfig.Dangerous.AcceptMissing, "accept empty as valid signature")
}

var DefultFeedVerifierConfig = VerifierConfig{
	AllowedAddresses:   []string{},
	AcceptBatchPosters: true,
	Dangerous: DangerousVerifierConfig{
		AcceptMissing: true,
	},
}

var TestingFeedVerifierConfig = VerifierConfig{
	AllowedAddresses:   []string{},
	AcceptBatchPosters: false,
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
	if bpValidator == nil && config.AcceptBatchPosters {
		return nil, errors.New("cannot read batch poster addresses")
	}
	return &Verifier{
		config:        config,
		authorizedMap: authorizedMap,
		bpValidator:   bpValidator,
	}, nil
}

func (v *Verifier) VerifyHash(ctx context.Context, signature []byte, hash common.Hash) (bool, error) {
	return v.verifyClosure(ctx, signature, hash)
}

func (v *Verifier) VerifyData(ctx context.Context, signature []byte, data ...[]byte) (bool, error) {
	return v.verifyClosure(ctx, signature, crypto.Keccak256Hash(data...))
}

var ErrMissingSignature = errors.New("missing required signature")

func (v *Verifier) verifyClosureLocal(sig []byte, hash common.Hash) (bool, common.Address, error) {
	if len(sig) == 0 {
		if v.config.Dangerous.AcceptMissing {
			// Signature missing and not required
			return true, common.Address{}, nil
		}
		return false, common.Address{}, ErrMissingSignature
	}

	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		// nolint:nilerr
		return false, common.Address{}, nil
	}

	addr := crypto.PubkeyToAddress(*sigPublicKey)

	if _, exists := v.authorizedMap[addr]; exists {
		return true, addr, nil
	}

	return false, addr, nil
}

func (v *Verifier) verifyClosure(ctx context.Context, sig []byte, hash common.Hash) (bool, error) {
	valid, addr, err := v.verifyClosureLocal(sig, hash)
	if err != nil {
		return false, err
	}
	if valid {
		return true, nil
	}
	if v.bpValidator == nil || !v.config.AcceptBatchPosters {
		return false, nil
	}

	return v.bpValidator.IsBatchPoster(ctx, addr)
}

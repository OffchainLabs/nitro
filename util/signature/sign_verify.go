package signature

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/util/contracts"
	flag "github.com/spf13/pflag"
)

type SignVerify struct {
	verifier   *Verifier
	signerFunc DataSignerFunc
	fallback   *SimpleHmac
	config     *SignVerifyConfig
}

type SignVerifyConfig struct {
	ECDSA             VerifierConfig   `koanf:"ecdsa"`
	SymmetricFallback bool             `koanf:"symmetric-fallback"`
	SymmetricSign     bool             `koanf:"symmetric-sign"`
	Symmetric         SimpleHmacConfig `koanf:"symmetric"`
}

func SignVerifyConfigAddOptions(prefix string, f *flag.FlagSet) {
	FeedVerifierConfigAddOptions(prefix+".ecdsa", f)
	f.Bool(prefix+".symmetric-fallback", DefaultSignVerifyConfig.SymmetricFallback, "if to fall back to symmetric hmac")
	f.Bool(prefix+".symmetric-sign", DefaultSignVerifyConfig.SymmetricSign, "if to sign with symmetric hmac")
	SimpleHmacConfigAddOptions(prefix+".symmetric", f)
}

var DefaultSignVerifyConfig = SignVerifyConfig{
	ECDSA:             DefultFeedVerifierConfig,
	SymmetricFallback: false,
	SymmetricSign:     false,
	Symmetric:         EmptySimpleHmacConfig,
}
var TestSignVerifyConfig = SignVerifyConfig{
	ECDSA: VerifierConfig{
		AcceptSequencer: true,
	},
	SymmetricFallback: false,
	SymmetricSign:     false,
	Symmetric:         TestSimpleHmacConfig,
}

func NewSignVerify(config *SignVerifyConfig, signerFunc DataSignerFunc, bpValidator contracts.BatchPosterVerifierInterface) (*SignVerify, error) {
	var fallback *SimpleHmac
	if config.SymmetricFallback {
		var err error
		fallback, err = NewSimpleHmac(&config.Symmetric)
		if err != nil {
			return nil, err
		}
	}
	verifier, err := NewVerifier(&config.ECDSA, bpValidator)
	if err != nil {
		return nil, err
	}
	return &SignVerify{
		verifier:   verifier,
		signerFunc: signerFunc,
		fallback:   fallback,
		config:     config,
	}, nil
}

func (v *SignVerify) VerifySignature(ctx context.Context, signature []byte, data ...[]byte) error {
	ecdsaErr := v.verifier.verifyClosure(ctx, signature, crypto.Keccak256Hash(data...))
	if ecdsaErr == nil {
		return nil
	}
	if !v.config.SymmetricFallback {
		return ecdsaErr
	}
	return v.fallback.VerifySignature(signature, data...)
}

func (v *SignVerify) SignMessage(data ...[]byte) ([]byte, error) {
	if v.config.SymmetricSign {
		return v.fallback.SignMessage(data...)
	}
	if v.signerFunc == nil {
		if v.config.ECDSA.Dangerous.AcceptMissing {
			return make([]byte, 0), nil
		}
		return nil, errors.New("no private key. cannot sign messages")
	}
	return v.signerFunc(crypto.Keccak256Hash(data...).Bytes())
}

package signature

import (
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
	ECDSA: VerifierConfig{
		AcceptBatchPosters: true,
	},
	SymmetricFallback: true,
	SymmetricSign:     true,
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

// does not check batch posters
func (v *SignVerify) VerifySignature(signature []byte, data ...[]byte) (bool, error) {
	ecdsaValid, _, ecdsaErr := v.verifier.verifyClosureLocal(signature, crypto.Keccak256Hash(data...))
	if ecdsaErr == nil && ecdsaValid {
		return true, nil
	}
	if !v.config.SymmetricFallback {
		return ecdsaValid, ecdsaErr
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

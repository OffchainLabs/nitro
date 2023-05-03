package signature

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type SimpleHmac struct {
	config                  *SimpleHmacConfig
	signingKey              *common.Hash
	fallbackVerificationKey *common.Hash
}

func NewSimpleHmac(config *SimpleHmacConfig) (*SimpleHmac, error) {
	signingKey, err := LoadSigningKey(config.SigningKey)
	if err != nil {
		return nil, err
	}
	fallbackVerificationKey, err := LoadSigningKey(config.FallbackVerificationKey)
	if err != nil {
		return nil, err
	}
	if signingKey == nil && fallbackVerificationKey != nil {
		return nil, errors.New("cannot have fallback-verification-key without signing-key")
	}
	if signingKey == nil && !config.Dangerous.DisableSignatureVerification {
		return nil, errors.New("signature verification is enabled but no key is present")
	}
	return &SimpleHmac{
		config:                  config,
		signingKey:              signingKey,
		fallbackVerificationKey: fallbackVerificationKey,
	}, nil
}

type SimpleHmacConfig struct {
	SigningKey              string                    `koanf:"signing-key"`
	FallbackVerificationKey string                    `koanf:"fallback-verification-key"`
	Dangerous               SimpleHmacDangerousConfig `koanf:"dangerous"`
}

type SimpleHmacDangerousConfig struct {
	DisableSignatureVerification bool `koanf:"disable-signature-verification"`
}

func SimpleHmacDangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".disable-signature-verification", DefaultSimpleHmacDangerousConfig.DisableSignatureVerification, "disable message signature verification")
}

func SimpleHmacConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".signing-key", EmptySimpleHmacConfig.SigningKey, "a 32-byte (64-character) hex string used to sign messages, or a path to a file containing it")
	f.String(prefix+".fallback-verification-key", EmptySimpleHmacConfig.SigningKey, "a fallback key used for message verification")
	SimpleHmacDangerousConfigAddOptions(prefix+".dangerous", f)
}

var DefaultSimpleHmacDangerousConfig = SimpleHmacDangerousConfig{
	DisableSignatureVerification: false,
}

var EmptySimpleHmacConfig = SimpleHmacConfig{
	SigningKey:              "",
	FallbackVerificationKey: "",
}

var TestSimpleHmacConfig = SimpleHmacConfig{
	SigningKey:              "b561f5d5d98debc783aa8a1472d67ec3bcd532a1c8d95e5cb23caa70c649f7c9",
	FallbackVerificationKey: "",
	Dangerous: SimpleHmacDangerousConfig{
		DisableSignatureVerification: false,
	},
}

var keyIsHexRegex = regexp.MustCompile("^(0x)?[a-fA-F0-9]{64}$")

func LoadSigningKey(keyConfig string) (*common.Hash, error) {
	if keyConfig == "" {
		return nil, nil
	}
	keyIsHex := keyIsHexRegex.Match([]byte(keyConfig))
	var keyString string
	if keyIsHex {
		keyString = keyConfig
	} else {
		contents, err := os.ReadFile(keyConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read signing key file: %w", err)
		}
		s := strings.TrimSpace(string(contents))
		if keyIsHexRegex.Match([]byte(s)) {
			keyString = s
		} else {
			return nil, errors.New("signing key file contents are not 32 bytes of hex")
		}
	}
	hash := common.HexToHash(keyString)
	return &hash, nil
}

func prependBytes(first []byte, rest ...[]byte) [][]byte {
	return append([][]byte{first}, rest...)
}

// On success, extracts the message from the message+signature data passed in, and returns it
func (h *SimpleHmac) VerifySignature(sig []byte, data ...[]byte) error {
	if h.config.Dangerous.DisableSignatureVerification {
		return nil
	}
	if len(sig) != 32 {
		return fmt.Errorf("%w: signature must be exactly 32 bytes", ErrSignatureNotVerified)
	}

	expectHmac := crypto.Keccak256Hash(prependBytes(h.signingKey[:], data...)...)
	if subtle.ConstantTimeCompare(expectHmac[:], sig) == 1 {
		return nil
	}

	if h.fallbackVerificationKey != nil {
		expectHmac = crypto.Keccak256Hash(prependBytes(h.fallbackVerificationKey[:], data...)...)
		if subtle.ConstantTimeCompare(expectHmac[:], sig) == 1 {
			return nil
		}
	}

	return ErrSignatureNotVerified
}

func (h *SimpleHmac) SignMessage(data ...[]byte) ([]byte, error) {
	var hmac [32]byte
	if h.signingKey != nil {
		hmac = crypto.Keccak256Hash(prependBytes(h.signingKey[:], data...)...)
	}
	return hmac[:], nil
}

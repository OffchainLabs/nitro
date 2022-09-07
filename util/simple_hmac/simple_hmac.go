package simple_hmac

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
	signingKey, err := loadSigningKey(config.SigningKey)
	if err != nil {
		return nil, err
	}
	fallbackVerificationKey, err := loadSigningKey(config.FallbackVerificationKey)
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
	f.String(prefix+".signing-key", DefaultSimpleHmacConfig.SigningKey, "a 32-byte (64-character) hex string used to sign messages, or a path to a file containing it")
	f.String(prefix+".fallback-verification-key", DefaultSimpleHmacConfig.SigningKey, "a fallback key used for message verification")
	SimpleHmacDangerousConfigAddOptions(prefix+".dangerous", f)
}

var DefaultSimpleHmacDangerousConfig = SimpleHmacDangerousConfig{
	DisableSignatureVerification: false,
}

var DefaultSimpleHmacConfig = SimpleHmacConfig{
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

func loadSigningKey(keyConfig string) (*common.Hash, error) {
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

// On success, extracts the message from the message+signature data passed in, and returns it
func (h *SimpleHmac) VerifyMessageSignature(prefix []byte, data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}
	msg := data[32:]
	if h.config.Dangerous.DisableSignatureVerification {
		return msg, nil
	}
	var haveHmac common.Hash
	copy(haveHmac[:], data[:32])

	expectHmac := crypto.Keccak256Hash(h.signingKey[:], prefix, msg)
	if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
		return msg, nil
	}

	if h.fallbackVerificationKey != nil {
		expectHmac = crypto.Keccak256Hash(h.fallbackVerificationKey[:], prefix, msg)
		if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
			return msg, nil
		}
	}

	if haveHmac == (common.Hash{}) {
		return nil, errors.New("no HMAC signature present but signature verification is enabled")
	} else {
		return nil, errors.New("HMAC signature doesn't match expected value(s)")
	}
}

func (h *SimpleHmac) SignMessage(prefix []byte, msg []byte) []byte {
	var hmac [32]byte
	if h.signingKey != nil {
		hmac = crypto.Keccak256Hash(h.signingKey[:], prefix, msg)
	}
	return append(hmac[:], msg...)
}

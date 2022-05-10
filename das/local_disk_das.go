// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	flag "github.com/spf13/pflag"
)

type LocalDiskDASConfig struct {
	KeyDir            string `koanf:"key-dir"`
	PrivKey           string `koanf:"priv-key"`
	DataDir           string `koanf:"data-dir"`
	AllowGenerateKeys bool   `koanf:"allow-generate-keys"`
}

func LocalDiskDASConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".key-dir", "", fmt.Sprintf("The directory to read the bls keypair ('%s' and '%s') from", DefaultPubKeyFilename, DefaultPrivKeyFilename))
	f.String(prefix+".priv-key", "", "The base64 BLS private key to use for signing DAS certificates")
	f.String(prefix+".data-dir", "", "The directory to use as the DAS file-based database")
	f.Bool(prefix+".allow-generate-keys", false, "Allow the local disk DAS to generate its own keys in key-dir if they don't already exist")
}

type LocalDiskDAS struct {
	config  LocalDiskDASConfig
	privKey *blsSignatures.PrivateKey
}

func NewLocalDiskDAS(config LocalDiskDASConfig) (*LocalDiskDAS, error) {
	var privKey *blsSignatures.PrivateKey
	var err error
	if len(config.PrivKey) != 0 {
		privKey, err = DecodeBase64BLSPrivateKey([]byte(config.PrivKey))
		if err != nil {
			return nil, fmt.Errorf("'priv-key' was invalid: %w", err)
		}
	} else {
		_, privKey, err = ReadKeysFromFile(config.KeyDir)
		if err != nil {
			if os.IsNotExist(err) {
				if config.AllowGenerateKeys {
					_, privKey, err = GenerateAndStoreKeys(config.KeyDir)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("Required BLS keypair did not exist at %s", config.KeyDir)
				}
			} else {
				return nil, err
			}
		}
	}

	return &LocalDiskDAS{
		config:  config,
		privKey: privKey,
	}, nil
}

func (das *LocalDiskDAS) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = 0 // The aggregator decides on the mask for each signer.

	fields := serializeSignableFields(*c)
	c.Sig, err = blsSignatures.SignMessage(*das.privKey, fields)
	if err != nil {
		return nil, err
	}

	path := das.config.DataDir + "/" + base32.StdEncoding.EncodeToString(c.DataHash[:])
	log.Debug("Storing message at", "path", path)

	err = os.WriteFile(path, message, 0600)
	if err != nil {
		return nil, err
	}

	// leave the keyset field of c as zeroes

	return c, nil
}

func (das *LocalDiskDAS) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := das.config.DataDir + "/" + base32.StdEncoding.EncodeToString(cert.DataHash[:])
	log.Debug("Retrieving message from", "path", path)

	originalMessage, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var originalMessageHash [32]byte
	copy(originalMessageHash[:], crypto.Keccak256(originalMessage))
	if originalMessageHash != cert.DataHash {
		return nil, errors.New("Retrieved message stored hash doesn't match calculated hash.")
	}

	// The cert passed in may have an aggregate signature, so we don't
	// check the signature against this DAS's public key here.

	return originalMessage, nil
}

func (d *LocalDiskDAS) String() string {
	return fmt.Sprintf("LocalDiskDAS{config:%v}", d.config)
}

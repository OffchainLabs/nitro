// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
	flag "github.com/spf13/pflag"
)

var ErrDasKeysetNotFound = errors.New("no such keyset")

type LocalDiskDASConfig struct {
	KeyDir             string `koanf:"key-dir"`
	PrivKey            string `koanf:"priv-key"`
	DataDir            string `koanf:"data-dir"`
	AllowGenerateKeys  bool   `koanf:"allow-generate-keys"`
	StoreSignerAddress string `koanf:"store-signer-address"`
}

func LocalDiskDASConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".key-dir", "", fmt.Sprintf("The directory to read the bls keypair ('%s' and '%s') from", DefaultPubKeyFilename, DefaultPrivKeyFilename))
	f.String(prefix+".priv-key", "", "The base64 BLS private key to use for signing DAS certificates")
	f.String(prefix+".data-dir", "", "The directory to use as the DAS file-based database")
	f.Bool(prefix+".allow-generate-keys", false, "Allow the local disk DAS to generate its own keys in key-dir if they don't already exist")
	f.String(prefix+".store-signer-address", "", "Address required to sign stores, or empty if anyone can store")
}

type LocalDiskDAS struct {
	config          LocalDiskDASConfig
	privKey         *blsSignatures.PrivateKey
	keysetHash      [32]byte
	keysetBytes     []byte
	storeSignerAddr *common.Address
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

	publicKey, err := blsSignatures.PublicKeyFromPrivateKey(*privKey)
	if err != nil {
		return nil, err
	}

	keyset := &arbstate.DataAvailabilityKeyset{
		AssumedHonest: 1,
		PubKeys:       []blsSignatures.PublicKey{publicKey},
	}
	ksBuf := bytes.NewBuffer([]byte{})
	if err := keyset.Serialize(ksBuf); err != nil {
		return nil, err
	}
	ksHashBuf, err := keyset.Hash()
	if err != nil {
		return nil, err
	}
	var ksHash [32]byte
	copy(ksHash[:], ksHashBuf)

	return &LocalDiskDAS{
		config:          config,
		privKey:         privKey,
		keysetHash:      ksHash,
		keysetBytes:     ksBuf.Bytes(),
		storeSignerAddr: StoreSignerAddressFromString(config.StoreSignerAddress),
	}, nil
}

func (das *LocalDiskDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (c *arbstate.DataAvailabilityCertificate, err error) {
	if das.storeSignerAddr != nil {
		actualSigner, err := DasRecoverSigner(message, timeout, sig)
		if err != nil {
			return nil, err
		}
		if actualSigner != *das.storeSignerAddr {
			return nil, errors.New("store request not properly signed")
		}
	}

	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = 1 // The aggregator will override this if we're part of a committee.

	fields := c.SerializeSignableFields()
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

	c.KeysetHash = das.keysetHash

	return c, nil
}

func (das *LocalDiskDAS) Retrieve(ctx context.Context, cert *arbstate.DataAvailabilityCertificate) ([]byte, error) {
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

func (das *LocalDiskDAS) KeysetFromHash(ctx context.Context, ksHash []byte) ([]byte, error) {
	if !bytes.Equal(ksHash, das.keysetHash[:]) {
		return nil, ErrDasKeysetNotFound
	}
	return das.keysetBytes, nil
}

func (das *LocalDiskDAS) CurrentKeysetBytes(ctx context.Context) ([]byte, error) {
	return das.keysetBytes, nil
}

func (d *LocalDiskDAS) String() string {
	return fmt.Sprintf("LocalDiskDAS{config:%v}", d.config)
}

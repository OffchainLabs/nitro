// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"math/bits"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type LocalDiskDataAvailabilityService struct {
	dbPath     string
	pubKey     *blsSignatures.PublicKey
	privKey    blsSignatures.PrivateKey
	signerMask uint64
}

func readKeysFromFile(dbPath string) (*blsSignatures.PublicKey, blsSignatures.PrivateKey, error) {
	pubKeyPath := dbPath + "/pubkey"
	privKeyPath := dbPath + "/privkey"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, nil, err
	}
	privKeyBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return nil, nil, err
	}
	pubKey, err := blsSignatures.PublicKeyFromBytes(pubKeyBytes, true)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := blsSignatures.PrivateKeyFromBytes(privKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	return &pubKey, privKey, nil
}

func generateAndStoreKeys(dbPath string) (*blsSignatures.PublicKey, blsSignatures.PrivateKey, error) {
	pubKey, privKey, err := blsSignatures.GenerateKeys()
	if err != nil {
		return nil, nil, err
	}
	pubKeyPath := dbPath + "/pubkey"
	privKeyPath := dbPath + "/privkey"
	err = os.WriteFile(pubKeyPath, blsSignatures.PublicKeyToBytes(pubKey), 0600)
	if err != nil {
		return nil, nil, err
	}
	err = os.WriteFile(privKeyPath, blsSignatures.PrivateKeyToBytes(privKey), 0600)
	if err != nil {
		return nil, nil, err
	}
	return &pubKey, privKey, nil
}

func NewLocalDiskDataAvailabilityService(dbPath string, signerMask uint64) (*LocalDiskDataAvailabilityService, error) {
	if bits.OnesCount64(signerMask) != 1 {
		return nil, fmt.Errorf("Tried to construct a local DAS with invalid signerMask %X", signerMask)
	}

	pubKey, privKey, err := readKeysFromFile(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			pubKey, privKey, err = generateAndStoreKeys(dbPath)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &LocalDiskDataAvailabilityService{
		dbPath:     dbPath,
		pubKey:     pubKey,
		privKey:    privKey,
		signerMask: signerMask,
	}, nil
}

func (das *LocalDiskDataAvailabilityService) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = das.signerMask

	fields := serializeSignableFields(*c)
	c.Sig, err = blsSignatures.SignMessage(das.privKey, fields)
	if err != nil {
		return nil, err
	}

	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(c.DataHash[:])
	log.Debug("Storing message at", "path", path)

	err = os.WriteFile(path, message, 0600)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (das *LocalDiskDataAvailabilityService) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(cert.DataHash[:])
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

func (d *LocalDiskDataAvailabilityService) String() string {
	return fmt.Sprintf("LocalDiskDataAvailabilityService{signersMask:%d,dbPath:%s}", d.signerMask, d.dbPath)
}

func (d *LocalDiskDataAvailabilityService) PrivateKey() blsSignatures.PrivateKey {
	return d.privKey
}

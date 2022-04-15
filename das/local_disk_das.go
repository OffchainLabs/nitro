// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/base32"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

var dasMutex sync.Mutex

type LocalDiskDataAvailabilityService struct {
	dbPath          string
	pubKey          *blsSignatures.PublicKey
	privKey         blsSignatures.PrivateKey
	retentionPeriod time.Duration
	signerMask      uint64
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

func NewLocalDiskDataAvailabilityService(dbPath string) (*LocalDiskDataAvailabilityService, error) {
	dasMutex.Lock()
	defer dasMutex.Unlock()
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
		dbPath:  dbPath,
		pubKey:  pubKey,
		privKey: privKey,
	}, nil
}

func (das *LocalDiskDataAvailabilityService) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	dasMutex.Lock()
	defer dasMutex.Unlock()

	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	if timeout == CALLEE_PICKS_TIMEOUT {
		c.Timeout = uint64(time.Now().Add(das.retentionPeriod).Unix())
	} else {
		c.Timeout = timeout
	}
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
	dasMutex.Lock()
	defer dasMutex.Unlock()

	cert, _, err := arbstate.DeserializeDASCertFrom(certBytes)
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

	signedBlob := serializeSignableFields(*cert)
	sigMatch, err := blsSignatures.VerifySignature(cert.Sig, signedBlob, *das.pubKey)
	if err != nil {
		return nil, err
	}
	if !sigMatch {
		return nil, errors.New("Signature of data in cert passed in doesn't match")
	}

	return originalMessage, nil
}

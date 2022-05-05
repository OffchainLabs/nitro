// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math/bits"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

var dasMutex sync.Mutex

type LocalDiskDataAvailabilityService struct {
	dbPath     string
	pubKey     *blsSignatures.PublicKey
	privKey    blsSignatures.PrivateKey
	signerMask uint64
}

func readKeysFromFile(dbPath string) (*blsSignatures.PublicKey, blsSignatures.PrivateKey, error) {
	pubKeyPath := dbPath + "/pubkey"
	pubKeyEncodedBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, nil, err
	}

	// Ethereum's BLS library doesn't like the byte slice containing the BLS keys to be
	// any larger than necessary, so we need to create a Decoder to avoid returning any padding.
	pubKeyDecoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(pubKeyEncodedBytes))
	pubKeyBytes, err := ioutil.ReadAll(pubKeyDecoder)
	if err != nil {
		return nil, nil, err
	}
	pubKey, err := blsSignatures.PublicKeyFromBytes(pubKeyBytes, true)
	if err != nil {
		return nil, nil, err
	}

	privKeyPath := dbPath + "/privkey"
	privKeyEncodedBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return nil, nil, err
	}
	privKeyDecoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(privKeyEncodedBytes))
	privKeyBytes, err := ioutil.ReadAll(privKeyDecoder)
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
	pubKeyBytes := blsSignatures.PublicKeyToBytes(pubKey)
	encodedPubKey := make([]byte, base64.StdEncoding.EncodedLen(len(pubKeyBytes)))
	base64.StdEncoding.Encode(encodedPubKey, pubKeyBytes)
	err = os.WriteFile(pubKeyPath, encodedPubKey, 0600)
	if err != nil {
		return nil, nil, err
	}

	privKeyPath := dbPath + "/privkey"
	privKeyBytes := blsSignatures.PrivateKeyToBytes(privKey)
	encodedPrivKey := make([]byte, base64.StdEncoding.EncodedLen(len(privKeyBytes)))
	base64.StdEncoding.Encode(encodedPrivKey, privKeyBytes)
	err = os.WriteFile(privKeyPath, encodedPrivKey, 0600)
	if err != nil {
		return nil, nil, err
	}
	return &pubKey, privKey, nil
}

func NewLocalDiskDataAvailabilityService(dbPath string, signerMask uint64) (*LocalDiskDataAvailabilityService, error) {
	dasMutex.Lock()
	defer dasMutex.Unlock()
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
	dasMutex.Lock()
	defer dasMutex.Unlock()

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
	dasMutex.Lock()
	defer dasMutex.Unlock()

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

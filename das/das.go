//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"context"
	"encoding/base32"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbstate"
	"github.com/offchainlabs/arbstate/blsSignatures"
)

type DataAvailabilityMode uint64

const (
	OnchainDataAvailability DataAvailabilityMode = iota
	LocalDataAvailability
)

type DataAvailabilityConfig struct {
	LocalDiskDataDir string
}

var DefaultDataAvailabilityConfig = DataAvailabilityConfig{}

func SetDASMessageHeaderByte(header *byte) {
	*header |= arbstate.DASMessageHeaderFlag
}

type DataAvailabilityServiceWriter interface {
	Store(ctx context.Context, message []byte) ([]byte, blsSignatures.Signature, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityServiceReader
	DataAvailabilityServiceWriter
}

type LocalDiskDataAvailabilityService struct {
	dbPath  string
	pubKey  *blsSignatures.PublicKey
	privKey blsSignatures.PrivateKey
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

func (das *LocalDiskDataAvailabilityService) Store(ctx context.Context, message []byte) ([]byte, blsSignatures.Signature, error) {
	h := crypto.Keccak256(message)

	sig, err := blsSignatures.SignMessage(das.privKey, h)
	if err != nil {
		return nil, nil, err
	}

	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(h)

	log.Debug("Storing message at", "path", path)

	err = os.WriteFile(path, message, 0600)
	if err != nil {
		return nil, nil, err
	}

	return h, sig, nil
}

func (das *LocalDiskDataAvailabilityService) Retrieve(ctx context.Context, hash []byte) ([]byte, error) {
	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(hash)
	log.Debug("Retrieving message from", "path", path)
	return os.ReadFile(path)
}

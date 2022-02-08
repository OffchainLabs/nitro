//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"encoding/base32"
	"io/ioutil"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/blsSignatures"
)

type DataAvailabilityService interface {
	Store(message []byte) ([]byte, blsSignatures.Signature, error)
	Retrieve(hash []byte) ([]byte, error)
}

type LocalDiskDataAvailabilityService struct {
	dbPath  string
	pubKey  *blsSignatures.PublicKey
	privKey blsSignatures.PrivateKey
}

var singletonDAS DataAvailabilityService
var singletonDASMutex sync.Mutex

func GetSingletonTestingDAS() DataAvailabilityService {
	singletonDASMutex.Lock()
	defer singletonDASMutex.Unlock()
	if singletonDAS == nil {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		if err != nil {
			panic(err)
		}

		singletonDAS, err = NewLocalDiskDataAvailabilityService(dbPath)
		if err != nil {
			panic(err)
		}
		log.Error("Created the singleton das using", "dbPath", dbPath)
	} else {
		log.Error("Getting the singleton das")
	}
	return singletonDAS
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

func (das *LocalDiskDataAvailabilityService) Store(message []byte) ([]byte, blsSignatures.Signature, error) {
	h := crypto.Keccak256(message)

	sig, err := blsSignatures.SignMessage(das.privKey, h)
	if err != nil {
		return nil, nil, err
	}

	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(h)

	log.Error("Storing message at", "path", path)

	err = os.WriteFile(path, message, 0600)
	if err != nil {
		return nil, nil, err
	}

	return h, sig, nil
}

func (das *LocalDiskDataAvailabilityService) Retrieve(hash []byte) ([]byte, error) {
	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(hash)
	log.Error("Retrieving message from", "path", path)
	return os.ReadFile(path)
}

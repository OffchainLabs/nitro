//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"context"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"time"

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

func serializeSignableFields(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 32+8+8)
	buf = append(buf, c.DataHash[:]...)

	var intData [8]byte
	binary.BigEndian.PutUint64(intData[:], c.Timeout)
	buf = append(buf, intData[:]...)

	binary.BigEndian.PutUint64(intData[:], c.SignersMask)
	buf = append(buf, intData[:]...)
	return buf
}

func Serialize(c arbstate.DataAvailabilityCertificate) []byte {
	buf := make([]byte, 0, 1+reflect.TypeOf(arbstate.DataAvailabilityCertificate{}).Size())

	buf = append(buf, arbstate.DASMessageHeaderFlag)

	buf = append(buf, serializeSignableFields(c)...)

	return append(buf, blsSignatures.SignatureToBytes(c.Sig)...)
}

type DataAvailabilityServiceWriter interface {
	Store(ctx context.Context, message []byte) (*arbstate.DataAvailabilityCertificate, error)
}

type DataAvailabilityService interface {
	arbstate.DataAvailabilityServiceReader
	DataAvailabilityServiceWriter
}

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
	pubKey, privKey, err := readKeysFromFile(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			pubKey, privKey, err = generateAndStoreKeys(dbPath)
			log.Error("GENERATING keys", "pubkey", blsSignatures.PublicKeyToBytes(*pubKey))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		log.Error("READ keys", "pubkey", blsSignatures.PublicKeyToBytes(*pubKey))
	}

	return &LocalDiskDataAvailabilityService{
		dbPath:  dbPath,
		pubKey:  pubKey,
		privKey: privKey,
	}, nil
}

func (das *LocalDiskDataAvailabilityService) Store(ctx context.Context, message []byte) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = uint64(time.Now().Add(das.retentionPeriod).Unix())
	c.SignersMask = das.signerMask

	fields := serializeSignableFields(*c)
	log.Error("SIGNING fields", "blob", fields, "pubkey", blsSignatures.PublicKeyToBytes(*das.pubKey), "privKey", blsSignatures.PrivateKeyToBytes(das.privKey))
	c.Sig, err = blsSignatures.SignMessage(das.privKey, fields)
	if err != nil {
		return nil, err
	}

	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(c.DataHash[:])
	log.Debug("Storing message at", "path", path)

	// Store the cert at the beginning of the message so we can validate it.
	toWrite := Serialize(*c)
	toWrite = append(toWrite, message...)

	err = os.WriteFile(path, toWrite, 0600)
	if err != nil {
		return nil, err
	}
	log.Error("WRITE File and total hash", "path", path, "hash", crypto.Keccak256(toWrite))

	return c, nil
}

func (das *LocalDiskDataAvailabilityService) Retrieve(ctx context.Context, hash []byte) ([]byte, error) {
	path := das.dbPath + "/" + base32.StdEncoding.EncodeToString(hash)
	log.Debug("Retrieving message from", "path", path)

	fileData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	log.Error("READ File and total hash", "path", path, "hash", crypto.Keccak256(fileData))

	cert, bytesRead, err := arbstate.DeserializeDASCertFrom(fileData)
	if err != nil {
		return nil, err
	}

	originalMessage := fileData[bytesRead:]
	var originalMessageHash [32]byte
	copy(originalMessageHash[:], crypto.Keccak256(originalMessage))
	if originalMessageHash != cert.DataHash {
		return nil, errors.New("Retrieved message stored hash doesn't match calculated hash.")
	}

	signedBlob := serializeSignableFields(*cert)
	log.Error("CHECK fields", "blob", signedBlob, "pubkey", blsSignatures.PublicKeyToBytes(*das.pubKey), "privKey", blsSignatures.PrivateKeyToBytes(das.privKey))
	sigMatch, err := blsSignatures.VerifySignature(cert.Sig, signedBlob, *das.pubKey)
	if err != nil {
		return nil, err
	}
	if !sigMatch {
		return nil, errors.New("Signature of DAS stored data doesn't match")
	}

	return originalMessage, nil
}

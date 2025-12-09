// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build challengetest && !race

package arbtest

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/referenceda"
	"github.com/offchainlabs/nitro/util/containers"
)

type EvilStrategy int

const (
	NoEvil                  EvilStrategy = iota // Normal behavior
	EvilDataGoodCert                            // Wrong data, correct cert
	EvilDataEvilCert                            // Wrong data, matching evil cert
	UntrustedSignerCert                         // Valid format cert signed by untrusted key
	ValidCertClaimedInvalid                     // Valid cert, but validator claims invalid
)

const (
	// ValidityProofValid is the byte value indicating a valid certificate in evil proofs
	ValidityProofValid = 1
	// ValidityProofInvalid is the byte value indicating an invalid certificate in evil proofs
	ValidityProofInvalid = 0
	// ValidityProofMarker is the marker byte used in validity proofs
	ValidityProofMarker = 0x01
)

// EvilDAProvider implements both Reader and Validator interfaces
// It wraps the regular ReferenceDA components and intercepts specific certificates
// Note: It's safe to create new underlying readers/validators because they all use
// the same singleton storage instance via GetInMemoryStorage()
type EvilDAProvider struct {
	reader                 daprovider.Reader
	validator              daprovider.Validator
	evilData               map[common.Hash][]byte // sha256 dataHash -> evil data
	untrustedSignerAddress *common.Address        // Address of untrusted signer to lie about
	invalidClaimCerts      map[common.Hash]bool   // Keccak256 of certs to claim are invalid
	mu                     sync.RWMutex
}

func NewEvilDAProvider(l1Client *ethclient.Client, validatorAddr common.Address) *EvilDAProvider {
	// Create fresh ReferenceDA components - they'll all share the singleton storage
	storage := referenceda.GetInMemoryStorage()
	return &EvilDAProvider{
		reader:            referenceda.NewReader(storage, l1Client, validatorAddr),
		validator:         referenceda.NewValidator(l1Client, validatorAddr),
		evilData:          make(map[common.Hash][]byte),
		invalidClaimCerts: make(map[common.Hash]bool),
	}
}

// SetEvilData configures the provider to return evil data for a specific certificate
func (e *EvilDAProvider) SetEvilData(certHash common.Hash, evilData []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evilData[certHash] = evilData
}

// GetEvilData retrieves evil data for a specific certificate if it exists
func (e *EvilDAProvider) GetEvilData(dataHash [32]byte) ([]byte, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	data, exists := e.evilData[dataHash]
	return data, exists
}

// SetUntrustedSignerAddress configures the provider to lie about certificates from this signer
func (e *EvilDAProvider) SetUntrustedSignerAddress(addr common.Address) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.untrustedSignerAddress = &addr
}

func (e *EvilDAProvider) IsUntrustedSigner(signer common.Address) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.untrustedSignerAddress != nil && signer == *e.untrustedSignerAddress
}

// SetClaimCertInvalid marks a specific certificate (by keccak256 hash) to be claimed as invalid
func (e *EvilDAProvider) SetClaimCertInvalid(certKeccak common.Hash) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.invalidClaimCerts[certKeccak] = true
}

func (e *EvilDAProvider) ShouldClaimCertInvalid(certKeccak common.Hash) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.invalidClaimCerts[certKeccak]
}

// RecoverPayload intercepts and returns evil data if configured
func (e *EvilDAProvider) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	promise := containers.NewPromise[daprovider.PayloadResult](nil)
	go func() {
		certificate := sequencerMsg[40:]
		certKeccak := crypto.Keccak256Hash(certificate)

		if e.ShouldClaimCertInvalid(certKeccak) {
			log.Info("EvilDAProvider rejecting certificate we claim is invalid",
				"certKeccak", certKeccak.Hex(),
				"batchNum", batchNum)
			promise.ProduceError(fmt.Errorf("certificate validation failed: claimed to be invalid"))
			return
		}

		cert, err := referenceda.Deserialize(certificate)
		if err != nil {
			promise.ProduceError(err)
			return
		}

		signer, err := cert.RecoverSigner()
		if err != nil {
			promise.ProduceError(err)
			return
		}

		if e.IsUntrustedSigner(signer) {
			log.Info("EvilDAProvider accepting untrusted certificate",
				"signer", signer.Hex(),
				"dataHash", common.Hash(cert.DataHash).Hex())

			storage := referenceda.GetInMemoryStorage()
			data, err := storage.GetByHash(common.Hash(cert.DataHash))
			if err != nil {
				promise.ProduceError(fmt.Errorf("failed to get data for untrusted cert: %w", err))
			} else {
				promise.Produce(daprovider.PayloadResult{Payload: data})
			}
			return
		}

		if evilData, exists := e.GetEvilData(cert.DataHash); exists {
			log.Info("EvilDAProvider returning evil data",
				"dataHash", common.Hash(cert.DataHash).Hex(),
				"evilDataSize", len(evilData))

			promise.Produce(daprovider.PayloadResult{Payload: evilData})
			return
		}

		// Delegate to underlying reader
		delegatePromise := e.reader.RecoverPayload(batchNum, batchBlockHash, sequencerMsg)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		promise.ProduceResult(result, err)
	}()
	return &promise
}

// CollectPreimages collects preimages for the batch
func (e *EvilDAProvider) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	promise := containers.NewPromise[daprovider.PreimagesResult](nil)
	go func() {
		certificate := sequencerMsg[40:]
		certKeccak := crypto.Keccak256Hash(certificate)

		if e.ShouldClaimCertInvalid(certKeccak) {
			promise.Produce(daprovider.PreimagesResult{Preimages: make(daprovider.PreimagesMap)})
			return
		}

		cert, err := referenceda.Deserialize(certificate)
		if err != nil {
			promise.ProduceError(err)
			return
		}

		signer, err := cert.RecoverSigner()
		if err != nil {
			promise.ProduceError(err)
			return
		}

		if e.IsUntrustedSigner(signer) {
			storage := referenceda.GetInMemoryStorage()
			data, err := storage.GetByHash(common.Hash(cert.DataHash))
			if err != nil {
				promise.ProduceError(err)
				return
			}
			preimages := make(daprovider.PreimagesMap)
			preimageRecorder := daprovider.RecordPreimagesTo(preimages)
			preimageRecorder(certKeccak, data, arbutil.DACertificatePreimageType)
			promise.Produce(daprovider.PreimagesResult{Preimages: preimages})
			return
		}

		if evilData, exists := e.GetEvilData(cert.DataHash); exists {
			preimages := make(daprovider.PreimagesMap)
			preimageRecorder := daprovider.RecordPreimagesTo(preimages)
			preimageRecorder(certKeccak, evilData, arbutil.DACertificatePreimageType)

			promise.Produce(daprovider.PreimagesResult{Preimages: preimages})
			return
		}

		// Delegate to underlying reader
		delegatePromise := e.reader.CollectPreimages(batchNum, batchBlockHash, sequencerMsg)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		promise.ProduceResult(result, err)
	}()
	return &promise
}

// GenerateReadPreimageProof generates proof for evil data if configured, otherwise delegates
func (e *EvilDAProvider) GenerateReadPreimageProof(
	offset uint64,
	certificate []byte,
) containers.PromiseInterface[daprovider.PreimageProofResult] {
	promise := containers.NewPromise[daprovider.PreimageProofResult](nil)
	go func() {
		cert, err := referenceda.Deserialize(certificate)
		if err != nil {
			promise.ProduceError(err)
			return
		}

		if evilData, hasEvil := e.GetEvilData(cert.DataHash); hasEvil {
			// Format: [Version(1), CertificateSize(8), Certificate, PreimageSize(8), PreimageData]
			certLen := len(certificate)
			proof := make([]byte, 1+8+certLen+8+len(evilData))
			proof[0] = 1 // Version
			binary.BigEndian.PutUint64(proof[1:9], uint64(certLen))
			copy(proof[9:9+certLen], certificate)
			binary.BigEndian.PutUint64(proof[9+certLen:9+certLen+8], uint64(len(evilData)))
			copy(proof[9+certLen+8:], evilData)

			log.Debug("EvilDAProvider generating evil proof",
				"dataHash", common.Hash(cert.DataHash).Hex(),
				"evilDataSize", len(evilData))

			promise.Produce(daprovider.PreimageProofResult{Proof: proof})
			return
		}

		// Delegate to underlying validator
		delegatePromise := e.validator.GenerateReadPreimageProof(offset, certificate)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		promise.ProduceResult(result, err)
	}()
	return &promise
}

// GenerateCertificateValidityProof generates a proof of certificate validity
func (e *EvilDAProvider) GenerateCertificateValidityProof(certificate []byte) containers.PromiseInterface[daprovider.ValidityProofResult] {
	promise := containers.NewPromise[daprovider.ValidityProofResult](nil)
	go func() {
		cert, err := referenceda.Deserialize(certificate)
		if err != nil {
			promise.ProduceError(err)
			return
		}

		signer, err := cert.RecoverSigner()
		if err != nil {
			promise.ProduceError(err)
			return
		}

		if e.IsUntrustedSigner(signer) {
			log.Info("EvilDAProvider lying about untrusted certificate validity",
				"signer", signer.Hex(),
				"dataHash", common.Hash(cert.DataHash).Hex())
			promise.Produce(daprovider.ValidityProofResult{Proof: []byte{ValidityProofValid, ValidityProofMarker}})
			return
		}

		certKeccak := crypto.Keccak256Hash(certificate)
		if e.ShouldClaimCertInvalid(certKeccak) {
			log.Info("EvilDAProvider lying about valid certificate (claiming invalid)",
				"certKeccak", certKeccak.Hex(),
				"dataHash", common.Hash(cert.DataHash).Hex())
			promise.Produce(daprovider.ValidityProofResult{Proof: []byte{ValidityProofInvalid, ValidityProofMarker}})
			return
		}

		// Delegate to underlying validator
		delegatePromise := e.validator.GenerateCertificateValidityProof(certificate)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		promise.ProduceResult(result, err)
	}()
	return &promise
}

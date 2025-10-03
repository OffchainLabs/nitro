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

// EvilDAProvider implements both Reader and Validator interfaces
// It wraps the regular ReferenceDA components and intercepts specific certificates
// Note: It's safe to create new underlying readers/validators because they all use
// the same singleton storage instance via GetInMemoryStorage()
type EvilDAProvider struct {
	reader                 daprovider.Reader
	validator              daprovider.Validator
	evilMappings           map[common.Hash][]byte // sha256 dataHash -> evil data
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
		evilMappings:      make(map[common.Hash][]byte),
		invalidClaimCerts: make(map[common.Hash]bool),
	}
}

// SetMapping configures the provider to return evil data for a specific certificate
func (e *EvilDAProvider) SetMapping(certHash common.Hash, evilData []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evilMappings[certHash] = evilData
}

// SetUntrustedSignerAddress configures the provider to lie about certificates from this signer
func (e *EvilDAProvider) SetUntrustedSignerAddress(addr common.Address) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.untrustedSignerAddress = &addr
}

func (e *EvilDAProvider) GetUntrustedSignerAddress() *common.Address {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.untrustedSignerAddress
}

// SetClaimCertInvalid marks a specific certificate (by keccak256 hash) to be claimed as invalid
func (e *EvilDAProvider) SetClaimCertInvalid(certKeccak common.Hash) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.invalidClaimCerts[certKeccak] = true
}

// RecoverPayload intercepts and returns evil data if configured
func (e *EvilDAProvider) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	promise := containers.NewPromise[daprovider.PayloadResult](nil)
	go func() {
		// Check if this is a CustomDA message and extract certificate
		if len(sequencerMsg) > 40 && daprovider.IsDACertificateMessageHeaderByte(sequencerMsg[40]) {
			certificate := sequencerMsg[40:]

			// Check if we're supposed to claim this certificate is invalid
			certKeccak := crypto.Keccak256Hash(certificate)
			e.mu.RLock()
			shouldClaimInvalid := e.invalidClaimCerts[certKeccak]
			e.mu.RUnlock()

			if shouldClaimInvalid {
				log.Info("EvilDAProvider rejecting certificate we claim is invalid",
					"certKeccak", certKeccak.Hex(),
					"batchNum", batchNum)
				// Return an error similar to what would happen with an actually invalid certificate
				promise.ProduceError(fmt.Errorf("certificate validation failed: claimed to be invalid"))
				return
			}

			// Try to deserialize certificate
			cert, err := referenceda.Deserialize(certificate)
			if err == nil {
				// Check if this certificate is from our untrusted signer
				signer, signerErr := cert.RecoverSigner()
				if signerErr == nil {
					untrustedAddr := e.GetUntrustedSignerAddress()

					// If this cert was signed by our known untrusted signer, accept it and return the data
					if untrustedAddr != nil && signer == *untrustedAddr {
						log.Info("EvilDAProvider accepting untrusted certificate",
							"signer", signer.Hex(),
							"dataHash", common.Hash(cert.DataHash).Hex())

						// Get the data from the underlying storage (it was stored with untrusted signer)
						// Delegate to underlying reader
						delegatePromise := e.reader.RecoverPayload(batchNum, batchBlockHash, sequencerMsg)
						ctx := context.Background()
						result, err := delegatePromise.Await(ctx)
						if err != nil {
							promise.ProduceError(err)
						} else {
							promise.Produce(result)
						}
						return
					}
				}

				// Extract data hash (SHA256) from certificate
				dataHash := cert.DataHash

				e.mu.RLock()
				if evilData, exists := e.evilMappings[dataHash]; exists {
					e.mu.RUnlock()

					log.Info("EvilDAProvider returning evil data",
						"dataHash", common.Hash(dataHash).Hex(),
						"evilDataSize", len(evilData))

					promise.Produce(daprovider.PayloadResult{Payload: evilData})
					return
				}
				e.mu.RUnlock()
			}
		}

		// Fall back to underlying reader for non-evil certificates
		delegatePromise := e.reader.RecoverPayload(batchNum, batchBlockHash, sequencerMsg)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		if err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(result)
		}
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
		// Check if this is a CustomDA message and extract certificate
		if len(sequencerMsg) > 40 && daprovider.IsDACertificateMessageHeaderByte(sequencerMsg[40]) {
			certificate := sequencerMsg[40:]

			// Check if we're supposed to claim this certificate is invalid
			certKeccak := crypto.Keccak256Hash(certificate)
			e.mu.RLock()
			shouldClaimInvalid := e.invalidClaimCerts[certKeccak]
			e.mu.RUnlock()

			if shouldClaimInvalid {
				// For invalid certificates, we still return empty preimages (no error)
				// This matches the behavior where validation fails but preimages aren't needed
				promise.Produce(daprovider.PreimagesResult{Preimages: make(daprovider.PreimagesMap)})
				return
			}

			// Try to deserialize certificate
			cert, err := referenceda.Deserialize(certificate)
			if err == nil {
				// Check if this certificate is from our untrusted signer
				signer, signerErr := cert.RecoverSigner()
				if signerErr == nil {
					untrustedAddr := e.GetUntrustedSignerAddress()

					// If this cert was signed by our known untrusted signer, delegate to reader
					if untrustedAddr != nil && signer == *untrustedAddr {
						// Delegate to underlying reader which will get the data from storage
						delegatePromise := e.reader.CollectPreimages(batchNum, batchBlockHash, sequencerMsg)
						ctx := context.Background()
						result, err := delegatePromise.Await(ctx)
						if err != nil {
							promise.ProduceError(err)
						} else {
							promise.Produce(result)
						}
						return
					}
				}

				// Extract data hash (SHA256) from certificate
				dataHash := cert.DataHash

				e.mu.RLock()
				if evilData, exists := e.evilMappings[dataHash]; exists {
					e.mu.RUnlock()

					// Record preimages with evil data
					preimages := make(daprovider.PreimagesMap)
					preimageRecorder := daprovider.RecordPreimagesTo(preimages)
					// Use keccak256 of certificate for preimage recording
					preimageRecorder(certKeccak, evilData, arbutil.DACertificatePreimageType)

					promise.Produce(daprovider.PreimagesResult{Preimages: preimages})
					return
				}
				e.mu.RUnlock()
			}
		}

		// Fall back to underlying reader for non-evil certificates
		delegatePromise := e.reader.CollectPreimages(batchNum, batchBlockHash, sequencerMsg)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		if err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(result)
		}
	}()
	return &promise
}

// GenerateReadPreimageProof generates proof for evil data if configured, otherwise delegates
func (e *EvilDAProvider) GenerateReadPreimageProof(
	certHash common.Hash,
	offset uint64,
	certificate []byte,
) containers.PromiseInterface[daprovider.PreimageProofResult] {
	promise := containers.NewPromise[daprovider.PreimageProofResult](nil)
	go func() {
		// Try to deserialize certificate to check for evil mapping
		cert, err := referenceda.Deserialize(certificate)
		if err == nil {
			// Extract data hash (SHA256) from certificate
			dataHash := cert.DataHash

			e.mu.RLock()
			evilData, hasEvil := e.evilMappings[dataHash]
			e.mu.RUnlock()

			if hasEvil {
				// Generate proof with evil data
				// Format: [Version(1), CertificateSize(8), Certificate, PreimageSize(8), PreimageData]
				certLen := len(certificate)
				proof := make([]byte, 1+8+certLen+8+len(evilData))
				proof[0] = 1 // Version
				binary.BigEndian.PutUint64(proof[1:9], uint64(certLen))
				copy(proof[9:9+certLen], certificate)
				binary.BigEndian.PutUint64(proof[9+certLen:9+certLen+8], uint64(len(evilData)))
				copy(proof[9+certLen+8:], evilData)

				log.Debug("EvilDAProvider generating evil proof",
					"certHash", certHash.Hex(),
					"dataHash", common.Hash(dataHash).Hex(),
					"evilDataSize", len(evilData))

				promise.Produce(daprovider.PreimageProofResult{Proof: proof})
				return
			}
		}

		// No evil mapping, delegate to underlying validator
		delegatePromise := e.validator.GenerateReadPreimageProof(certHash, offset, certificate)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		if err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(result)
		}
	}()
	return &promise
}

// GenerateCertificateValidityProof generates a proof of certificate validity
func (e *EvilDAProvider) GenerateCertificateValidityProof(certificate []byte) containers.PromiseInterface[daprovider.ValidityProofResult] {
	promise := containers.NewPromise[daprovider.ValidityProofResult](nil)
	go func() {
		// Check if we should lie about this certificate
		cert, err := referenceda.Deserialize(certificate)
		if err == nil {
			signer, err := cert.RecoverSigner()
			if err == nil {
				untrustedAddr := e.GetUntrustedSignerAddress()

				// If this cert was signed by our known untrusted signer, lie and say it's valid
				if untrustedAddr != nil && signer == *untrustedAddr {
					log.Info("EvilDAProvider lying about untrusted certificate validity",
						"signer", signer.Hex(),
						"dataHash", common.Hash(cert.DataHash).Hex())
					promise.Produce(daprovider.ValidityProofResult{Proof: []byte{1, 0x01}}) // EVIL: claim valid when it's not
					return
				}
			}

			// Check if we should claim this specific valid cert is invalid
			certKeccak := crypto.Keccak256Hash(certificate)
			e.mu.RLock()
			shouldClaimInvalid := e.invalidClaimCerts[certKeccak]
			e.mu.RUnlock()

			if shouldClaimInvalid {
				log.Info("EvilDAProvider lying about valid certificate (claiming invalid)",
					"certKeccak", certKeccak.Hex(),
					"dataHash", common.Hash(cert.DataHash).Hex())
				promise.Produce(daprovider.ValidityProofResult{Proof: []byte{0, 0x01}}) // EVIL: claim invalid when it's valid
				return
			}
		}

		// For all other cases, delegate to underlying validator
		delegatePromise := e.validator.GenerateCertificateValidityProof(certificate)
		ctx := context.Background()
		result, err := delegatePromise.Await(ctx)
		if err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(result)
		}
	}()
	return &promise
}

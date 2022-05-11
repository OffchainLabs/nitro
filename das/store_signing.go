// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbstate"
)

var uniquifyingPrefix = []byte("Arbitrum Nitro DAS API Store:")

type DasSigner func([]byte) ([]byte, error) // takes 32-byte array (hash of data) and produces signature bytes (and/or error)

func DasSignerFromPrivateKey(privateKey *ecdsa.PrivateKey) DasSigner {
	return func(data []byte) ([]byte, error) {
		return crypto.Sign(data, privateKey)
	}
}

func applyDasSigner(signer DasSigner, data []byte, timeout uint64) ([]byte, error) {
	return signer(dasStoreHash(data, timeout))
}

func DasRecoverSigner(data []byte, timeout uint64, sig []byte) (common.Address, error) {
	pk, err := crypto.SigToPub(dasStoreHash(data, timeout), sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pk), nil
}

func dasStoreHash(data []byte, timeout uint64) []byte {
	var buf8 [8]byte
	binary.BigEndian.PutUint64(buf8[:], timeout)
	return crypto.Keccak256(uniquifyingPrefix, buf8[:], data)
}

type StoreSigningDAS struct {
	DataAvailabilityService
	signer DasSigner
}

func NewStoreSigningDAS(inner DataAvailabilityService, signer DasSigner) DataAvailabilityService {
	return &StoreSigningDAS{inner, signer}
}

func (s *StoreSigningDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	mySig, err := applyDasSigner(s.signer, message, timeout)
	if err != nil {
		return nil, err
	}
	return s.DataAvailabilityService.Store(ctx, message, timeout, mySig)
}

func (s *StoreSigningDAS) String() string {
	addrStr := "[error]"
	addr, err := s.SignerAddress()
	if err == nil {
		addrStr = addr.Hex()
	}
	return "StoreSigningDAS (" + addrStr + " ," + s.DataAvailabilityService.String() + ")"
}

func (s *StoreSigningDAS) SignerAddress() (common.Address, error) {
	sig, err := applyDasSigner(s.signer, []byte{}, 0)
	if err != nil {
		return common.Address{}, err
	}
	return DasRecoverSigner([]byte{}, 0, sig)
}

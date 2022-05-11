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

func DasSignStore(data []byte, timeout uint64, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return crypto.Sign(dasStoreHash(data, timeout), privateKey)
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
	inner      DataAvailabilityService
	privateKey *ecdsa.PrivateKey
}

func NewStoreSigningDAS(inner DataAvailabilityService, privateKey *ecdsa.PrivateKey) DataAvailabilityService {
	return &StoreSigningDAS{inner, privateKey}
}

func (s *StoreSigningDAS) Retrieve(ctx context.Context, cert *arbstate.DataAvailabilityCertificate) ([]byte, error) {
	return s.inner.Retrieve(ctx, cert)
}

func (s *StoreSigningDAS) KeysetFromHash(ctx context.Context, ksHash []byte) ([]byte, error) {
	return s.inner.KeysetFromHash(ctx, ksHash)
}

func (s *StoreSigningDAS) CurrentKeysetBytes(ctx context.Context) ([]byte, error) {
	return s.inner.CurrentKeysetBytes(ctx)
}

func (s *StoreSigningDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	mySig, err := DasSignStore(message, timeout, s.privateKey)
	if err != nil {
		return nil, err
	}
	return s.inner.Store(ctx, message, timeout, mySig)
}

func (s *StoreSigningDAS) String() string {
	return "StoreSigningDAS (" + s.SignerAddress().Hex() + " ," + s.inner.String() + ")"
}

func (s *StoreSigningDAS) SignerAddress() common.Address {
	publicKey := s.privateKey.Public()
	return crypto.PubkeyToAddress(*publicKey.(*ecdsa.PublicKey))
}

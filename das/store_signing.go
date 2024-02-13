// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/signature"
)

var uniquifyingPrefix = []byte("Arbitrum Nitro DAS API Store:")

func applyDasSigner(signer signature.DataSignerFunc, data []byte, timeout uint64) ([]byte, error) {
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
	return dastree.HashBytes(uniquifyingPrefix, buf8[:], data)
}

type StoreSigningDAS struct {
	DataAvailabilityServiceWriter
	signer signature.DataSignerFunc
	addr   common.Address
}

func NewStoreSigningDAS(inner DataAvailabilityServiceWriter, signer signature.DataSignerFunc) (DataAvailabilityServiceWriter, error) {
	sig, err := applyDasSigner(signer, []byte{}, 0)
	if err != nil {
		return nil, err
	}
	addr, err := DasRecoverSigner([]byte{}, 0, sig)
	if err != nil {
		return nil, err
	}
	return &StoreSigningDAS{inner, signer, addr}, nil
}

func (s *StoreSigningDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	log.Trace("das.StoreSigningDAS.Store(...)", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "sig", pretty.FirstFewBytes(sig), "this", s)
	mySig, err := applyDasSigner(s.signer, message, timeout)
	if err != nil {
		return nil, err
	}
	return s.DataAvailabilityServiceWriter.Store(ctx, message, timeout, mySig)
}

func (s *StoreSigningDAS) String() string {
	return "StoreSigningDAS (" + s.SignerAddress().Hex() + " ," + s.DataAvailabilityServiceWriter.String() + ")"
}

func (s *StoreSigningDAS) SignerAddress() common.Address {
	return s.addr
}

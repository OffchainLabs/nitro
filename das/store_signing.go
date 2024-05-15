// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/signature"
)

var uniquifyingPrefix = []byte("Arbitrum Nitro DAS API Store:")

func applyDasSigner(signer signature.DataSignerFunc, data []byte, extraFields ...uint64) ([]byte, error) {
	return signer(dasStoreHash(data, extraFields...))
}

func DasRecoverSigner(data []byte, sig []byte, extraFields ...uint64) (common.Address, error) {
	pk, err := crypto.SigToPub(dasStoreHash(data, extraFields...), sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pk), nil
}

func dasStoreHash(data []byte, extraFields ...uint64) []byte {
	var buf []byte

	for _, field := range extraFields {
		buf = binary.BigEndian.AppendUint64(buf, field)
	}

	return dastree.HashBytes(uniquifyingPrefix, buf, data)
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
	addr, err := DasRecoverSigner([]byte{}, sig, 0)
	if err != nil {
		return nil, err
	}
	return &StoreSigningDAS{inner, signer, addr}, nil
}

func (s *StoreSigningDAS) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*daprovider.DataAvailabilityCertificate, error) {
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

// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/signature"
)

// Writer implements the daprovider.Writer interface for ReferenceDA
type Writer struct {
	storage *InMemoryStorage
	signer  signature.DataSignerFunc
}

// NewWriter creates a new ReferenceDA writer
func NewWriter(signer signature.DataSignerFunc) *Writer {
	return &Writer{
		storage: GetInMemoryStorage(),
		signer:  signer,
	}
}

func (w *Writer) String() string {
	return fmt.Sprintf("Writer{%v}", w.storage)
}

func (w *Writer) Store(
	ctx context.Context,
	message []byte,
	timeout uint64,
) ([]byte, error) {
	if w.signer == nil {
		return nil, fmt.Errorf("no signer configured")
	}

	// Create and sign certificate
	cert, err := NewCertificate(message, w.signer)
	if err != nil {
		return nil, err
	}

	// Store the message in the singleton storage
	err = w.storage.Store(ctx, message)
	if err != nil {
		return nil, err
	}

	// Serialize certificate for on-chain storage
	certificate := cert.Serialize()
	hashKey := common.BytesToHash(cert.DataHash[:])

	log.Debug("ReferenceDA batch stored with signature",
		"sha256", hashKey.Hex(),
		"certificateSize", len(certificate),
		"batchSize", len(message),
	)

	return certificate, nil
}

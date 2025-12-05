// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/signature"
)

// Writer implements the daprovider.Writer interface for ReferenceDA
type Writer struct {
	storage        *InMemoryStorage
	signer         signature.DataSignerFunc
	maxMessageSize int
}

// NewWriter creates a new ReferenceDA writer
func NewWriter(signer signature.DataSignerFunc, maxMessageSize int) *Writer {
	return &Writer{
		storage:        GetInMemoryStorage(),
		signer:         signer,
		maxMessageSize: maxMessageSize,
	}
}

func (w *Writer) Store(
	message []byte,
	timeout uint64,
) containers.PromiseInterface[[]byte] {
	certificate, err := w.store(message)
	return containers.NewReadyPromise(certificate, err)
}

func (w *Writer) GetMaxMessageSize() containers.PromiseInterface[int] {
	return containers.NewReadyPromise(w.maxMessageSize, nil)
}

func (w *Writer) store(
	message []byte,
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
	err = w.storage.Store(message)
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

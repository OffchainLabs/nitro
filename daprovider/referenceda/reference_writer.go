// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/daprovider"
)

// Writer implements the daprovider.Writer interface for ReferenceDA
type Writer struct {
	storage *InMemoryStorage
}

// NewWriter creates a new ReferenceDA writer
func NewWriter() *Writer {
	return &Writer{
		storage: GetInMemoryStorage(),
	}
}

func (w *Writer) Store(
	ctx context.Context,
	message []byte,
	timeout uint64,
	disableFallbackStoreDataOnChain bool,
) ([]byte, error) {
	// Calculate SHA256 hash of the message
	sha256Hash := sha256.Sum256(message)
	hashKey := common.BytesToHash(sha256Hash[:])

	// Store the message in the singleton storage
	err := w.storage.Store(ctx, message)
	if err != nil {
		return nil, err
	}

	// Create certificate for the batch (this is what gets stored on-chain)
	// Format: 1 byte header (CustomDAMessageHeaderFlag) + 32 bytes SHA256 hash
	certificate := make([]byte, 1+32)
	certificate[0] = daprovider.CustomDAMessageHeaderFlag
	copy(certificate[1:33], sha256Hash[:])

	log.Debug("ReferenceDA batch stored",
		"sha256", hashKey.Hex(),
		"certificateSize", len(certificate),
		"batchSize", len(message),
	)

	return certificate, nil
}

// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package customda

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/daprovider"
)

// Writer implements the daprovider.Writer interface for CustomDA
type Writer struct {
	mu            sync.Mutex
	storedBatches map[common.Hash][]byte
	validator     daprovider.Validator
}

// NewWriter creates a new CustomDA writer
func NewWriter(validator daprovider.Validator) *Writer {
	return &Writer{
		storedBatches: make(map[common.Hash][]byte),
		validator:     validator,
	}
}

// Store processes the message data and returns a reference to retrieve it
// in the CustomDA system
func (w *Writer) Store(
	ctx context.Context,
	message []byte,
	timeout uint64,
	disableFallbackStoreDataOnChain bool,
) ([]byte, error) {
	// First byte of the message should be CustomDAMessageHeaderFlag
	// which should have been set by the batch_poster
	if len(message) == 0 || message[0] != daprovider.CustomDAMessageHeaderFlag {
		log.Warn("Message doesn't have CustomDA header flag, will use original message")
		return message, nil
	}

	// Extract the actual batch data (removing the header byte)
	batchData := message[1:]

	// Generate a unique nonce for this batch
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Hash the batch data to get a reference
	keccak256Hash := crypto.Keccak256Hash(batchData)
	sha256Hash := sha256.Sum256(batchData)

	// Record any preimages from the batch if a validator is provided
	if w.validator != nil {
		preimages, err := w.validator.RecordPreimages(ctx, batchData)
		if err != nil {
			log.Warn("Failed to record preimages", "err", err)
			// Continue despite error, as this shouldn't block batch posting
		}

		// Log the preimages that were recorded
		for _, p := range preimages {
			log.Debug("Recorded preimage", "type", p.PreimageType, "hash", p.Hash.Hex(), "size", len(p.Data))
		}
	}

	// Store the batch data for later retrieval
	w.mu.Lock()
	w.storedBatches[keccak256Hash] = batchData
	w.mu.Unlock()

	// Create certificate for the batch (this is what gets stored on-chain)
	// Format:
	// - 1 byte header (CustomDAMessageHeaderFlag)
	// - 32 bytes Keccak256 hash
	// - 32 bytes SHA256 hash (for additional verification)
	// - 16 bytes nonce
	// - 4 bytes length of batch data in big-endian
	certificate := make([]byte, 1+32+32+16+4)
	certificate[0] = daprovider.CustomDAMessageHeaderFlag
	copy(certificate[1:33], keccak256Hash[:])
	copy(certificate[33:65], sha256Hash[:])
	copy(certificate[65:81], nonce)
	binary.BigEndian.PutUint32(certificate[81:85], uint32(len(batchData)))

	log.Info("CustomDA batch stored",
		"keccak256", keccak256Hash.Hex(),
		"sha256", common.BytesToHash(sha256Hash[:]).Hex(),
		"certificateSize", len(certificate),
		"batchSize", len(batchData),
	)

	return certificate, nil
}

// GetBatch retrieves a batch by its hash
func (w *Writer) GetBatch(hash common.Hash) ([]byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	batch, exists := w.storedBatches[hash]
	return batch, exists
}

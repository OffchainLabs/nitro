// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

// storageBytes defines the interface for ArbOS storage.
type storageBytes interface {
	Get() ([]byte, error)
	Set(val []byte) error
}

// StorageResourceConstraints defines a storage-backed ResourceConstraints.
type StorageResourceConstraints struct {
	storage storageBytes
}

// NewStorageResourceConstraints creates a new storage-backed ResourceConstraints.
func NewStorageResourceConstraints(storage storageBytes) *StorageResourceConstraints {
	return &StorageResourceConstraints{
		storage: storage,
	}
}

type resourceConstraintRLP struct {
	Resources    []ResourceWeight
	Period       PeriodSecs
	TargetPerSec uint64
	Backlog      uint64
}

// EncodeRLP encodes ResourceConstraint deterministically,
// ensuring the fixed-length weights array is preserved.
func (c *ResourceConstraint) EncodeRLP(w io.Writer) error {
	weights := make([]ResourceWeight, len(c.Resources.weights))
	copy(weights, c.Resources.weights[:])
	return rlp.Encode(w, resourceConstraintRLP{
		Resources:    weights,
		Period:       c.Period,
		TargetPerSec: c.TargetPerSec,
		Backlog:      c.Backlog,
	})
}

// DecodeRLP decodes ResourceConstraint deterministically,
// padding or truncating the weights slice to the correct array length.
func (c *ResourceConstraint) DecodeRLP(s *rlp.Stream) error {
	var raw resourceConstraintRLP
	if err := s.Decode(&raw); err != nil {
		return err
	}
	c.Period = raw.Period
	c.TargetPerSec = raw.TargetPerSec
	c.Backlog = raw.Backlog

	for i := range c.Resources.weights {
		if i < len(raw.Resources) {
			c.Resources.weights[i] = raw.Resources[i]
		} else {
			c.Resources.weights[i] = 0
		}
	}
	return nil
}

// Load decodes ResourceConstraints from storage using RLP.
// If storage is empty, returns an empty ResourceConstraints.
func (src *StorageResourceConstraints) Load() (*ResourceConstraints, error) {
	data, err := src.storage.Get()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return NewResourceConstraints(), nil
	}

	var list []*ResourceConstraint
	if err := rlp.DecodeBytes(data, &list); err != nil {
		return nil, err
	}

	rc := NewResourceConstraints()
	for _, c := range list {
		rc.Set(c.Resources, c.Period, c.TargetPerSec)
		ptr := rc.Get(c.Resources, c.Period)
		ptr.Backlog = c.Backlog
	}

	return rc, nil
}

// Write encodes ResourceConstraints into storage using RLP.
func (src *StorageResourceConstraints) Write(rc *ResourceConstraints) error {
	var list []*ResourceConstraint
	for c := range rc.All() {
		list = append(list, c)
	}

	// If there are no constraints, clear the storage instead of writing 0xC0
	if len(list) == 0 {
		return src.storage.Set(nil)
	}

	data, err := rlp.EncodeToBytes(list)
	if err != nil {
		return err
	}
	return src.storage.Set(data)
}

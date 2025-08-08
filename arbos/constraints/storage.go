// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

type storageBytes interface {
	Get() ([]byte, error)
	Set(val []byte) error
}

// ResourceConstraintsStorage stores the resources constraints in the ArbOS storage as bytes.
// When updating the storage, the code will read the bytes, deserialize them, make the changes,
// serialize the struct, and write it back to storage.
type ResourceConstraintsStorage struct {
	bytes storageBytes
}

// Open returns a struct that manages the storage for the resource constraints.
// This function receives a storageBytes to facilitate unit testing.
func Open(bytes storageBytes) *ResourceConstraintsStorage {
	return &ResourceConstraintsStorage{
		bytes: bytes,
	}
}

// SetConstraint adds or updates the given resource constraint.
func (sto *ResourceConstraintsStorage) SetConstraint(resourceId uint8, periodSecs uint32, targetPerPeriod uint64) error {
	resource, err := multigas.CheckResourceKind(resourceId)
	if err != nil {
		return err
	}
	constraints, err := sto.load()
	if err != nil {
		return err
	}
	constraints.SetConstraint(resource, PeriodSecs(periodSecs), targetPerPeriod)
	return sto.store(constraints)
}

// ClearConstraint removes the given resource constraint.
func (sto *ResourceConstraintsStorage) ClearConstraint(resourceId uint8, periodSecs uint32) error {
	resource, err := multigas.CheckResourceKind(resourceId)
	if err != nil {
		return err
	}
	constraints, err := sto.load()
	if err != nil {
		return err
	}
	constraints.ClearConstraint(resource, PeriodSecs(periodSecs))
	return sto.store(constraints)
}

func (sto *ResourceConstraintsStorage) store(constraints ResourceConstraints) error {
	bytes, err := json.Marshal(constraints)
	if err != nil {
		return fmt.Errorf("failed to marshal resource constraints: %w", err)
	}
	err = sto.bytes.Set(bytes)
	if err != nil {
		return fmt.Errorf("failed to set resource constraints: %w", err)
	}
	return nil
}

func (sto *ResourceConstraintsStorage) load() (ResourceConstraints, error) {
	bytes, err := sto.bytes.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get resources constraints: %w", err)
	}
	constraints := NewResourceConstraints()
	if len(bytes) == 0 {
		return constraints, nil
	}
	err = json.Unmarshal(bytes, &constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources constraints: %w", err)
	}
	return constraints, nil
}

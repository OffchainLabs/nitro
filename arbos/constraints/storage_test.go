// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package constraints

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

type storageBytesMock struct {
	bytes []byte
}

func (sto *storageBytesMock) Get() ([]byte, error) {
	return sto.bytes, nil
}

func (sto *storageBytesMock) Set(val []byte) error {
	sto.bytes = val
	return nil
}

func TestStorageSetConstraint(t *testing.T) {
	mock := &storageBytesMock{bytes: nil}
	storage := Open(mock)
	if err := storage.SetConstraint(1, 10, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.SetConstraint(1, 20, 800); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.SetConstraint(2, 60, 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStorage := `{"1":{` +
		`"10":{"period":10000000000,"target":50},` +
		`"20":{"period":20000000000,"target":40}},"2":{` +
		`"60":{"period":60000000000,"target":0}},"3":{},"4":{}}`
	if string(mock.bytes) != wantStorage {
		t.Errorf("wrong resource constraint storage: got %v, want %v", string(mock.bytes), wantStorage)
	}
}

func TestStorageClearConstraint(t *testing.T) {
	initialStorage := `{"1":{` +
		`"10":{"period":10000000000,"target":50},` +
		`"20":{"period":20000000000,"target":40}},"2":{` +
		`"60":{"period":60000000000,"target":0}},"3":{},"4":{}}`
	mock := &storageBytesMock{bytes: []byte(initialStorage)}
	storage := Open(mock)
	if err := storage.ClearConstraint(1, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.ClearConstraint(1, 20); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := storage.ClearConstraint(2, 60); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStorage := `{"1":{},"2":{},"3":{},"4":{}}`
	if string(mock.bytes) != wantStorage {
		t.Errorf("wrong resource constraint storage: got %v, want %v", string(mock.bytes), wantStorage)
	}
}

func TestStorageStore(t *testing.T) {
	for _, tc := range []struct {
		name        string
		constraints []resourceConstraintDescription
		wantStorage string
	}{
		{
			name:        "EmptyConstraints",
			constraints: []resourceConstraintDescription{},
			wantStorage: `{"1":{},"2":{},"3":{},"4":{}}`,
		},
		{
			name: "OneConstraint",
			constraints: []resourceConstraintDescription{
				{multigas.ResourceKindComputation, 100, 33000},
			},
			wantStorage: `{"1":{"100":{"period":100000000000,"target":330}},"2":{},"3":{},"4":{}}`,
		},
		{
			name: "MultipleConstraints",
			constraints: []resourceConstraintDescription{
				{multigas.ResourceKindComputation, 100, 33000},
				{multigas.ResourceKindComputation, 200, 44000},
				{multigas.ResourceKindHistoryGrowth, 300, 55000},
			},
			wantStorage: `{"1":{` +
				`"100":{"period":100000000000,"target":330},` +
				`"200":{"period":200000000000,"target":220}},"2":{` +
				`"300":{"period":300000000000,"target":183}},"3":{},"4":{}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := &storageBytesMock{bytes: nil}
			constraints := NewResourceConstraints()
			for _, constraint := range tc.constraints {
				constraints.SetConstraint(constraint.resource, constraint.periodSecs, constraint.targetPerPeriod)
			}
			err := Open(mock).store(constraints)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(mock.bytes) != tc.wantStorage {
				t.Errorf("wrong storage: got %v, want %v", string(mock.bytes), tc.wantStorage)
			}
		})
	}
}

func TestStorageLoad(t *testing.T) {
	for _, tc := range []struct {
		name            string
		storage         string
		wantConstraints []resourceConstraintDescription
	}{
		{
			name:            "ZeroBytes",
			storage:         "",
			wantConstraints: []resourceConstraintDescription{},
		},
		{
			name:            "NoResources",
			storage:         "{}",
			wantConstraints: []resourceConstraintDescription{},
		},
		{
			name:            "EmptyConstraints",
			storage:         `{"1":{},"2":{},"3":{},"4":{}}`,
			wantConstraints: []resourceConstraintDescription{},
		},
		{
			name:    "OneConstraint",
			storage: `{"1":{"100":{"period":100000000000,"target":330}},"2":{},"3":{},"4":{}}`,
			wantConstraints: []resourceConstraintDescription{
				{multigas.ResourceKindComputation, 100, 33000},
			},
		},
		{
			name: "MultipleConstraints",
			storage: `{"1":{` +
				`"100":{"period":10000000000,"target":220},` +
				`"200":{"period":20000000000,"target":330}},"2":{` +
				`"300":{"period":30000000000,"target":440}},"3":{},"4":{}}`,
			wantConstraints: []resourceConstraintDescription{
				{multigas.ResourceKindComputation, 100, 22000},
				{multigas.ResourceKindComputation, 200, 66000},
				{multigas.ResourceKindHistoryGrowth, 300, 132000},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := &storageBytesMock{bytes: []byte(tc.storage)}
			constraints, err := Open(mock).load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := constraints.getConstraints(); !reflect.DeepEqual(got, tc.wantConstraints) {
				t.Errorf("wrong resource constraints: got %v, want %v", got, tc.wantConstraints)
			}
		})
	}
}

// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package staker

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

type testBOLDExecutionSpawner struct {
	moduleRoots []common.Hash
	err         error
}

func (s *testBOLDExecutionSpawner) Start(context.Context) error { return nil }

func (s *testBOLDExecutionSpawner) Stop() {}

func (s *testBOLDExecutionSpawner) WasmModuleRoots() ([]common.Hash, error) {
	return s.moduleRoots, s.err
}

func (s *testBOLDExecutionSpawner) GetMachineHashesWithStepSize(context.Context, common.Hash, *validator.ValidationInput, uint64, uint64, uint64) ([]common.Hash, error) {
	return nil, nil
}

func (s *testBOLDExecutionSpawner) GetProofAt(context.Context, common.Hash, *validator.ValidationInput, uint64) ([]byte, error) {
	return nil, nil
}

func TestBOLDExecutionSpawnerForModuleRootSelectsMatchingSpawner(t *testing.T) {
	targetRoot := common.HexToHash("0x2")
	first := &testBOLDExecutionSpawner{
		moduleRoots: []common.Hash{common.HexToHash("0x1")},
	}
	second := &testBOLDExecutionSpawner{
		moduleRoots: []common.Hash{targetRoot},
	}
	validator := &StatelessBlockValidator{
		boldExecSpawners: []validator.BOLDExecutionSpawner{first, second},
	}

	spawner, err := validator.BOLDExecutionSpawnerForModuleRoot(targetRoot)
	if err != nil {
		t.Fatalf("BOLDExecutionSpawnerForModuleRoot() error = %v", err)
	}
	if spawner != second {
		t.Fatalf("BOLDExecutionSpawnerForModuleRoot() returned %p, want %p", spawner, second)
	}
}

func TestBOLDExecutionSpawnerForModuleRootReturnsErrorWhenUnsupported(t *testing.T) {
	targetRoot := common.HexToHash("0x3")
	validator := &StatelessBlockValidator{
		boldExecSpawners: []validator.BOLDExecutionSpawner{
			&testBOLDExecutionSpawner{
				moduleRoots: []common.Hash{common.HexToHash("0x1")},
			},
			&testBOLDExecutionSpawner{
				err: errors.New("unavailable"),
			},
		},
	}

	_, err := validator.BOLDExecutionSpawnerForModuleRoot(targetRoot)
	if err == nil {
		t.Fatal("BOLDExecutionSpawnerForModuleRoot() error = nil, want non-nil")
	}
}

// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/api"
	"github.com/offchainlabs/nitro/bold/api/db"
)

// Mock implementations for testing
type mockDatabase struct {
	collectMachineHashes []*api.JsonCollectMachineHashes
	shouldError          bool
}

func (m *mockDatabase) GetCollectMachineHashes(opts ...db.CollectMachineHashesOption) ([]*api.JsonCollectMachineHashes, error) {
	if m.shouldError {
		return nil, &mockError{message: "database error"}
	}
	return m.collectMachineHashes, nil
}

// Implement other required methods with empty implementations
func (m *mockDatabase) GetAssertions(opts ...db.AssertionOption) ([]*api.JsonAssertion, error) {
	return nil, nil
}
func (m *mockDatabase) GetChallengedAssertions(opts ...db.AssertionOption) ([]*api.JsonAssertion, error) {
	return nil, nil
}
func (m *mockDatabase) GetEdges(opts ...db.EdgeOption) ([]*api.JsonEdge, error)        { return nil, nil }
func (m *mockDatabase) UpdateAssertions(assertions []*api.JsonAssertion) error         { return nil }
func (m *mockDatabase) UpdateEdges(edges []*api.JsonEdge) error                        { return nil }
func (m *mockDatabase) InsertCollectMachineHash(h *api.JsonCollectMachineHashes) error { return nil }
func (m *mockDatabase) UpdateCollectMachineHash(h *api.JsonCollectMachineHashes) error { return nil }

type mockError struct {
	message string
}

func (m *mockError) Error() string {
	return m.message
}

func TestBackend_GetCollectMachineHashes(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		ctx := context.Background()
		mockDB := &mockDatabase{
			collectMachineHashes: []*api.JsonCollectMachineHashes{
				{
					WasmModuleRoot:    common.BytesToHash([]byte("test")),
					RawStepHeights:    "1,2,3",
					NumDesiredHashes:  3,
					MachineStartIndex: 0,
					StepSize:          1,
				},
			},
		}

		backend := &Backend{
			db: mockDB,
		}

		result, err := backend.GetCollectMachineHashes(ctx)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Len(t, result[0].StepHeights, 3)
		require.Equal(t, uint64(1), result[0].StepHeights[0])
		require.Equal(t, uint64(2), result[0].StepHeights[1])
		require.Equal(t, uint64(3), result[0].StepHeights[2])
	})

	t.Run("database error handling", func(t *testing.T) {
		ctx := context.Background()
		mockDB := &mockDatabase{
			shouldError: true,
		}

		backend := &Backend{
			db: mockDB,
		}

		result, err := backend.GetCollectMachineHashes(ctx)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "database error")
	})

	t.Run("empty step heights", func(t *testing.T) {
		ctx := context.Background()
		mockDB := &mockDatabase{
			collectMachineHashes: []*api.JsonCollectMachineHashes{
				{
					WasmModuleRoot:    common.BytesToHash([]byte("test")),
					RawStepHeights:    "",
					NumDesiredHashes:  0,
					MachineStartIndex: 0,
					StepSize:          1,
				},
			},
		}

		backend := &Backend{
			db: mockDB,
		}

		result, err := backend.GetCollectMachineHashes(ctx)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Nil(t, result[0].StepHeights)
	})

	t.Run("step heights with empty values", func(t *testing.T) {
		ctx := context.Background()
		mockDB := &mockDatabase{
			collectMachineHashes: []*api.JsonCollectMachineHashes{
				{
					WasmModuleRoot:    common.BytesToHash([]byte("test")),
					RawStepHeights:    "1,,3,",
					NumDesiredHashes:  3,
					MachineStartIndex: 0,
					StepSize:          1,
				},
			},
		}

		backend := &Backend{
			db: mockDB,
		}

		result, err := backend.GetCollectMachineHashes(ctx)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Len(t, result[0].StepHeights, 2)
		require.Equal(t, uint64(1), result[0].StepHeights[0])
		require.Equal(t, uint64(3), result[0].StepHeights[1])
	})

	t.Run("invalid step height parsing", func(t *testing.T) {
		ctx := context.Background()
		mockDB := &mockDatabase{
			collectMachineHashes: []*api.JsonCollectMachineHashes{
				{
					WasmModuleRoot:    common.BytesToHash([]byte("test")),
					RawStepHeights:    "1,invalid,3",
					NumDesiredHashes:  3,
					MachineStartIndex: 0,
					StepSize:          1,
				},
			},
		}

		backend := &Backend{
			db: mockDB,
		}

		result, err := backend.GetCollectMachineHashes(ctx)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "could not parse step height invalid")
	})
}

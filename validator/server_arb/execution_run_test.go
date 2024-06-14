package server_arb

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

var _ GlobalStateGetter = (*mockMachine)(nil)

type mockMachine struct {
	gs         validator.GoGlobalState
	totalSteps uint64
}

func (m *mockMachine) Hash() common.Hash {
	if m.gs.PosInBatch == m.totalSteps-1 {
		return machineFinishedHash(m.gs)
	}
	return m.gs.Hash()
}

func (m *mockMachine) GetGlobalState() validator.GoGlobalState {
	return m.gs
}

func (m *mockMachine) Step(ctx context.Context, stepSize uint64) error {
	for i := uint64(0); i < stepSize; i++ {
		if m.gs.PosInBatch == m.totalSteps-1 {
			return nil
		}
		m.gs.PosInBatch += 1
	}
	return nil
}

func Test_machineHashesWithStep(t *testing.T) {
	mm := &mockMachine{}
	e := &executionRun{}
	ctx := context.Background()

	machGetter := func(ctx context.Context, index uint64) (GlobalStateGetter, error) {
		return mm, nil
	}
	t.Run("basic argument checks", func(t *testing.T) {
		_, err := e.machineHashesWithStepSize(ctx, machineHashesWithStepSizeArgs{
			stepSize: 0,
		})
		if !strings.Contains(err.Error(), "step size cannot be 0") {
			t.Fatal("Wrong error")
		}
		_, err = e.machineHashesWithStepSize(ctx, machineHashesWithStepSizeArgs{
			stepSize:          1,
			requiredNumHashes: 0,
		})
		if !strings.Contains(err.Error(), "required number of hashes cannot be 0") {
			t.Fatal("Wrong error")
		}
	})
	t.Run("machine at start index 0 hash is the finished state hash", func(t *testing.T) {
		mm.gs = validator.GoGlobalState{
			Batch: 1,
		}
		hashes, err := e.machineHashesWithStepSize(ctx, machineHashesWithStepSizeArgs{
			fromBatch:         0,
			stepSize:          1,
			requiredNumHashes: 1,
			startIndex:        0,
			getMachineAtIndex: machGetter,
		})
		if err != nil {
			t.Fatal(err)
		}
		expected := machineFinishedHash(mm.gs)
		if len(hashes) != 1 {
			t.Fatal("Wanted one hash")
		}
		if expected != hashes[0] {
			t.Fatalf("Wanted %#x, got %#x", expected, hashes[0])
		}
	})
	t.Run("can step in step size increments and collect hashes", func(t *testing.T) {
		initialGs := validator.GoGlobalState{
			Batch:      1,
			PosInBatch: 0,
		}
		mm.gs = initialGs
		mm.totalSteps = 20
		stepSize := uint64(5)
		hashes, err := e.machineHashesWithStepSize(ctx, machineHashesWithStepSizeArgs{
			fromBatch:         1,
			stepSize:          stepSize,
			requiredNumHashes: 4,
			startIndex:        0,
			getMachineAtIndex: machGetter,
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedHashes := make([]common.Hash, 0)
		for i := uint64(0); i < 4; i++ {
			if i == 0 {
				expectedHashes = append(expectedHashes, machineFinishedHash(initialGs))
				continue
			}
			gs := validator.GoGlobalState{
				Batch:      1,
				PosInBatch: uint64(i * stepSize),
			}
			expectedHashes = append(expectedHashes, gs.Hash())
		}
		if len(hashes) != len(expectedHashes) {
			t.Fatal("Wanted one hash")
		}
		for i := range hashes {
			if expectedHashes[i] != hashes[i] {
				t.Fatalf("Wanted at index %d, %#x, got %#x", i, expectedHashes[i], hashes[i])
			}
		}
	})
	t.Run("if finishes execution early, simply pads the remaining desired hashes with the machine finished hash", func(t *testing.T) {
		initialGs := validator.GoGlobalState{
			Batch:      1,
			PosInBatch: 0,
		}
		mm.gs = initialGs
		mm.totalSteps = 20
		stepSize := uint64(5)
		hashes, err := e.machineHashesWithStepSize(ctx, machineHashesWithStepSizeArgs{
			fromBatch:         1,
			stepSize:          stepSize,
			requiredNumHashes: 10,
			startIndex:        0,
			getMachineAtIndex: machGetter,
		})
		if err != nil {
			t.Fatal(err)
		}
		expectedHashes := make([]common.Hash, 0)
		for i := uint64(0); i < 4; i++ {
			if i == 0 {
				expectedHashes = append(expectedHashes, machineFinishedHash(initialGs))
				continue
			}
			gs := validator.GoGlobalState{
				Batch:      1,
				PosInBatch: uint64(i * stepSize),
			}
			expectedHashes = append(expectedHashes, gs.Hash())
		}
		// The rest of the expected hashes should be the machine finished hash repeated.
		for i := uint64(4); i < 10; i++ {
			expectedHashes = append(expectedHashes, machineFinishedHash(mm.gs))
		}
		if len(hashes) != len(expectedHashes) {
			t.Fatal("Wanted one hash")
		}
		for i := range hashes {
			if expectedHashes[i] != hashes[i] {
				t.Fatalf("Wanted at index %d, %#x, got %#x", i, expectedHashes[i], hashes[i])
			}
		}
	})
}
